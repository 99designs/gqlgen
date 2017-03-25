package exec

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/schema"
)

type Exec struct {
	queryExec    iExec
	mutationExec iExec
	schema       *schema.Schema
	resolver     reflect.Value
}

type iExec interface {
	exec(ctx context.Context, sels []appliedSelection, resolver reflect.Value) interface{}
}

type objectExec struct {
	name           string
	fields         map[string]*fieldExec
	typeAssertions map[string]*typeAssertExec
	nonNull        bool
}

type fieldExec struct {
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

type typeAssertExec struct {
	methodIndex int
	typeExec    iExec
}

type listExec struct {
	elem    iExec
	nonNull bool
}

type scalarExec struct{}

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

	fieldExecs := make(map[string]*fieldExec)
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

func (b *execBuilder) makeFieldExec(typeName string, f *schema.Field, m reflect.Method, methodIndex int, methodHasReceiver bool) (*fieldExec, error) {
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

	fe := &fieldExec{
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

func unwrapNonNull(t common.Type) (common.Type, bool) {
	if nn, ok := t.(*common.NonNull); ok {
		return nn.OfType, true
	}
	return t, false
}
