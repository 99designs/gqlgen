package exec

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

type Exec struct {
	iExec
	resolver reflect.Value
}

func Make(s *schema.Schema, resolver interface{}) *Exec {
	t := s.Types[s.EntryPoints["query"]]
	return &Exec{
		iExec:    makeExec(s, t, reflect.TypeOf(resolver), make(map[typeRefMapKey]*typeRefExec)),
		resolver: reflect.ValueOf(resolver),
	}
}

func (e *Exec) Exec(document *query.Document, variables map[string]interface{}, selSet *query.SelectionSet) interface{} {
	return e.exec(&request{document, variables}, selSet, e.resolver)
}

func makeExec(s *schema.Schema, t schema.Type, resolverType reflect.Type, typeRefMap map[typeRefMapKey]*typeRefExec) iExec {
	switch t := t.(type) {
	case *schema.Scalar:
		return &scalarExec{}

	case *schema.Object:
		fields := make(map[string]*fieldExec)

		for name, f := range t.Fields {
			methodIndex := -1
			for i := 0; i < resolverType.NumMethod(); i++ {
				if strings.EqualFold(name, resolverType.Method(i).Name) {
					methodIndex = i
					break
				}
			}
			if methodIndex == -1 {
				// continue // TODO error
				panic(fmt.Errorf("%s does not implement %q: missing method for %q", resolverType, t.Name, name)) // TODO proper error handling
			}

			fields[name] = &fieldExec{
				field:       f,
				methodIndex: methodIndex,
				valueExec:   makeExec(s, f.Type, resolverType.Method(methodIndex).Type.Out(0), typeRefMap),
			}
		}

		return &objectExec{
			fields: fields,
		}

	case *schema.Union:
		return nil // TODO

	case *schema.Enum:
		return &scalarExec{}

	case *schema.List:
		if resolverType.Kind() != reflect.Slice {
			panic(fmt.Errorf("%s is not a slice", resolverType)) // TODO proper error handling
		}
		return &listExec{
			elem: makeExec(s, t.Elem, resolverType.Elem(), typeRefMap),
		}

	case *schema.TypeReference:
		refT, ok := s.Types[t.Name]
		if !ok {
			panic("type not found: " + t.Name) // TODO proper error
		}
		k := typeRefMapKey{refT, resolverType}
		e, ok := typeRefMap[k]
		if !ok {
			e = &typeRefExec{}
			typeRefMap[k] = e
			e.iExec = makeExec(s, refT, resolverType, typeRefMap)
		}
		return e

	default:
		panic("invalid type")
	}
}

type request struct {
	*query.Document
	Variables map[string]interface{}
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
			l[i] = e.elem.exec(r, selSet, resolver.Index(i))
			wg.Done()
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
	fields map[string]*fieldExec
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
	e.execFragment(r, selSet, resolver, addResult)
	return results
}

func (e *objectExec) execFragment(r *request, selSet *query.SelectionSet, resolver reflect.Value, addResult addResultFn) {
	var wg sync.WaitGroup
	for _, sel := range selSet.Selections {
		switch sel := sel.(type) {
		case *query.Field:
			if !skipByDirective(r, sel.Directives) {
				wg.Add(1)
				go func(f *query.Field) {
					e.fields[f.Name].execField(r, f, resolver, addResult)
					wg.Done()
				}(sel)
			}

		case *query.FragmentSpread:
			if !skipByDirective(r, sel.Directives) {
				wg.Add(1)
				go func(fs *query.FragmentSpread) {
					e.execFragment(r, r.Fragments[fs.Name].SelSet, resolver, addResult)
					wg.Done()
				}(sel)
			}

		default:
			panic("invalid type")
		}
	}
	wg.Wait()
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
			rf := args.Elem().FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, name) })
			rf.Set(reflect.ValueOf(execValue(r, value)))
		}
		in = []reflect.Value{args.Elem()}
	}
	addResult(f.Alias, e.valueExec.exec(r, f.SelSet, m.Call(in)[0]))
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
