package exec

import (
	"bytes"
	"context"
	"reflect"
	"strconv"
	"sync"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/exec/resolvable"
	"github.com/neelance/graphql-go/internal/exec/selected"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/log"
	"github.com/neelance/graphql-go/trace"
)

type Request struct {
	selected.Request
	Limiter chan struct{}
	Tracer  trace.Tracer
	Logger  log.Logger
}

type fieldResult struct {
	name  string
	value []byte
}

func (r *Request) handlePanic(ctx context.Context) {
	if value := recover(); value != nil {
		r.Logger.LogPanic(ctx, value)
		r.AddError(makePanicError(value))
	}
}

func makePanicError(value interface{}) *errors.QueryError {
	return errors.Errorf("graphql: panic occurred: %v", value)
}

func (r *Request) Execute(ctx context.Context, s *resolvable.Schema, op *query.Operation) ([]byte, []*errors.QueryError) {
	var out bytes.Buffer
	func() {
		defer r.handlePanic(ctx)
		sels := selected.ApplyOperation(&r.Request, s, op)
		r.execSelections(ctx, sels, s.Resolver, &out, op.Type == query.Mutation)
	}()

	if err := ctx.Err(); err != nil {
		return nil, []*errors.QueryError{errors.Errorf("%s", err)}
	}

	return out.Bytes(), r.Errs
}

type fieldWithResolver struct {
	field    *selected.SchemaField
	resolver reflect.Value
	out      bytes.Buffer
}

func (r *Request) execSelections(ctx context.Context, sels []selected.Selection, resolver reflect.Value, out *bytes.Buffer, serially bool) {
	async := !serially && selected.HasAsyncSel(sels)

	var fields []*fieldWithResolver
	collectFieldsToResolve(sels, resolver, &fields)

	if async {
		var wg sync.WaitGroup
		wg.Add(len(fields))
		for _, f := range fields {
			go func(f *fieldWithResolver) {
				defer r.handlePanic(ctx)
				r.execFieldSelection(ctx, f.field, f.resolver, &f.out, false)
				wg.Done()
			}(f)
		}
		wg.Wait()
	}

	out.WriteByte('{')
	for i, f := range fields {
		if i > 0 {
			out.WriteByte(',')
		}
		out.WriteByte('"')
		out.WriteString(f.field.Alias)
		out.WriteByte('"')
		out.WriteByte(':')
		if async {
			out.Write(f.out.Bytes())
			continue
		}
		r.execFieldSelection(ctx, f.field, f.resolver, out, false)
	}
	out.WriteByte('}')
}

func collectFieldsToResolve(sels []selected.Selection, resolver reflect.Value, fields *[]*fieldWithResolver) {
	for _, sel := range sels {
		switch sel := sel.(type) {
		case *selected.SchemaField:
			*fields = append(*fields, &fieldWithResolver{field: sel, resolver: resolver})

		case *selected.TypenameField:
			sf := &selected.SchemaField{
				Field:       resolvable.MetaFieldTypename,
				Alias:       sel.Alias,
				FixedResult: reflect.ValueOf(typeOf(sel, resolver)),
			}
			*fields = append(*fields, &fieldWithResolver{field: sf, resolver: resolver})

		case *selected.TypeAssertion:
			out := resolver.Method(sel.MethodIndex).Call(nil)
			if !out[1].Bool() {
				continue
			}
			collectFieldsToResolve(sel.Sels, out[0], fields)

		default:
			panic("unreachable")
		}
	}
}

func typeOf(tf *selected.TypenameField, resolver reflect.Value) string {
	if len(tf.TypeAssertions) == 0 {
		return tf.Name
	}
	for name, a := range tf.TypeAssertions {
		out := resolver.Method(a.MethodIndex).Call(nil)
		if out[1].Bool() {
			return name
		}
	}
	return ""
}

