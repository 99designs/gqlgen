package exec

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

type Exec struct {
	iExec
	schema   *schema.Schema
	resolver reflect.Value
}

var scalarTypes = map[string]iExec{
	"Int":     &scalarExec{},
	"Float":   &scalarExec{},
	"String":  &scalarExec{},
	"Boolean": &scalarExec{},
	"ID":      &scalarExec{},
}
var scalarTypeNames = []string{"Int", "Float", "String", "Boolean", "ID"}

func Make(s *schema.Schema, resolver interface{}) (*Exec, error) {
	t := s.AllTypes[s.EntryPoints["query"]]
	e, err := makeExec(s, t, reflect.TypeOf(resolver), make(map[typeRefMapKey]*typeRefExec))
	if err != nil {
		return nil, err
	}
	return &Exec{
		iExec:    e,
		schema:   s,
		resolver: reflect.ValueOf(resolver),
	}, nil
}

func makeExec(s *schema.Schema, t schema.Type, resolverType reflect.Type, typeRefMap map[typeRefMapKey]*typeRefExec) (iExec, error) {
	switch t := t.(type) {
	case *schema.Object:
		fields, err := makeFieldExecs(s, t.Name, t.Fields, resolverType, typeRefMap)
		if err != nil {
			return nil, err
		}

		return &objectExec{
			name:   t.Name,
			fields: fields,
		}, nil

	case *schema.Interface:
		fields, err := makeFieldExecs(s, t.Name, t.Fields, resolverType, typeRefMap)
		if err != nil {
			return nil, err
		}

		typeAssertions, err := makeTypeAssertions(s, t.Name, t.ImplementedBy, resolverType, typeRefMap)
		if err != nil {
			return nil, err
		}

		return &objectExec{
			name:           t.Name,
			fields:         fields,
			typeAssertions: typeAssertions,
		}, nil

	case *schema.Union:
		typeAssertions, err := makeTypeAssertions(s, t.Name, t.Types, resolverType, typeRefMap)
		if err != nil {
			return nil, err
		}
		return &objectExec{
			name:           t.Name,
			typeAssertions: typeAssertions,
		}, nil

	case *schema.Enum:
		return &scalarExec{}, nil

	case *schema.List:
		if resolverType.Kind() != reflect.Slice {
			return nil, fmt.Errorf("%s is not a slice", resolverType)
		}
		e, err := makeExec(s, t.Elem, resolverType.Elem(), typeRefMap)
		if err != nil {
			return nil, err
		}
		return &listExec{
			elem: e,
		}, nil

	case *schema.NonNull:
		return makeExec(s, t.Elem, resolverType, typeRefMap)

	case *schema.TypeReference:
		if scalar, ok := scalarTypes[t.Name]; ok {
			return scalar, nil
		}
		e, err := resolveType(s, t.Name, resolverType, typeRefMap)
		return e, err

	default:
		panic("invalid type")
	}
}

func makeFieldExecs(s *schema.Schema, typeName string, fields map[string]*schema.Field, resolverType reflect.Type, typeRefMap map[typeRefMapKey]*typeRefExec) (map[string]*fieldExec, error) {
	fieldExecs := make(map[string]*fieldExec)
	for name, f := range fields {
		methodIndex := findMethod(resolverType, name)
		if methodIndex == -1 {
			return nil, fmt.Errorf("%s does not resolve %q: missing method for field %q", resolverType, typeName, name)
		}

		m := resolverType.Method(methodIndex)
		numIn := m.Type.NumIn()
		if resolverType.Kind() != reflect.Interface {
			numIn-- // first parameter is receiver
		}
		if len(f.Parameters) == 0 && numIn != 1 {
			return nil, fmt.Errorf("method %q of %s must have exactly one parameter", m.Name, resolverType)
		}
		if len(f.Parameters) > 0 && numIn != 2 {
			return nil, fmt.Errorf("method %q of %s must have exactly two parameters", m.Name, resolverType)
		}
		// TODO check parameter types
		if m.Type.NumOut() != 1 {
			return nil, fmt.Errorf("method %q of %s must have exactly one return value", m.Name, resolverType)
		}

		ve, err := makeExec(s, f.Type, m.Type.Out(0), typeRefMap)
		if err != nil {
			return nil, err
		}
		fieldExecs[name] = &fieldExec{
			field:       f,
			methodIndex: methodIndex,
			valueExec:   ve,
		}
	}
	return fieldExecs, nil
}

