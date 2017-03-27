package exec

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"sync"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/lexer"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/introspection"
	"github.com/neelance/graphql-go/trace"
)

type Request struct {
	Doc     *query.Document
	Vars    map[string]interface{}
	Schema  *schema.Schema
	Limiter chan struct{}
	Tracer  trace.Tracer
	wg      sync.WaitGroup
	mu      sync.Mutex
	errs    []*errors.QueryError
}

func (r *Request) addError(err *errors.QueryError) {
	r.mu.Lock()
	r.errs = append(r.errs, err)
	r.mu.Unlock()
}

func (r *Request) handlePanic() {
	if err := recover(); err != nil {
		r.addError(makePanicError(err))
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

func (r *Request) resolveVar(value interface{}) interface{} {
	if v, ok := value.(lexer.Variable); ok {
		value = r.Vars[string(v)]
	}
	return value
}

func (r *Request) Execute(ctx context.Context, e *Exec, op *query.Operation) (interface{}, []*errors.QueryError) {
	var opExec *objectExec
	var serially bool
	switch op.Type {
	case query.Query:
		opExec = e.queryExec.(*objectExec)
		serially = false
	case query.Mutation:
		opExec = e.mutationExec.(*objectExec)
		serially = true
	}

	results := make(map[string]interface{})
	func() {
		defer r.handlePanic()
		sels := applySelectionSet(r, opExec, op.SelSet)
		for _, sel := range sels {
			execSelection(ctx, sel, e.resolver, results)
			if serially {
				r.wg.Wait()
			}
		}
	}()
	r.wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, []*errors.QueryError{errors.Errorf("%s", err)}
	}

	return results, r.errs
}

type appliedSelectionSet struct {
	sels []appliedSelection
}

type appliedSelection interface{}

type appliedFieldSelection struct {
	fieldExec
	req        *Request
	alias      string
	args       map[string]interface{}
	packedArgs reflect.Value
	sels       []appliedSelection
}

type appliedTypeAssertion struct {
	methodIndex int
	sels        []appliedSelection
}

type typenameFieldSelection struct {
	objectExec
	alias string
}

type metaFieldSelection struct {
	alias    string
	sels     []appliedSelection
	resolver reflect.Value
}

func applySelectionSet(r *Request, e *objectExec, selSet *query.SelectionSet) (sels []appliedSelection) {
	if selSet == nil {
		return nil
	}
	for _, sel := range selSet.Selections {
		switch sel := sel.(type) {
		case *query.Field:
			field := sel
			if skipByDirective(r, field.Directives) {
				continue
			}

			switch field.Name.Name {
			case "__typename":
				sels = append(sels, &typenameFieldSelection{
					objectExec: *e,
					alias:      field.Alias.Name,
				})

			case "__schema":
				sels = append(sels, &metaFieldSelection{
					alias:    field.Alias.Name,
					sels:     applySelectionSet(r, schemaExec, field.SelSet),
					resolver: reflect.ValueOf(introspection.WrapSchema(r.Schema)),
				})

			case "__type":
				p := valuePacker{valueType: reflect.TypeOf("")}
				v, err := p.pack(r, r.resolveVar(field.Arguments.MustGet("name").Value))
				if err != nil {
					r.addError(errors.Errorf("%s", err))
					return nil
				}

				t, ok := r.Schema.Types[v.String()]
				if !ok {
					return nil
				}

				sels = append(sels, &metaFieldSelection{
					alias:    field.Alias.Name,
					sels:     applySelectionSet(r, typeExec, field.SelSet),
					resolver: reflect.ValueOf(introspection.WrapType(t)),
				})

			default:
				fe := e.fields[field.Name.Name]

				var args map[string]interface{}
				var packedArgs reflect.Value
				if fe.argsPacker != nil {
					args = make(map[string]interface{})
					for _, arg := range field.Arguments {
						args[arg.Name.Name] = arg.Value.Value
					}
					var err error
					packedArgs, err = fe.argsPacker.pack(r, args)
					if err != nil {
						r.addError(errors.Errorf("%s", err))
						return
					}
				}

				sels = append(sels, &appliedFieldSelection{
					fieldExec:  *fe,
					req:        r,
					alias:      field.Alias.Name,
					args:       args,
					packedArgs: packedArgs,
					sels:       applyField(r, fe.valueExec, field.SelSet),
				})
			}

		case *query.InlineFragment:
			frag := sel
			if skipByDirective(r, frag.Directives) {
				continue
			}
			sels = append(sels, applyFragment(r, e, &frag.Fragment)...)

		case *query.FragmentSpread:
			spread := sel
			if skipByDirective(r, spread.Directives) {
				continue
			}
			sels = append(sels, applyFragment(r, e, &r.Doc.Fragments.Get(spread.Name.Name).Fragment)...)

		default:
			panic("invalid type")
		}
	}
	return
}

func applyFragment(r *Request, e *objectExec, frag *query.Fragment) []appliedSelection {
	if frag.On.Name != "" && frag.On.Name != e.name {
		a, ok := e.typeAssertions[frag.On.Name]
		if !ok {
			panic(fmt.Errorf("%q does not implement %q", frag.On, e.name)) // TODO proper error handling
		}

		return []appliedSelection{&appliedTypeAssertion{
			methodIndex: a.methodIndex,
			sels:        applySelectionSet(r, a.typeExec.(*objectExec), frag.SelSet),
		}}
	}
	return applySelectionSet(r, e, frag.SelSet)
}

func applyField(r *Request, e iExec, selSet *query.SelectionSet) []appliedSelection {
	switch e := e.(type) {
	case *objectExec:
		return applySelectionSet(r, e, selSet)
	case *listExec:
		return applyField(r, e.elem, selSet)
	case *scalarExec:
		return nil
	default:
		panic("unreachable")
	}
}

func skipByDirective(r *Request, directives common.DirectiveList) bool {
	if d := directives.Get("skip"); d != nil {
		p := valuePacker{valueType: reflect.TypeOf(false)}
		v, err := p.pack(r, r.resolveVar(d.Args.MustGet("if").Value))
		if err != nil {
			r.addError(errors.Errorf("%s", err))
		}
		if err == nil && v.Bool() {
			return true
		}
	}

	if d := directives.Get("include"); d != nil {
		p := valuePacker{valueType: reflect.TypeOf(false)}
		v, err := p.pack(r, r.resolveVar(d.Args.MustGet("if").Value))
		if err != nil {
			r.addError(errors.Errorf("%s", err))
		}
		if err == nil && !v.Bool() {
			return true
		}
	}

	return false
}
