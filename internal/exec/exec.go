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
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

type Exec struct {
	queryExec    iExec
	mutationExec iExec
	schema       *schema.Schema
	resolver     reflect.Value
}

func Make(s *schema.Schema, resolver interface{}) (*Exec, error) {
	e := &Exec{
		schema:   s,
		resolver: reflect.ValueOf(resolver),
	}

	if t, ok := s.EntryPoints["query"]; ok {
		var err error
		e.queryExec, err = makeWithType(s, t, resolver)
		if err != nil {
			return nil, err
		}
	}

	if t, ok := s.EntryPoints["mutation"]; ok {
		var err error
		e.mutationExec, err = makeWithType(s, t, resolver)
		if err != nil {
			return nil, err
		}
	}

	return e, nil
}

type typeRefMapKey struct {
	s common.Type
	r reflect.Type
}

type typeRef struct {
	targets []*iExec
	exec    iExec
}

func makeWithType(s *schema.Schema, t common.Type, resolver interface{}) (iExec, error) {
	m := make(map[typeRefMapKey]*typeRef)
	var e iExec
	if err := makeExec(&e, s, t, reflect.TypeOf(resolver), m); err != nil {
		return nil, err
	}

	for _, ref := range m {
		for _, target := range ref.targets {
			*target = ref.exec
		}
	}

	return e, nil
}

func makeExec(target *iExec, s *schema.Schema, t common.Type, resolverType reflect.Type, typeRefMap map[typeRefMapKey]*typeRef) error {
	k := typeRefMapKey{t, resolverType}
	ref, ok := typeRefMap[k]
	if !ok {
		ref = &typeRef{}
		typeRefMap[k] = ref
		var err error
		ref.exec, err = makeExec2(s, t, resolverType, typeRefMap)
		if err != nil {
			return err
		}
	}
	ref.targets = append(ref.targets, target)
	return nil
}

var scalarTypes = map[string]reflect.Type{
	"Int":     reflect.TypeOf(int32(0)),
	"Float":   reflect.TypeOf(float64(0)),
	"String":  reflect.TypeOf(""),
	"Boolean": reflect.TypeOf(true),
	"ID":      reflect.TypeOf(""),
}

