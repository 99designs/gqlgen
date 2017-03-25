package exec

import (
	"context"
	"reflect"

	"github.com/neelance/graphql-go/errors"
)

func execSelection(ctx context.Context, sel appliedSelection, resolver reflect.Value, results map[string]interface{}) {
	switch sel := sel.(type) {
	case *appliedTypeAssertion:
		out := resolver.Method(sel.methodIndex).Call(nil)
		if !out[1].Bool() {
			return
		}
		for _, sel := range sel.sels {
			execSelection(ctx, sel, out[0], results)
		}

	case *appliedFieldSelection:
		sel.execSelection(ctx, resolver, results)

	case *typenameFieldSelection:
		results[sel.alias] = typenameOf(sel.oe, resolver)

	case *metaFieldSelection:
		results[sel.alias] = schemaExec.exec(ctx, sel.sels, sel.resolver)

	default:
		panic("unreachable")
	}
}

func typenameOf(e *objectExec, resolver reflect.Value) interface{} {
	if len(e.typeAssertions) == 0 {
		return e.name
	}

	for name, a := range e.typeAssertions {
		out := resolver.Method(a.methodIndex).Call(nil)
		if out[1].Bool() {
			return name
		}
	}

	return nil
}

func (afs *appliedFieldSelection) execSelection(ctx context.Context, resolver reflect.Value, results map[string]interface{}) {
	fe := afs.exec
	do := func(applyLimiter bool) interface{} {
		if applyLimiter {
			afs.req.Limiter <- struct{}{}
		}

		var result reflect.Value
		var err *errors.QueryError

		traceCtx, finish := afs.req.Tracer.TraceField(ctx, fe.traceLabel, fe.typeName, fe.field.Name, fe.trivial, afs.args)
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
			if fe.hasContext {
				in = append(in, reflect.ValueOf(traceCtx))
			}
			if fe.argsPacker != nil {
				in = append(in, afs.packedArgs)
			}
			out := resolver.Method(fe.methodIndex).Call(in)
			result = out[0]
			if fe.hasError && !out[1].IsNil() {
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

		return fe.valueExec.exec(traceCtx, afs.sels, result)
	}

	if fe.trivial {
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

func (e *objectExec) exec(ctx context.Context, sels []appliedSelection, resolver reflect.Value) interface{} {
	if resolver.IsNil() {
		if e.nonNull {
			panic(errors.Errorf("got nil for non-null %q", e.name))
		}
		return nil
	}
	results := make(map[string]interface{})
	for _, sel := range sels {
		execSelection(ctx, sel, resolver, results)
	}
	return results
}

func (e *listExec) exec(ctx context.Context, sels []appliedSelection, resolver reflect.Value) interface{} {
	if !e.nonNull {
		if resolver.IsNil() {
			return nil
		}
		resolver = resolver.Elem()
	}
	l := make([]interface{}, resolver.Len())
	for i := range l {
		l[i] = e.elem.exec(ctx, sels, resolver.Index(i))
	}
	return l
}

func (e *scalarExec) exec(ctx context.Context, sels []appliedSelection, resolver reflect.Value) interface{} {
	return resolver.Interface()
}
