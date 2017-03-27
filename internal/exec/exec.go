package exec

import (
	"context"
	"reflect"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/schema"
)

func execSelection(ctx context.Context, sel appliedSelection, resolver reflect.Value, results map[string]interface{}) {
	switch sel := sel.(type) {
	case *appliedFieldSelection:
		execFieldSelection(ctx, sel, resolver, results)

	case *typenameFieldSelection:
		if len(sel.typeAssertions) == 0 {
			results[sel.alias] = sel.name
			return
		}
		for name, a := range sel.typeAssertions {
			out := resolver.Method(a.methodIndex).Call(nil)
			if out[1].Bool() {
				results[sel.alias] = name
				return
			}
		}

	case *metaFieldSelection:
		subresults := make(map[string]interface{})
		for _, subsel := range sel.sels {
			execSelection(ctx, subsel, sel.resolver, subresults)
		}
		results[sel.alias] = subresults

	case *appliedTypeAssertion:
		out := resolver.Method(sel.methodIndex).Call(nil)
		if !out[1].Bool() {
			return
		}
		for _, sel := range sel.sels {
			execSelection(ctx, sel, out[0], results)
		}

	default:
		panic("unreachable")
	}
}

func execFieldSelection(ctx context.Context, afs *appliedFieldSelection, resolver reflect.Value, results map[string]interface{}) {
	do := func(applyLimiter bool) interface{} {
		if applyLimiter {
			afs.req.Limiter <- struct{}{}
		}

		var result reflect.Value
		var err *errors.QueryError

		traceCtx, finish := afs.req.Tracer.TraceField(ctx, afs.traceLabel, afs.typeName, afs.field.Name, afs.trivial, afs.args)
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
			if afs.hasContext {
				in = append(in, reflect.ValueOf(traceCtx))
			}
			if afs.argsPacker != nil {
				in = append(in, afs.packedArgs)
			}
			out := resolver.Method(afs.methodIndex).Call(in)
			result = out[0]
			if afs.hasError && !out[1].IsNil() {
				resolverErr := out[1].Interface().(error)
				err := errors.Errorf("%s", resolverErr)
				err.ResolverError = resolverErr
				return err
			}
			return nil
		}()

		if applyLimiter {
			<-afs.req.Limiter
		}

		if err != nil {
			afs.req.addError(err)
			return nil // TODO handle non-nil
		}

		return execSelectionSet(traceCtx, afs.sels, afs.field.Type, result)
	}

	if afs.trivial {
		results[afs.alias] = do(false)
		return
	}

	result := new(interface{})
	afs.req.wg.Add(1)
	go func() {
		defer afs.req.wg.Done()
		*result = do(true)
	}()
	results[afs.alias] = result
}

func execSelectionSet(ctx context.Context, sels []appliedSelection, typ common.Type, resolver reflect.Value) interface{} {
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
			execSelection(ctx, sel, resolver, results)
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
			l[i] = execSelectionSet(ctx, sels, t.OfType, resolver.Index(i))
		}
		return l

	case *schema.Scalar, *schema.Enum:
		return resolver.Interface()

	default:
		panic("unreachable")
	}
}