func makeExec2(s *schema.Schema, t common.Type, resolverType reflect.Type, typeRefMap map[typeRefMapKey]*typeRef) (iExec, error) {
	nonNull := false
	if nn, ok := t.(*common.NonNull); ok {
		nonNull = true
		t = nn.OfType
	}

	if !nonNull {
		if resolverType.Kind() != reflect.Ptr && resolverType.Kind() != reflect.Interface {
			return nil, fmt.Errorf("%s is not a pointer or interface", resolverType)
		}
	}

	switch t := t.(type) {
	case *schema.Scalar:
		if !nonNull {
			resolverType = resolverType.Elem()
		}
		scalarType := scalarTypes[t.Name]
		if resolverType != scalarType {
			return nil, fmt.Errorf("expected %s, got %s", scalarType, resolverType)
		}
		return &scalarExec{}, nil

	case *schema.Object:
		fields, err := makeFieldExecs(s, t.Name, t.Fields, resolverType, typeRefMap)
		if err != nil {
			return nil, err
		}

		return &objectExec{
			name:    t.Name,
			fields:  fields,
			nonNull: nonNull,
		}, nil

	case *schema.Interface:
		fields, err := makeFieldExecs(s, t.Name, t.Fields, resolverType, typeRefMap)
		if err != nil {
			return nil, err
		}

		typeAssertions, err := makeTypeAssertions(s, t.Name, t.PossibleTypes, resolverType, typeRefMap)
		if err != nil {
			return nil, err
		}

		return &objectExec{
			name:           t.Name,
			fields:         fields,
			typeAssertions: typeAssertions,
			nonNull:        nonNull,
		}, nil

	case *schema.Union:
		typeAssertions, err := makeTypeAssertions(s, t.Name, t.PossibleTypes, resolverType, typeRefMap)
		if err != nil {
			return nil, err
		}
		return &objectExec{
			name:           t.Name,
			typeAssertions: typeAssertions,
			nonNull:        nonNull,
		}, nil

	case *schema.Enum:
		return &scalarExec{}, nil

	case *common.List:
		if !nonNull {
			resolverType = resolverType.Elem()
		}
		if resolverType.Kind() != reflect.Slice {
			return nil, fmt.Errorf("%s is not a slice", resolverType)
		}
		e := &listExec{nonNull: nonNull}
		if err := makeExec(&e.elem, s, t.OfType, resolverType.Elem(), typeRefMap); err != nil {
			return nil, err
		}
		return e, nil

	default:
		panic("invalid type")
	}
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func makeFieldExecs(s *schema.Schema, typeName string, fields map[string]*schema.Field, resolverType reflect.Type, typeRefMap map[typeRefMapKey]*typeRef) (map[string]*fieldExec, error) {
	methodHasReceiver := resolverType.Kind() != reflect.Interface
	fieldExecs := make(map[string]*fieldExec)
	for name, f := range fields {
		methodIndex := findMethod(resolverType, name)
		if methodIndex == -1 {
			return nil, fmt.Errorf("%s does not resolve %q: missing method for field %q", resolverType, typeName, name)
		}

		m := resolverType.Method(methodIndex)
		fe, err := makeFieldExec(s, f, m, methodIndex, methodHasReceiver, typeRefMap)
		if err != nil {
			return nil, fmt.Errorf("method %q of %s: %s", m.Name, resolverType, err)
		}
		fieldExecs[name] = fe
	}
	return fieldExecs, nil
}

func makeFieldExec(s *schema.Schema, f *schema.Field, m reflect.Method, methodIndex int, methodHasReceiver bool, typeRefMap map[typeRefMapKey]*typeRef) (*fieldExec, error) {
	in := make([]reflect.Type, m.Type.NumIn())
	for i := range in {
		in[i] = m.Type.In(i)
	}
	if methodHasReceiver {
		in = in[1:] // first parameter is receiver
	}

	hasContext := len(in) > 0 && in[0] == contextType
	if hasContext {
		in = in[1:]
	}

	var argsExec *inputObjectExec
	if len(f.Args.InputFields) > 0 {
		if len(in) == 0 {
			return nil, fmt.Errorf("must have parameter for field arguments")
		}
		var err error
		argsExec, err = makeInputObjectExec(&f.Args, in[0])
		if err != nil {
			return nil, err
		}
		in = in[1:]
	}

	if len(in) > 0 {
		return nil, fmt.Errorf("too many parameters")
	}

	if m.Type.NumOut() > 2 {
		return nil, fmt.Errorf("too many return values")
	}

	hasError := m.Type.NumOut() == 2
	if hasError {
		if m.Type.Out(1) != errorType {
			return nil, fmt.Errorf(`must have "error" as its second return value`)
		}
	}

	fe := &fieldExec{
		field:       f,
		methodIndex: methodIndex,
		hasContext:  hasContext,
		argsExec:    argsExec,
		hasError:    hasError,
	}
	if err := makeExec(&fe.valueExec, s, f.Type, m.Type.Out(0), typeRefMap); err != nil {
		return nil, err
	}
	return fe, nil
}

func makeInputObjectExec(obj *schema.InputObject, typ reflect.Type) (*inputObjectExec, error) {
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected pointer to struct, got %s", typ)
	}
	structType := typ.Elem()

	var fields []*inputFieldExec
	defaultStruct := reflect.New(structType).Elem()
	for _, f := range obj.InputFields {
		fe := &inputFieldExec{
			name: f.Name,
		}

		sf, ok := structType.FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, f.Name) })
		if !ok {
			return nil, fmt.Errorf("missing argument %q", f.Name)
		}
		fe.fieldIndex = sf.Index

		ft := f.Type
		nonNull := (f.Default != nil)
		if nn, ok := ft.(*common.NonNull); ok {
			ft = nn.OfType
			nonNull = true
		}
		expectType := func(got, want reflect.Type) error {
			if got != want {
				return fmt.Errorf("%q has wrong type, expected %s", sf.Name, want)
			}
			return nil
		}
		switch ft := ft.(type) {
		case *schema.Scalar:
			want := scalarTypes[ft.Name]
			if !nonNull {
				want = reflect.PtrTo(want)
			}
			if err := expectType(sf.Type, want); err != nil {
				return nil, err
			}
			fe.exec = &scalarInputExec{
				scalar:  ft,
				nonNull: nonNull,
			}
		case *schema.Enum:
			want := scalarTypes["String"]
			if !nonNull {
				want = reflect.PtrTo(want)
			}
			if err := expectType(sf.Type, want); err != nil {
				return nil, err
			}
			fe.exec = &scalarInputExec{
				scalar:  &schema.Scalar{Name: "String"},
				nonNull: nonNull,
			}
		case *schema.InputObject:
			e, err := makeInputObjectExec(ft, sf.Type)
			if err != nil {
				return nil, err
			}
			fe.exec = e
		default:
			panic("TODO")
		}

		if f.Default != nil {
			defaultStruct.FieldByIndex(fe.fieldIndex).Set(fe.exec.eval(f.Default))
		}

		fields = append(fields, fe)
	}

	return &inputObjectExec{
		structType:    structType,
		defaultStruct: defaultStruct,
		fields:        fields,
	}, nil
}

