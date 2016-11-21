package exec

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

// keep in sync with main package
const OpenTracingTagType = "graphql.type"
const OpenTracingTagField = "graphql.field"
const OpenTracingTagTrivial = "graphql.trivial"
const OpenTracingTagArgsPrefix = "graphql.args."
const OpenTracingTagError = "graphql.error"

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

func makeExec2(s *schema.Schema, t common.Type, resolverType reflect.Type, typeRefMap map[typeRefMapKey]*typeRef) (iExec, error) {
	var nonNull bool
	t, nonNull = unwrapNonNull(t)

	if !nonNull {
		if resolverType.Kind() != reflect.Ptr && resolverType.Kind() != reflect.Interface {
			return nil, fmt.Errorf("%s is not a pointer or interface", resolverType)
		}
	}

	switch t := t.(type) {
	case *scalar:
		if !nonNull {
			resolverType = resolverType.Elem()
		}
		scalarType := t.reflectType
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
			name:       t.Name,
			fields:     fields,
			isConcrete: true,
			nonNull:    nonNull,
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
		fe, err := makeFieldExec(s, typeName, f, m, methodIndex, methodHasReceiver, typeRefMap)
		if err != nil {
			return nil, fmt.Errorf("method %q of %s: %s", m.Name, resolverType, err)
		}
		fieldExecs[name] = fe
	}
	return fieldExecs, nil
}

func makeFieldExec(s *schema.Schema, typeName string, f *schema.Field, m reflect.Method, methodIndex int, methodHasReceiver bool, typeRefMap map[typeRefMapKey]*typeRef) (*fieldExec, error) {
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

	var argsPacker *structPacker
	if len(f.Args.Fields) > 0 {
		if len(in) == 0 {
			return nil, fmt.Errorf("must have parameter for field arguments")
		}
		var err error
		argsPacker, err = makeStructPacker(s, &f.Args, in[0])
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
		typeName:    typeName,
		field:       f,
		methodIndex: methodIndex,
		hasContext:  hasContext,
		argsPacker:  argsPacker,
		hasError:    hasError,
	}
	if err := makeExec(&fe.valueExec, s, f.Type, m.Type.Out(0), typeRefMap); err != nil {
		return nil, err
	}
	return fe, nil
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
	doc    *query.Document
	vars   map[string]interface{}
	schema *schema.Schema
	mu     sync.Mutex
	errs   []*errors.QueryError
}

