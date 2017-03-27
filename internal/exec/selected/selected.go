package selected

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/exec/resolvable"
	"github.com/neelance/graphql-go/internal/lexer"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/introspection"
)

type Request struct {
	Schema *schema.Schema
	Doc    *query.Document
	Vars   map[string]interface{}
	Mu     sync.Mutex
	Errs   []*errors.QueryError
}

func (r *Request) resolveVar(value interface{}) interface{} {
	if v, ok := value.(lexer.Variable); ok {
		value = r.Vars[string(v)]
	}
	return value
}

func (r *Request) AddError(err *errors.QueryError) {
	r.Mu.Lock()
	r.Errs = append(r.Errs, err)
	r.Mu.Unlock()
}

func ApplyOperation(r *Request, s *resolvable.Schema, op *query.Operation) []Selection {
	var obj *resolvable.Object
	switch op.Type {
	case query.Query:
		obj = s.Query.(*resolvable.Object)
	case query.Mutation:
		obj = s.Mutation.(*resolvable.Object)
	}
	return applySelectionSet(r, obj, op.SelSet)
}

type Selection interface {
	isSelection()
}

type SchemaField struct {
	resolvable.Field
	Alias       string
	Args        map[string]interface{}
	PackedArgs  reflect.Value
	Sels        []Selection
	FixedResult reflect.Value
}

type TypeAssertion struct {
	resolvable.TypeAssertion
	Sels []Selection
}

type TypenameField struct {
	resolvable.Object
	Alias string
}

func (*SchemaField) isSelection()   {}
func (*TypeAssertion) isSelection() {}
func (*TypenameField) isSelection() {}

func applySelectionSet(r *Request, e *resolvable.Object, selSet *query.SelectionSet) (sels []Selection) {
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
				sels = append(sels, &TypenameField{
					Object: *e,
					Alias:  field.Alias.Name,
				})

			case "__schema":
				sels = append(sels, &SchemaField{
					Field:       resolvable.MetaFieldSchema,
					Alias:       field.Alias.Name,
					Sels:        applySelectionSet(r, resolvable.MetaSchema, field.SelSet),
					FixedResult: reflect.ValueOf(introspection.WrapSchema(r.Schema)),
				})

			case "__type":
				p := resolvable.ValuePacker{ValueType: reflect.TypeOf("")}
				v, err := p.Pack(&resolvable.Request{Vars: r.Vars}, r.resolveVar(field.Arguments.MustGet("name").Value))
				if err != nil {
					r.AddError(errors.Errorf("%s", err))
					return nil
				}

				t, ok := r.Schema.Types[v.String()]
				if !ok {
					return nil
				}

				sels = append(sels, &SchemaField{
					Field:       resolvable.MetaFieldType,
					Alias:       field.Alias.Name,
					Sels:        applySelectionSet(r, resolvable.MetaType, field.SelSet),
					FixedResult: reflect.ValueOf(introspection.WrapType(t)),
				})

			default:
				fe := e.Fields[field.Name.Name]

				var args map[string]interface{}
				var packedArgs reflect.Value
				if fe.ArgsPacker != nil {
					args = make(map[string]interface{})
					for _, arg := range field.Arguments {
						args[arg.Name.Name] = arg.Value.Value
					}
					var err error
					packedArgs, err = fe.ArgsPacker.Pack(&resolvable.Request{Vars: r.Vars}, args)
					if err != nil {
						r.AddError(errors.Errorf("%s", err))
						return
					}
				}

				sels = append(sels, &SchemaField{
					Field:      *fe,
					Alias:      field.Alias.Name,
					Args:       args,
					PackedArgs: packedArgs,
					Sels:       applyField(r, fe.ValueExec, field.SelSet),
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

func applyFragment(r *Request, e *resolvable.Object, frag *query.Fragment) []Selection {
	if frag.On.Name != "" && frag.On.Name != e.Name {
		a, ok := e.TypeAssertions[frag.On.Name]
		if !ok {
			panic(fmt.Errorf("%q does not implement %q", frag.On, e.Name)) // TODO proper error handling
		}

		return []Selection{&TypeAssertion{
			TypeAssertion: *a,
			Sels:          applySelectionSet(r, a.TypeExec.(*resolvable.Object), frag.SelSet),
		}}
	}
	return applySelectionSet(r, e, frag.SelSet)
}

func applyField(r *Request, e resolvable.Resolvable, selSet *query.SelectionSet) []Selection {
	switch e := e.(type) {
	case *resolvable.Object:
		return applySelectionSet(r, e, selSet)
	case *resolvable.List:
		return applyField(r, e.Elem, selSet)
	case *resolvable.Scalar:
		return nil
	default:
		panic("unreachable")
	}
}

func skipByDirective(r *Request, directives common.DirectiveList) bool {
	if d := directives.Get("skip"); d != nil {
		p := resolvable.ValuePacker{ValueType: reflect.TypeOf(false)}
		v, err := p.Pack(&resolvable.Request{Vars: r.Vars}, r.resolveVar(d.Args.MustGet("if").Value))
		if err != nil {
			r.AddError(errors.Errorf("%s", err))
		}
		if err == nil && v.Bool() {
			return true
		}
	}

	if d := directives.Get("include"); d != nil {
		p := resolvable.ValuePacker{ValueType: reflect.TypeOf(false)}
		v, err := p.Pack(&resolvable.Request{Vars: r.Vars}, r.resolveVar(d.Args.MustGet("if").Value))
		if err != nil {
			r.AddError(errors.Errorf("%s", err))
		}
		if err == nil && !v.Bool() {
			return true
		}
	}

	return false
}