func makeTypeAssertions(s *schema.Schema, typeName string, impls []*schema.Object, resolverType reflect.Type, typeRefMap map[typeRefMapKey]*typeRef) (map[string]*typeAssertExec, error) {
	typeAssertions := make(map[string]*typeAssertExec)
	for _, impl := range impls {
		methodIndex := findMethod(resolverType, "to"+impl.Name)
		if methodIndex == -1 {
			return nil, fmt.Errorf("%s does not resolve %q: missing method %q to convert to %q", resolverType, typeName, "to"+impl.Name, impl.Name)
		}
		a := &typeAssertExec{
			methodIndex: methodIndex,
		}
		if err := makeExec(&a.typeExec, s, impl, resolverType.Method(methodIndex).Type.Out(0), typeRefMap); err != nil {
			return nil, err
		}
		typeAssertions[impl.Name] = a
	}
	return typeAssertions, nil
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

func (r *request) addError(err *errors.GraphQLError) {
	r.mu.Lock()
	r.errs = append(r.errs, err)
	r.mu.Unlock()
}

func (r *request) handlePanic() {
	if err := recover(); err != nil {
		execErr := errors.Errorf("graphql: panic occured: %v", err)
		r.addError(execErr)

		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		log.Printf("%s\n%s", execErr, buf)
	}
}

func (e *Exec) Exec(ctx context.Context, document *query.Document, variables map[string]interface{}, op *query.Operation) (interface{}, []*errors.GraphQLError) {
	r := &request{
		ctx:    ctx,
		doc:    document,
		vars:   variables,
		schema: e.schema,
	}

	var opExec iExec
	switch op.Type {
	case query.Query:
		opExec = e.queryExec
	case query.Mutation:
		opExec = e.mutationExec
	}

	data := func() interface{} {
		defer r.handlePanic()
		return opExec.exec(r, op.SelSet, e.resolver, op.Type == query.Mutation)
	}()

	return data, r.errs
}

type iExec interface {
	exec(r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{}
}

type scalarExec struct{}

func (e *scalarExec) exec(r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{} {
	return resolver.Interface()
}

type listExec struct {
	elem    iExec
	nonNull bool
}

func (e *listExec) exec(r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{} {
	if !e.nonNull {
		if resolver.IsNil() {
			return nil
		}
		resolver = resolver.Elem()
	}
	l := make([]interface{}, resolver.Len())
	var wg sync.WaitGroup
	for i := range l {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer r.handlePanic()
			l[i] = e.elem.exec(r, selSet, resolver.Index(i), false)
		}(i)
	}
	wg.Wait()
	return l
}

type objectExec struct {
	name           string
	fields         map[string]*fieldExec
	typeAssertions map[string]*typeAssertExec
	nonNull        bool
}

type addResultFn func(key string, value interface{})

func (e *objectExec) exec(r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{} {
	if resolver.IsNil() {
		if e.nonNull {
			r.addError(errors.Errorf("got nil for non-null %q", e.name))
		}
		return nil
	}
	var mu sync.Mutex
	results := make(map[string]interface{})
	addResult := func(key string, value interface{}) {
		mu.Lock()
		results[key] = value
		mu.Unlock()
	}
	e.execSelectionSet(r, selSet, resolver, addResult, serially)
	return results
}

func (e *objectExec) execSelectionSet(r *request, selSet *query.SelectionSet, resolver reflect.Value, addResult addResultFn, serially bool) {
	var wg sync.WaitGroup
	for _, sel := range selSet.Selections {
		execSel := func(f func()) {
			if serially {
				defer r.handlePanic()
				f()
				return
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer r.handlePanic()
				f()
			}()
		}

		switch sel := sel.(type) {
		case *query.Field:
			if !skipByDirective(r, sel.Directives) {
				f := sel
				execSel(func() {
					switch f.Name {
					case "__typename":
						for name, a := range e.typeAssertions {
							out := resolver.Method(a.methodIndex).Call(nil)
							if out[1].Bool() {
								addResult(f.Alias, name)
								return
							}
						}

					case "__schema":
						addResult(f.Alias, introspectSchema(r, f.SelSet))

					case "__type":
						addResult(f.Alias, introspectType(r, f.Arguments["name"].Eval(r.vars).(string), f.SelSet))

					default:
						fe, ok := e.fields[f.Name]
						if !ok {
							panic(fmt.Errorf("%q has no field %q", e.name, f.Name)) // TODO proper error handling
						}
						fe.execField(r, f, resolver, addResult)
					}
				})
			}

		case *query.FragmentSpread:
			if !skipByDirective(r, sel.Directives) {
				fs := sel
				execSel(func() {
					frag, ok := r.doc.Fragments[fs.Name]
					if !ok {
						panic(fmt.Errorf("fragment %q not found", fs.Name)) // TODO proper error handling
					}
					e.execFragment(r, &frag.Fragment, resolver, addResult)
				})
			}

		case *query.InlineFragment:
			if !skipByDirective(r, sel.Directives) {
				frag := sel
				execSel(func() {
					e.execFragment(r, &frag.Fragment, resolver, addResult)
				})
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
		a.typeExec.(*objectExec).execSelectionSet(r, frag.SelSet, out[0], addResult, false)
		return
	}
	e.execSelectionSet(r, frag.SelSet, resolver, addResult, false)
}

type fieldExec struct {
	field       *schema.Field
	methodIndex int
	hasContext  bool
	argsExec    *inputObjectExec
	hasError    bool
	valueExec   iExec
}

func (e *fieldExec) execField(r *request, f *query.Field, resolver reflect.Value, addResult addResultFn) {
	var in []reflect.Value

	if e.hasContext {
		in = append(in, reflect.ValueOf(r.ctx))
	}

	if e.argsExec != nil {
		values := make(map[string]interface{})
		for name, arg := range f.Arguments {
			values[name] = arg.Eval(r.vars)
		}
		in = append(in, e.argsExec.eval(values))
	}

	m := resolver.Method(e.methodIndex)
	out := m.Call(in)
	if e.hasError && !out[1].IsNil() {
		err := out[1].Interface().(error)
		r.addError(errors.Errorf("%s", err))
		addResult(f.Alias, nil) // TODO handle non-nil
		return
	}
	addResult(f.Alias, e.valueExec.exec(r, f.SelSet, out[0], false))
}

type typeAssertExec struct {
	methodIndex int
	typeExec    iExec
}

type inputObjectExec struct {
	structType    reflect.Type
	defaultStruct reflect.Value
	fields        []*inputFieldExec
}

type inputExec interface {
	eval(value interface{}) reflect.Value
}

type inputFieldExec struct {
	name       string
	fieldIndex []int
	exec       inputExec
}

func (e *inputObjectExec) eval(value interface{}) reflect.Value {
	values := value.(map[string]interface{})
	v := reflect.New(e.structType)
	v.Elem().Set(e.defaultStruct)
	for _, f := range e.fields {
		if value, ok := values[f.name]; ok {
			v.Elem().FieldByIndex(f.fieldIndex).Set(f.exec.eval(value))
		}
	}
	return v
}

type scalarInputExec struct {
	scalar  *schema.Scalar
	nonNull bool
}

func (e *scalarInputExec) eval(value interface{}) reflect.Value {
	var v reflect.Value
	switch e.scalar.Name {
	case "Int":
		v = reflect.ValueOf(int32(value.(int)))
	default:
		v = reflect.ValueOf(value)
	}
	if !e.nonNull {
		p := reflect.New(v.Type())
		p.Elem().Set(v)
		return p
	}
	return v
}

func skipByDirective(r *request, d map[string]*query.Directive) bool {
	if skip, ok := d["skip"]; ok {
		if skip.Arguments["if"].Eval(r.vars).(bool) {
			return true
		}
	}
	if include, ok := d["include"]; ok {
		if !include.Arguments["if"].Eval(r.vars).(bool) {
			return true
		}
	}
	return false
}