func (r *request) addError(err *errors.QueryError) {
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

func ExecuteRequest(ctx context.Context, e *Exec, document *query.Document, operationName string, variables map[string]interface{}) (interface{}, []*errors.QueryError) {
	op, err := getOperation(document, operationName)
	if err != nil {
		return nil, []*errors.QueryError{errors.Errorf("%s", err)}
	}

	coercedVariables, err := coerceMap(nil, &op.Vars, variables)
	if err != nil {
		return nil, []*errors.QueryError{errors.Errorf("%s", err)}
	}

	r := &request{
		doc:    document,
		vars:   coercedVariables,
		schema: e.schema,
	}

	var opExec iExec
	var serially bool
	switch op.Type {
	case query.Query:
		opExec = e.queryExec
		serially = false
	case query.Mutation:
		opExec = e.mutationExec
		serially = true
	}

	data := func() interface{} {
		defer r.handlePanic()
		return opExec.exec(ctx, r, op.SelSet, e.resolver, serially)
	}()

	return data, r.errs
}

func getOperation(document *query.Document, operationName string) (*query.Operation, error) {
	if len(document.Operations) == 0 {
		return nil, fmt.Errorf("no operations in query document")
	}

	if operationName == "" {
		if len(document.Operations) > 1 {
			return nil, fmt.Errorf("more than one operation in query document and no operation name given")
		}
		for _, op := range document.Operations {
			return op, nil // return the one and only operation
		}
	}

	op, ok := document.Operations[operationName]
	if !ok {
		return nil, fmt.Errorf("no operation with name %q", operationName)
	}
	return op, nil
}

type iExec interface {
	exec(ctx context.Context, r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{}
}

type scalarExec struct{}

func (e *scalarExec) exec(ctx context.Context, r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{} {
	return resolver.Interface()
}

type listExec struct {
	elem    iExec
	nonNull bool
}

func (e *listExec) exec(ctx context.Context, r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{} {
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
			l[i] = e.elem.exec(ctx, r, selSet, resolver.Index(i), false)
		}(i)
	}
	wg.Wait()
	return l
}

type objectExec struct {
	name           string
	fields         map[string]*fieldExec
	isConcrete     bool
	typeAssertions map[string]*typeAssertExec
	nonNull        bool
}

type addResultFn func(key string, value interface{})

func (e *objectExec) exec(ctx context.Context, r *request, selSet *query.SelectionSet, resolver reflect.Value, serially bool) interface{} {
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
	e.execSelectionSet(ctx, r, selSet, resolver, addResult, serially)
	return results
}

func (e *objectExec) execSelectionSet(ctx context.Context, r *request, selSet *query.SelectionSet, resolver reflect.Value, addResult addResultFn, serially bool) {
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
						if e.isConcrete {
							addResult(f.Alias, e.name)
							return
						}

						for name, a := range e.typeAssertions {
							out := resolver.Method(a.methodIndex).Call(nil)
							if out[1].Bool() {
								addResult(f.Alias, name)
								return
							}
						}

					case "__schema":
						addResult(f.Alias, introspectSchema(ctx, r, f.SelSet))

					case "__type":
						v, err := coerceValue(r, stringScalar, f.Arguments["name"])
						if err != nil {
							r.addError(errors.Errorf("%s", err))
							addResult(f.Alias, nil)
							return
						}
						addResult(f.Alias, introspectType(ctx, r, v.(string), f.SelSet))

					default:
						fe, ok := e.fields[f.Name]
						if !ok {
							panic(fmt.Errorf("%q has no field %q", e.name, f.Name)) // TODO proper error handling
						}
						fe.execField(ctx, r, f, resolver, addResult)
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
					e.execFragment(ctx, r, &frag.Fragment, resolver, addResult)
				})
			}

		case *query.InlineFragment:
			if !skipByDirective(r, sel.Directives) {
				frag := sel
				execSel(func() {
					e.execFragment(ctx, r, &frag.Fragment, resolver, addResult)
				})
			}

		default:
			panic("invalid type")
		}
	}
	wg.Wait()
}

func (e *objectExec) execFragment(ctx context.Context, r *request, frag *query.Fragment, resolver reflect.Value, addResult addResultFn) {
	if frag.On != "" && frag.On != e.name {
		a, ok := e.typeAssertions[frag.On]
		if !ok {
			panic(fmt.Errorf("%q does not implement %q", frag.On, e.name)) // TODO proper error handling
		}
		out := resolver.Method(a.methodIndex).Call(nil)
		if !out[1].Bool() {
			return
		}
		a.typeExec.(*objectExec).execSelectionSet(ctx, r, frag.SelSet, out[0], addResult, false)
		return
	}
	e.execSelectionSet(ctx, r, frag.SelSet, resolver, addResult, false)
}

type fieldExec struct {
	typeName    string
	field       *schema.Field
	methodIndex int
	hasContext  bool
	argsPacker  *structPacker
	hasError    bool
	valueExec   iExec
}

