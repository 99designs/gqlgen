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
	"github.com/neelance/graphql-go/internal/lexer"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/trace"
)

type Exec struct {
	queryExec    iExec
	mutationExec iExec
	schema       *schema.Schema
	resolver     reflect.Value
}

func Make(s *schema.Schema, resolver interface{}) (*Exec, error) {
	b := newExecBuilder(s)

	var queryExec, mutationExec iExec

	if t, ok := s.EntryPoints["query"]; ok {
		if err := b.assignExec(&queryExec, t, reflect.TypeOf(resolver)); err != nil {
			return nil, err
		}
	}

	if t, ok := s.EntryPoints["mutation"]; ok {
		if err := b.assignExec(&mutationExec, t, reflect.TypeOf(resolver)); err != nil {
			return nil, err
		}
	}

	if err := b.finish(); err != nil {
		return nil, err
	}

	return &Exec{
		schema:       s,
		resolver:     reflect.ValueOf(resolver),
		queryExec:    queryExec,
		mutationExec: mutationExec,
	}, nil
}

type execBuilder struct {
	schema        *schema.Schema
	execMap       map[typePair]*execMapEntry
	packerMap     map[typePair]*packerMapEntry
	structPackers []*structPacker
}

type typePair struct {
	graphQLType  common.Type
	resolverType reflect.Type
}

type execMapEntry struct {
	exec    iExec
	targets []*iExec
}

type packerMapEntry struct {
	packer  packer
	targets []*packer
}

func newExecBuilder(s *schema.Schema) *execBuilder {
	return &execBuilder{
		schema:    s,
		execMap:   make(map[typePair]*execMapEntry),
		packerMap: make(map[typePair]*packerMapEntry),
	}
}

func (b *execBuilder) finish() error {
	for _, entry := range b.execMap {
		for _, target := range entry.targets {
			*target = entry.exec
		}
	}

	for _, entry := range b.packerMap {
		for _, target := range entry.targets {
			*target = entry.packer
		}
	}

	for _, p := range b.structPackers {
		p.defaultStruct = reflect.New(p.structType).Elem()
		for _, f := range p.fields {
			if defaultVal := f.field.Default; defaultVal != nil {
				v, err := f.fieldPacker.pack(nil, defaultVal.Value)
				if err != nil {
					return err
				}
				p.defaultStruct.FieldByIndex(f.fieldIndex).Set(v)
			}
		}
	}

	return nil
}

func (b *execBuilder) assignExec(target *iExec, t common.Type, resolverType reflect.Type) error {
	k := typePair{t, resolverType}
	ref, ok := b.execMap[k]
	if !ok {
		ref = &execMapEntry{}
		b.execMap[k] = ref
		var err error
		ref.exec, err = b.makeExec(t, resolverType)
		if err != nil {
			return err
		}
	}
	ref.targets = append(ref.targets, target)
	return nil
}

func (b *execBuilder) makeExec(t common.Type, resolverType reflect.Type) (iExec, error) {
	var nonNull bool
	t, nonNull = unwrapNonNull(t)

	switch t := t.(type) {
	case *schema.Object:
		return b.makeObjectExec(t.Name, t.Fields, nil, nonNull, resolverType)

	case *schema.Interface:
		return b.makeObjectExec(t.Name, t.Fields, t.PossibleTypes, nonNull, resolverType)

	case *schema.Union:
		return b.makeObjectExec(t.Name, nil, t.PossibleTypes, nonNull, resolverType)
	}

	if !nonNull {
		if resolverType.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("%s is not a pointer", resolverType)
		}
		resolverType = resolverType.Elem()
	}

	switch t := t.(type) {
	case *schema.Scalar:
		return makeScalarExec(t, resolverType)

	case *schema.Enum:
		return &scalarExec{}, nil

	case *common.List:
		if resolverType.Kind() != reflect.Slice {
			return nil, fmt.Errorf("%s is not a slice", resolverType)
		}
		e := &listExec{nonNull: nonNull}
		if err := b.assignExec(&e.elem, t.OfType, resolverType.Elem()); err != nil {
			return nil, err
		}
		return e, nil

	default:
		panic("invalid type")
	}
}

func makeScalarExec(t *schema.Scalar, resolverType reflect.Type) (iExec, error) {
	implementsType := false
	switch r := reflect.New(resolverType).Interface().(type) {
	case *int32:
		implementsType = (t.Name == "Int")
	case *float64:
		implementsType = (t.Name == "Float")
	case *string:
		implementsType = (t.Name == "String")
	case *bool:
		implementsType = (t.Name == "Boolean")
	case Unmarshaler:
		implementsType = r.ImplementsGraphQLType(t.Name)
	}
	if !implementsType {
		return nil, fmt.Errorf("can not use %s as %s", resolverType, t.Name)
	}
	return &scalarExec{}, nil
}