func makeTypeAssertions(s *schema.Schema, typeName string, impls []string, resolverType reflect.Type, typeRefMap map[typeRefMapKey]*typeRefExec) (map[string]*typeAssertExec, error) {
	typeAssertions := make(map[string]*typeAssertExec)
	for _, impl := range impls {
		methodIndex := findMethod(resolverType, "to"+impl)
		if methodIndex == -1 {
			return nil, fmt.Errorf("%s does not resolve %q: missing method %q to convert to %q", resolverType, typeName, "to"+impl, impl)
		}
		e, err := resolveType(s, impl, resolverType.Method(methodIndex).Type.Out(0), typeRefMap)
		if err != nil {
			return nil, err
		}
		typeAssertions[impl] = &typeAssertExec{
			methodIndex: methodIndex,
			typeExec:    e,
		}
	}
	return typeAssertions, nil
}

func resolveType(s *schema.Schema, name string, resolverType reflect.Type, typeRefMap map[typeRefMapKey]*typeRefExec) (*typeRefExec, error) {
	refT, ok := s.AllTypes[name]
	if !ok {
		return nil, fmt.Errorf("type %q not found", name)
	}
	k := typeRefMapKey{refT, resolverType}
	e, ok := typeRefMap[k]
	if !ok {
		e = &typeRefExec{}
		typeRefMap[k] = e
		var err error
		e.iExec, err = makeExec(s, refT, resolverType, typeRefMap)
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

func findMethod(t reflect.Type, name string) int {
	for i := 0; i < t.NumMethod(); i++ {
		if strings.EqualFold(name, t.Method(i).Name) {
			return i
		}
	}
	return -1
}

type request struct {
	ctx    context.Context
	doc    *query.Document
	vars   map[string]interface{}
	schema *schema.Schema
	mu     sync.Mutex
	errs   []*errors.GraphQLError
}

func (r *request) handlePanic() {
	if err := recover(); err != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
		execErr := errors.Errorf("graphql: panic occured: %v", err)
		r.errs = append(r.errs, execErr)

		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		log.Printf("%s\n%s", execErr, buf)
	}
}

func (e *Exec) Exec(ctx context.Context, document *query.Document, variables map[string]interface{}, selSet *query.SelectionSet) (interface{}, []*errors.GraphQLError) {
	r := &request{
		ctx:    ctx,
		doc:    document,
		vars:   variables,
		schema: e.schema,
	}

	data := func() interface{} {
		defer r.handlePanic()
		return e.exec(r, selSet, e.resolver)
	}()

	return data, r.errs
}

type iExec interface {
	exec(r *request, selSet *query.SelectionSet, resolver reflect.Value) interface{}
}

type scalarExec struct{}

func (e *scalarExec) exec(r *request, selSet *query.SelectionSet, resolver reflect.Value) interface{} {
	return resolver.Interface()
}

type listExec struct {
	elem iExec
}

func (e *listExec) exec(r *request, selSet *query.SelectionSet, resolver reflect.Value) interface{} {
	l := make([]interface{}, resolver.Len())
	var wg sync.WaitGroup
	for i := range l {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer r.handlePanic()
			l[i] = e.elem.exec(r, selSet, resolver.Index(i))
		}(i)
	}
	wg.Wait()
	return l
}

type typeRefExec struct {
	iExec
}

type typeRefMapKey struct {
	s schema.Type
	r reflect.Type
}

type objectExec struct {
	name           string
	fields         map[string]*fieldExec
	typeAssertions map[string]*typeAssertExec
}

type addResultFn func(key string, value interface{})

func (e *objectExec) exec(r *request, selSet *query.SelectionSet, resolver reflect.Value) interface{} {
	var mu sync.Mutex
	results := make(map[string]interface{})
	addResult := func(key string, value interface{}) {
		mu.Lock()
		results[key] = value
		mu.Unlock()
	}
	e.execSelectionSet(r, selSet, resolver, addResult)
	return results
}