func (e *fieldExec) execField(ctx context.Context, r *request, f *query.Field, resolver reflect.Value, addResult addResultFn) {
	span, spanCtx := opentracing.StartSpanFromContext(ctx, fmt.Sprintf("GraphQL field: %s.%s", e.typeName, e.field.Name))
	defer span.Finish()
	span.SetTag(OpenTracingTagType, e.typeName)
	span.SetTag(OpenTracingTagField, e.field.Name)
	if !e.hasContext && e.argsPacker == nil && !e.hasError {
		span.SetTag(OpenTracingTagTrivial, true)
	}

	result, err := e.execField2(spanCtx, r, f, resolver, span)

	if err != nil {
		r.addError(errors.Errorf("%s", err))
		addResult(f.Alias, nil) // TODO handle non-nil

		ext.Error.Set(span, true)
		span.SetTag(OpenTracingTagError, err)
		return
	}

	addResult(f.Alias, result)
}

func (e *fieldExec) execField2(ctx context.Context, r *request, f *query.Field, resolver reflect.Value, span opentracing.Span) (interface{}, error) {
	var in []reflect.Value

	if e.hasContext {
		in = append(in, reflect.ValueOf(ctx))
	}

	if e.argsPacker != nil {
		values := make(map[string]interface{})
		for name, arg := range f.Arguments {
			v, err := coerceValue(r, e.field.Args.Fields[name].Type, arg)
			if err != nil {
				return nil, err
			}
			values[name] = v
			span.SetTag(OpenTracingTagArgsPrefix+name, v)
		}
		in = append(in, e.argsPacker.pack(values))
	}

	m := resolver.Method(e.methodIndex)
	out := m.Call(in)
	if e.hasError && !out[1].IsNil() {
		return nil, out[1].Interface().(error)
	}

	return e.valueExec.exec(ctx, r, f.SelSet, out[0], false), nil
}

type typeAssertExec struct {
	methodIndex int
	typeExec    iExec
}

func coerceMap(r *request, io *common.InputMap, m map[string]interface{}) (map[string]interface{}, error) {
	coerced := make(map[string]interface{})
	for _, iv := range io.Fields {
		value, ok := m[iv.Name]
		if !ok {
			if _, nonNull := unwrapNonNull(iv.Type); !nonNull {
				continue
			}
			if iv.Default == nil {
				return nil, errors.Errorf("missing %q", iv.Name)
			}
			value = iv.Default
		}
		c, err := coerceValue(r, iv.Type, value)
		if err != nil {
			return nil, err
		}
		coerced[iv.Name] = c
	}
	return coerced, nil
}

func coerceValue(r *request, typ common.Type, value interface{}) (interface{}, error) {
	if v, ok := value.(common.Variable); ok {
		return r.vars[string(v)], nil
	}

	t, nonNull := unwrapNonNull(typ)
	if !nonNull && value == nil {
		return nil, nil
	}
	switch t := t.(type) {
	case *scalar:
		v, err := t.coerceInput(value)
		if v == nil {
			return nil, fmt.Errorf("could not convert %#v (%T) to %s: %s", value, value, t.name, err)
		}
		return v, nil
	case *schema.InputObject:
		return coerceMap(r, &t.InputMap, value.(map[string]interface{}))
	case *common.List:
		list := value.([]interface{})
		coerced := make([]interface{}, len(list))
		for i, entry := range list {
			c, err := coerceValue(r, t.OfType, entry)
			if err != nil {
				return nil, err
			}
			coerced[i] = c
		}
		return coerced, nil
	}
	return value, nil
}

func skipByDirective(r *request, d map[string]*query.Directive) bool {
	if skip, ok := d["skip"]; ok {
		v, err := coerceValue(r, booleanScalar, skip.Arguments["if"])
		if err != nil {
			r.addError(errors.Errorf("%s", err))
		}
		if err == nil && v.(bool) {
			return true
		}
	}

	if include, ok := d["include"]; ok {
		v, err := coerceValue(r, booleanScalar, include.Arguments["if"])
		if err != nil {
			r.addError(errors.Errorf("%s", err))
		}
		if err == nil && !v.(bool) {
			return true
		}
	}

	return false
}

func unwrapNonNull(t common.Type) (common.Type, bool) {
	if nn, ok := t.(*common.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}
