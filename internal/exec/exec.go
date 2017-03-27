package exec

import (
	"context"
	"log"
	"reflect"
	"runtime"
	"sync"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/exec/resolvable"
	"github.com/neelance/graphql-go/internal/exec/selected"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/trace"
)

type Request struct {
	selected.Request
	Limiter chan struct{}
	Tracer  trace.Tracer
	wg      sync.WaitGroup
}

func (r *Request) handlePanic() {
	if err := recover(); err != nil {
		r.AddError(makePanicError(err))
	}
}

func makePanicError(value interface{}) *errors.QueryError {
	err := errors.Errorf("graphql: panic occurred: %v", value)
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Printf("%s\n%s", err, buf)
	return err
}

func (r *Request) Execute(ctx context.Context, s *resolvable.Schema, op *query.Operation) (interface{}, []*errors.QueryError) {
	results := make(map[string]interface{})
	func() {
		defer r.handlePanic()
		sels := selected.ApplyOperation(&r.Request, s, op)
		for _, sel := range sels {
			r.execSelection(ctx, sel, s.Resolver, results)
			if op.Type == query.Mutation {
				r.wg.Wait()
			}
		}
	}()
	r.wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, []*errors.QueryError{errors.Errorf("%s", err)}
	}

	return results, r.Errs
}

func (r *Request) execSelection(ctx context.Context, sel selected.Selection, resolver reflect.Value, results map[string]interface{}) {
	switch sel := sel.(type) {
	case *selected.SchemaField:
		r.execFieldSelection(ctx, sel, resolver, results)

	case *selected.TypenameField:
		if len(sel.TypeAssertions) == 0 {
			results[sel.Alias] = sel.Name
			return
		}
		for name, a := range sel.TypeAssertions {
			out := resolver.Method(a.MethodIndex).Call(nil)
			if out[1].Bool() {
				results[sel.Alias] = name
				return
			}
		}

	case *selected.MetaField:
		subresults := make(map[string]interface{})
		for _, subsel := range sel.Sels {
			r.execSelection(ctx, subsel, sel.Resolver, subresults)
		}
		results[sel.Alias] = subresults

	case *selected.TypeAssertion:
		out := resolver.Method(sel.MethodIndex).Call(nil)
		if !out[1].Bool() {
			return
		}
		for _, sel := range sel.Sels {
			r.execSelection(ctx, sel, out[0], results)
		}

	default:
		panic("unreachable")
	}
}

func (r *Request) execFieldSelection(ctx context.Context, field *selected.SchemaField, resolver reflect.Value, results map[string]interface{}) {
	do := func(applyLimiter bool) interface{} {
		if applyLimiter {
			r.Limiter <- struct{}{}
		}

		var result reflect.Value
		var err *errors.QueryError

		traceCtx, finish := r.Tracer.TraceField(ctx, field.TraceLabel, field.TypeName, field.Name, field.Trivial, field.Args)
		defer func() {
			finish(err)
		}()

		err = func() (err *errors.QueryError) {
			defer func() {
				if panicValue := recover(); panicValue != nil {
					err = makePanicError(panicValue)
				}
			}()

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
			out := resolver.Method(field.MethodIndex).Call(in)
			result = out[0]
			if field.HasError && !out[1].IsNil() {
				resolverErr := out[1].Interface().(error)
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
			return nil // TODO handle non-nil
		}

		return r.execSelectionSet(traceCtx, field.Sels, field.Type, result)
	}

	if field.Trivial {
		results[field.Alias] = do(false)
		return
	}

	result := new(interface{})
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		*result = do(true)
	}()
	results[field.Alias] = result
}

func (r *Request) execSelectionSet(ctx context.Context, sels []selected.Selection, typ common.Type, resolver reflect.Value) interface{} {
	t, nonNull := unwrapNonNull(typ)
	switch t := t.(type) {
	case *schema.Object, *schema.Interface, *schema.Union:
		if resolver.IsNil() {
			if nonNull {
				panic(errors.Errorf("got nil for non-null %q", t))
			}
			return nil
		}
		results := make(map[string]interface{})
		for _, sel := range sels {
			r.execSelection(ctx, sel, resolver, results)
		}
		return results

	case *common.List:
		if !nonNull {
			if resolver.IsNil() {
				return nil
			}
			resolver = resolver.Elem()
		}
		l := make([]interface{}, resolver.Len())
		for i := range l {
			l[i] = r.execSelectionSet(ctx, sels, t.OfType, resolver.Index(i))
		}
		return l

	case *schema.Scalar, *schema.Enum:
		return resolver.Interface()

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