func (e *objectExec) execSelectionSet(r *request, selSet *query.SelectionSet, resolver reflect.Value, addResult addResultFn) {
	var wg sync.WaitGroup
	for _, sel := range selSet.Selections {
		switch sel := sel.(type) {
		case *query.Field:
			if !skipByDirective(r, sel.Directives) {
				wg.Add(1)
				go func(f *query.Field) {
					defer wg.Done()
					defer r.handlePanic()
					switch f.Name {
					case "__typename":
						for name, a := range e.typeAssertions {
							out := resolver.Method(a.methodIndex).Call([]reflect.Value{reflect.ValueOf(r.ctx)})
							if out[1].Bool() {
								addResult(f.Alias, name)
								return
							}
						}

					case "__schema":
						addResult(f.Alias, introspectSchema(r, f.SelSet))

					case "__type":
						addResult(f.Alias, introspectType(r, execValue(r, f.Arguments["name"]).(string), f.SelSet))

					default:
						fe, ok := e.fields[f.Name]
						if !ok {
							panic(fmt.Errorf("%q has no field %q", e.name, f.Name)) // TODO proper error handling
						}
						fe.execField(r, f, resolver, addResult)
					}
				}(sel)
			}

		case *query.FragmentSpread:
			if !skipByDirective(r, sel.Directives) {
				wg.Add(1)
				go func(fs *query.FragmentSpread) {
					defer wg.Done()
					defer r.handlePanic()
					frag, ok := r.doc.Fragments[fs.Name]
					if !ok {
						panic(fmt.Errorf("fragment %q not found", fs.Name)) // TODO proper error handling
					}
					e.execFragment(r, &frag.Fragment, resolver, addResult)
				}(sel)
			}

		case *query.InlineFragment:
			if !skipByDirective(r, sel.Directives) {
				wg.Add(1)
				go func(frag *query.InlineFragment) {
					defer wg.Done()
					defer r.handlePanic()
					e.execFragment(r, &frag.Fragment, resolver, addResult)
				}(sel)
			}

		default:
			panic("invalid type")
		}
	}
	wg.Wait()
}

func (e *objectExec) execFragment(r *request, frag *query.Fragment, resolver reflect.Value, addResult addResultFn) {
	if frag.On != "" && frag.On != e.name {
		a, ok := e.typeAssertions[frag.On]
		if !ok {
			panic(fmt.Errorf("%q does not implement %q", frag.On, e.name)) // TODO proper error handling
		}
		out := resolver.Method(a.methodIndex).Call([]reflect.Value{reflect.ValueOf(r.ctx)})
		if !out[1].Bool() {
			return
		}
		a.typeExec.iExec.(*objectExec).execSelectionSet(r, frag.SelSet, out[0], addResult)
		return
	}
	e.execSelectionSet(r, frag.SelSet, resolver, addResult)
}

type fieldExec struct {
	field       *schema.Field
	methodIndex int
	valueExec   iExec
}

func (e *fieldExec) execField(r *request, f *query.Field, resolver reflect.Value, addResult addResultFn) {
	m := resolver.Method(e.methodIndex)
	in := []reflect.Value{reflect.ValueOf(r.ctx)}
	if len(e.field.Parameters) != 0 {
		args := reflect.New(m.Type().In(1))
		for name, param := range e.field.Parameters {
			value, ok := f.Arguments[name]
			if !ok {
				value = &query.Literal{Value: param.Default}
			}
			rf := args.Elem().FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, name) }) // TODO resolve at startup
			rf.Set(reflect.ValueOf(execValue(r, value)))
		}
		in = append(in, args.Elem())
	}
	addResult(f.Alias, e.valueExec.exec(r, f.SelSet, m.Call(in)[0]))
}

type typeAssertExec struct {
	methodIndex int
	typeExec    *typeRefExec
}

func skipByDirective(r *request, d map[string]*query.Directive) bool {
	if skip, ok := d["skip"]; ok {
		if execValue(r, skip.Arguments["if"]).(bool) {
			return true
		}
	}
	if include, ok := d["include"]; ok {
		if !execValue(r, include.Arguments["if"]).(bool) {
			return true
		}
	}
	return false
}

func execValue(r *request, v query.Value) interface{} {
	switch v := v.(type) {
	case *query.Variable:
		return r.vars[v.Name]
	case *query.Literal:
		return v.Value
	default:
		panic("invalid value")
	}
}