func (r *Request) execFieldSelection(ctx context.Context, field *selected.SchemaField, resolver reflect.Value, out *bytes.Buffer, applyLimiter bool) {
	if applyLimiter {
		r.Limiter <- struct{}{}
	}

	var result reflect.Value
	var err *errors.QueryError

	traceCtx, finish := r.Tracer.TraceField(ctx, field.TraceLabel, field.TypeName, field.Name, !field.Async, field.Args)
	defer func() {
		finish(err)
	}()

	err = func() (err *errors.QueryError) {
		defer func() {
			if panicValue := recover(); panicValue != nil {
				r.Logger.LogPanic(ctx, panicValue)
				err = makePanicError(panicValue)
			}
		}()

		if field.FixedResult.IsValid() {
			result = field.FixedResult
			return nil
		}

		if err := traceCtx.Err(); err != nil {
			return errors.Errorf("%s", err) // don't execute any more resolvers if context got cancelled
		}

		var in []reflect.Value
		if field.HasContext {
			in = append(in, reflect.ValueOf(traceCtx))
		}
		if field.ArgsPacker != nil {
			in = append(in, field.PackedArgs)
		}
		callOut := resolver.Method(field.MethodIndex).Call(in)
		result = callOut[0]
		if field.HasError && !callOut[1].IsNil() {
			resolverErr := callOut[1].Interface().(error)
			err := errors.Errorf("%s", resolverErr)
			err.ResolverError = resolverErr
			return err
		}
		return nil
	}()

	if applyLimiter {
		<-r.Limiter
	}

	if err != nil {
		r.AddError(err)
		out.WriteString("null") // TODO handle non-nil
		return
	}

	r.execSelectionSet(traceCtx, field.Sels, field.Type, result, out)
}

func (r *Request) execSelectionSet(ctx context.Context, sels []selected.Selection, typ common.Type, resolver reflect.Value, out *bytes.Buffer) {
	t, nonNull := unwrapNonNull(typ)
	switch t := t.(type) {
	case *schema.Object, *schema.Interface, *schema.Union:
		if resolver.IsNil() {
			if nonNull {
				panic(errors.Errorf("got nil for non-null %q", t))
			}
			out.WriteString("null")
			return
		}

		r.execSelections(ctx, sels, resolver, out, false)
		return
	}

	if !nonNull {
		if resolver.IsNil() {
			out.WriteString("null")
			return
		}
		resolver = resolver.Elem()
	}

	switch t := t.(type) {
	case *common.List:
		l := resolver.Len()

		if selected.HasAsyncSel(sels) {
			var wg sync.WaitGroup
			wg.Add(l)
			entryouts := make([]bytes.Buffer, l)
			for i := 0; i < l; i++ {
				go func(i int) {
					defer r.handlePanic(ctx)
					r.execSelectionSet(ctx, sels, t.OfType, resolver.Index(i), &entryouts[i])
					wg.Done()
				}(i)
			}
			wg.Wait()

			out.WriteByte('[')
			for i, entryout := range entryouts {
				if i > 0 {
					out.WriteByte(',')
				}
				out.Write(entryout.Bytes())
			}
			out.WriteByte(']')
			return
		}

		out.WriteByte('[')
		for i := 0; i < l; i++ {
			if i > 0 {
				out.WriteByte(',')
			}
			r.execSelectionSet(ctx, sels, t.OfType, resolver.Index(i), out)
		}
		out.WriteByte(']')

	case *schema.Scalar:
		var b []byte // TODO use scratch
		switch t.Name {
		case "Int":
			out.Write(strconv.AppendInt(b, resolver.Int(), 10))
		case "Float":
			out.Write(strconv.AppendFloat(b, resolver.Float(), 'f', -1, 64))
		case "String":
			out.Write(strconv.AppendQuote(b, resolver.String()))
		case "Boolean":
			out.Write(strconv.AppendBool(b, resolver.Bool()))
		default:
			v := resolver.Interface().(marshaler)
			data, err := v.MarshalJSON()
			if err != nil {
				panic(errors.Errorf("could not marshal %v", v))
			}
			out.Write(data)
		}

	case *schema.Enum:
		out.WriteByte('"')
		out.WriteString(resolver.String())
		out.WriteByte('"')

	default:
		panic("unreachable")
	}
}

func unwrapNonNull(t common.Type) (common.Type, bool) {
	if nn, ok := t.(*common.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}

type marshaler interface {
	MarshalJSON() ([]byte, error)
}