func (b *execBuilder) makeObjectExec(typeName string, fields schema.FieldList, possibleTypes []*schema.Object, nonNull bool, resolverType reflect.Type) (*objectExec, error) {
	if !nonNull {
		if resolverType.Kind() != reflect.Ptr && resolverType.Kind() != reflect.Interface {
			return nil, fmt.Errorf("%s is not a pointer or interface", resolverType)
		}
	}

	methodHasReceiver := resolverType.Kind() != reflect.Interface
	fieldExecs := map[string]fieldExec{
		"__typename": typenameFieldExec,
		"__schema":   schemaFieldExec,
		"__type":     typeFieldExec,
	}

	for _, f := range fields {
		methodIndex := findMethod(resolverType, f.Name)
		if methodIndex == -1 {
			hint := ""
			if findMethod(reflect.PtrTo(resolverType), f.Name) != -1 {
				hint = " (hint: the method exists on the pointer type)"
			}
			return nil, fmt.Errorf("%s does not resolve %q: missing method for field %q%s", resolverType, typeName, f.Name, hint)
		}

		m := resolverType.Method(methodIndex)
		fe, err := b.makeFieldExec(typeName, f, m, methodIndex, methodHasReceiver)
		if err != nil {
			return nil, fmt.Errorf("%s\n\treturned by (%s).%s", err, resolverType, m.Name)
		}
		fieldExecs[f.Name] = fe
	}

	typeAssertions := make(map[string]*typeAssertExec)
	for _, impl := range possibleTypes {
		methodIndex := findMethod(resolverType, "to"+impl.Name)
		if methodIndex == -1 {
			return nil, fmt.Errorf("%s does not resolve %q: missing method %q to convert to %q", resolverType, typeName, "to"+impl.Name, impl.Name)
		}
		a := &typeAssertExec{
			methodIndex: methodIndex,
		}
		if err := b.assignExec(&a.typeExec, impl, resolverType.Method(methodIndex).Type.Out(0)); err != nil {
			return nil, err
		}
		typeAssertions[impl.Name] = a
	}

	return &objectExec{
		name:           typeName,
		fields:         fieldExecs,
		typeAssertions: typeAssertions,
		nonNull:        nonNull,
	}, nil
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (b *execBuilder) makeFieldExec(typeName string, f *schema.Field, m reflect.Method, methodIndex int, methodHasReceiver bool) (*normalFieldExec, error) {
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
	if len(f.Args) > 0 {
		if len(in) == 0 {
			return nil, fmt.Errorf("must have parameter for field arguments")
		}
		var err error
		argsPacker, err = b.makeStructPacker(f.Args, in[0])
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

	fe := &normalFieldExec{
		typeName:    typeName,
		field:       f,
		methodIndex: methodIndex,
		hasContext:  hasContext,
		argsPacker:  argsPacker,
		hasError:    hasError,
		trivial:     !hasContext && argsPacker == nil && !hasError,
		traceLabel:  fmt.Sprintf("GraphQL field: %s.%s", typeName, f.Name),
	}
	if err := b.assignExec(&fe.valueExec, f.Type, m.Type.Out(0)); err != nil {
		return nil, err
	}
	return fe, nil
}

func findMethod(t reflect.Type, name string) int {
	for i := 0; i < t.NumMethod(); i++ {
		if strings.EqualFold(name, t.Method(i).Name) {
			return i
		}
	}
	return -1
}

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
		opExec.execSelectionSet(ctx, r, op.SelSet, e.resolver, results, serially)
	}()
	r.wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, []*errors.QueryError{errors.Errorf("%s", err)}
	}

	return results, r.errs
}

type iExec interface {
	exec(ctx context.Context, r *Request, selSet *query.SelectionSet, resolver reflect.Value) interface{}
}

type scalarExec struct{}

func (e *scalarExec) exec(ctx context.Context, r *Request, selSet *query.SelectionSet, resolver reflect.Value) interface{} {
	return resolver.Interface()
}

type listExec struct {
	elem    iExec
	nonNull bool
}

func (e *listExec) exec(ctx context.Context, r *Request, selSet *query.SelectionSet, resolver reflect.Value) interface{} {
	if !e.nonNull {
		if resolver.IsNil() {
			return nil
		}
		resolver = resolver.Elem()
	}
	l := make([]interface{}, resolver.Len())
	for i := range l {
		l[i] = e.elem.exec(ctx, r, selSet, resolver.Index(i))
	}
	return l
}

type objectExec struct {
	name           string
	fields         map[string]fieldExec
	typeAssertions map[string]*typeAssertExec
	nonNull        bool
}

type fieldExec interface {
	execField(ctx context.Context, r *Request, e *objectExec, f *query.Field, resolver reflect.Value) interface{}
}

type normalFieldExec struct {
	typeName    string
	field       *schema.Field
	methodIndex int
	hasContext  bool
	argsPacker  *structPacker
	hasError    bool
	trivial     bool
	valueExec   iExec
	traceLabel  string
}

