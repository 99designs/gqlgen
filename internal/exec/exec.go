package exec

import (
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

type Exec struct {
	iExec
	resolver reflect.Value
}

func Make(s *schema.Schema, resolver interface{}) (*Exec, error) {
	implementsMap := make(map[string][]string)
	for _, t := range s.Types {
		if obj, ok := t.(*schema.Object); ok && obj.Implements != "" {
			implementsMap[obj.Implements] = append(implementsMap[obj.Implements], obj.Name)
		}
	}

	t := s.Types[s.EntryPoints["query"]]
	e, err := makeExec(s, t, reflect.TypeOf(resolver), implementsMap, make(map[typeRefMapKey]*typeRefExec))
	if err != nil {
		return nil, err
	}
	return &Exec{
		iExec:    e,
		resolver: reflect.ValueOf(resolver),
	}, nil
}

func (e *Exec) Exec(document *query.Document, variables map[string]interface{}, selSet *query.SelectionSet) (interface{}, error) {
	r := &request{
		Document:  document,
		Variables: variables,
	}

	res := func() interface{} {
		defer r.handlePanic()
		return e.exec(r, selSet, e.resolver)
	}()

	if r.Error != nil {
		return nil, r.Error
	}
	return res, nil
}

func makeExec(s *schema.Schema, t schema.Type, resolverType reflect.Type, implementsMap map[string][]string, typeRefMap map[typeRefMapKey]*typeRefExec) (iExec, error) {
	switch t := t.(type) {
	case *schema.Scalar:
		return &scalarExec{}, nil

	case *schema.Object:
		fields := make(map[string]*fieldExec)
		for name, f := range t.Fields {
			methodIndex := findMethod(resolverType, name)
			if methodIndex == -1 {
				return nil, fmt.Errorf("%s does not resolve %q: missing method for field %q", resolverType, t.Name, name)
			}

			ve, err := makeExec(s, f.Type, resolverType.Method(methodIndex).Type.Out(0), implementsMap, typeRefMap)
			if err != nil {
				return nil, err
			}
			fields[name] = &fieldExec{
				field:       f,
				methodIndex: methodIndex,
				valueExec:   ve,
			}
		}

		typeAssertions, err := makeTypeAssertions(s, t.Name, implementsMap[t.Name], resolverType, implementsMap, typeRefMap)
		if err != nil {
			return nil, err
		}
		return &objectExec{
			name:           t.Name,
			fields:         fields,
			typeAssertions: typeAssertions,
		}, nil

	case *schema.Union:
		typeAssertions, err := makeTypeAssertions(s, t.Name, t.Types, resolverType, implementsMap, typeRefMap)
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
		e, err := makeExec(s, t.Elem, resolverType.Elem(), implementsMap, typeRefMap)
		if err != nil {
			return nil, err
		}
		return &listExec{
			elem: e,
		}, nil

	case *schema.TypeReference:
		e, err := resolveType(s, t.Name, resolverType, implementsMap, typeRefMap)
		return e, err

	default:
		panic("invalid type")
	}
}

func makeTypeAssertions(s *schema.Schema, typeName string, impls []string, resolverType reflect.Type, implementsMap map[string][]string, typeRefMap map[typeRefMapKey]*typeRefExec) (map[string]*typeAssertExec, error) {
	typeAssertions := make(map[string]*typeAssertExec)
	for _, impl := range impls {
		methodIndex := findMethod(resolverType, "to"+impl)
		if methodIndex == -1 {
			return nil, fmt.Errorf("%s does not resolve %q: missing method %q to convert to %q", resolverType, typeName, "to"+impl, impl)
		}
		e, err := resolveType(s, impl, resolverType.Method(methodIndex).Type.Out(0), implementsMap, typeRefMap)
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

func resolveType(s *schema.Schema, name string, resolverType reflect.Type, implementsMap map[string][]string, typeRefMap map[typeRefMapKey]*typeRefExec) (*typeRefExec, error) {
	refT, ok := s.Types[name]
	if !ok {
		return nil, fmt.Errorf("type %q not found", name)
	}
	k := typeRefMapKey{refT, resolverType}
	e, ok := typeRefMap[k]
	if !ok {
		e = &typeRefExec{}
		typeRefMap[k] = e
		var err error
		e.iExec, err = makeExec(s, refT, resolverType, implementsMap, typeRefMap)
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
	*query.Document
	Variables map[string]interface{}
	Mu        sync.Mutex
	Error     error
}

func (r *request) handlePanic() {
	if err := recover(); err != nil {
		r.Mu.Lock()
		defer r.Mu.Unlock()
		r.Error = fmt.Errorf("graphql: panic occured: %v", err)

		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		log.Printf("%s\n%s", r.Error, buf)
	}
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
					if f.Name == "__typename" {
						for name, a := range e.typeAssertions {
							out := resolver.Method(a.methodIndex).Call(nil)
							if out[1].Bool() {
								addResult(f.Alias, name)
								return
							}
						}
						return
					}
					fe, ok := e.fields[f.Name]
					if !ok {
						panic(fmt.Errorf("%q has no field %q", e.name, f.Name)) // TODO proper error handling
					}
					fe.execField(r, f, resolver, addResult)
				}(sel)
			}

		case *query.FragmentSpread:
			if !skipByDirective(r, sel.Directives) {
				wg.Add(1)
				go func(fs *query.FragmentSpread) {
					defer wg.Done()
					defer r.handlePanic()
					frag, ok := r.Fragments[fs.Name]
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
		out := resolver.Method(a.methodIndex).Call(nil)
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
	var in []reflect.Value
	if len(e.field.Parameters) != 0 {
		args := reflect.New(m.Type().In(0))
		for name, param := range e.field.Parameters {
			value, ok := f.Arguments[name]
			if !ok {
				value = &query.Literal{Value: param.Default}
			}
			rf := args.Elem().FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, name) }) // TODO resolve at startup
			rf.Set(reflect.ValueOf(execValue(r, value)))
		}
		in = []reflect.Value{args.Elem()}
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
		return r.Variables[v.Name]
	case *query.Literal:
		return v.Value
	default:
		panic("invalid value")
	}
}