type metaFieldExec func(ctx context.Context, r *Request, e *objectExec, f *query.Field, resolver reflect.Value) interface{}

func (fe metaFieldExec) execField(ctx context.Context, r *Request, e *objectExec, f *query.Field, resolver reflect.Value) interface{} {
	return fe(ctx, r, e, f, resolver)
}

var typenameFieldExec = metaFieldExec(func(ctx context.Context, r *Request, e *objectExec, f *query.Field, resolver reflect.Value) interface{} {
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
})

var schemaFieldExec = metaFieldExec(func(ctx context.Context, r *Request, e *objectExec, f *query.Field, resolver reflect.Value) interface{} {
	return introspectSchema(ctx, r, f.SelSet)
})

var typeFieldExec = metaFieldExec(func(ctx context.Context, r *Request, e *objectExec, f *query.Field, resolver reflect.Value) interface{} {
	p := valuePacker{valueType: reflect.TypeOf("")}
	v, err := p.pack(r, r.resolveVar(f.Arguments.MustGet("name").Value))
	if err != nil {
		r.addError(errors.Errorf("%s", err))
		return nil
	}
	return introspectType(ctx, r, v.String(), f.SelSet)
})

func (e *objectExec) exec(ctx context.Context, r *Request, selSet *query.SelectionSet, resolver reflect.Value) interface{} {
	if resolver.IsNil() {
		if e.nonNull {
			r.addError(errors.Errorf("got nil for non-null %q", e.name))
		}
		return nil
	}
	results := make(map[string]interface{})
	e.execSelectionSet(ctx, r, selSet, resolver, results, false)
	return results
}

func (e *objectExec) execSelectionSet(ctx context.Context, r *Request, selSet *query.SelectionSet, resolver reflect.Value, results map[string]interface{}, serially bool) {
	for _, sel := range selSet.Selections {
		switch sel := sel.(type) {
		case *query.Field:
			field := sel
			if skipByDirective(r, field.Directives) {
				continue
			}

			results[field.Alias.Name] = e.fields[field.Name.Name].execField(ctx, r, e, field, resolver)
			if serially {
				r.wg.Wait()
			}

		case *query.InlineFragment:
			frag := sel
			if skipByDirective(r, frag.Directives) {
				continue
			}
			e.execFragment(ctx, r, &frag.Fragment, resolver, results)

		case *query.FragmentSpread:
			spread := sel
			if skipByDirective(r, spread.Directives) {
				continue
			}
			e.execFragment(ctx, r, &r.Doc.Fragments.Get(spread.Name.Name).Fragment, resolver, results)

		default:
			panic("invalid type")
		}
	}
}

func (fe *normalFieldExec) execField(ctx context.Context, r *Request, e *objectExec, f *query.Field, resolver reflect.Value) interface{} {
	var args map[string]interface{}
	var packedArgs reflect.Value
	if fe.argsPacker != nil {
		args = make(map[string]interface{})
		for _, arg := range f.Arguments {
			args[arg.Name.Name] = arg.Value.Value
		}
		var err error
		packedArgs, err = fe.argsPacker.pack(r, args)
		if err != nil {
			r.addError(errors.Errorf("%s", err))
			return nil
		}
	}

	do := func(applyLimiter bool) interface{} {
		if applyLimiter {
			r.Limiter <- struct{}{}
		}

		var result reflect.Value
		var err *errors.QueryError

		traceCtx, finish := r.Tracer.TraceField(ctx, fe.traceLabel, fe.typeName, fe.field.Name, fe.trivial, args)
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
				in = append(in, packedArgs)
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
			<-r.Limiter
		}

		if err != nil {
			r.addError(err)
			return nil // TODO handle non-nil
		}

		return fe.valueExec.exec(traceCtx, r, f.SelSet, result)
	}

	if fe.trivial {
		return do(false)
	}

	result := new(interface{})
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		*result = do(true)
	}()
	return result
}

func (e *objectExec) execFragment(ctx context.Context, r *Request, frag *query.Fragment, resolver reflect.Value, results map[string]interface{}) {
	if frag.On.Name != "" && frag.On.Name != e.name {
		a, ok := e.typeAssertions[frag.On.Name]
		if !ok {
			panic(fmt.Errorf("%q does not implement %q", frag.On, e.name)) // TODO proper error handling
		}
		out := resolver.Method(a.methodIndex).Call(nil)
		if !out[1].Bool() {
			return
		}
		a.typeExec.(*objectExec).execSelectionSet(ctx, r, frag.SelSet, out[0], results, false)
		return
	}
	e.execSelectionSet(ctx, r, frag.SelSet, resolver, results, false)
}

type typeAssertExec struct {
	methodIndex int
	typeExec    iExec
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

func unwrapNonNull(t common.Type) (common.Type, bool) {
	if nn, ok := t.(*common.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}
