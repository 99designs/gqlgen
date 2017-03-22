package exec

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/lexer"
	"github.com/neelance/graphql-go/internal/schema"
)

type packer interface {
	pack(r *request, value interface{}) (reflect.Value, error)
}

func (b *execBuilder) assignPacker(target *packer, schemaType common.Type, reflectType reflect.Type) error {
	k := typePair{schemaType, reflectType}
	ref, ok := b.packerMap[k]
	if !ok {
		ref = &packerMapEntry{}
		b.packerMap[k] = ref
		var err error
		ref.packer, err = b.makePacker(schemaType, reflectType)
		if err != nil {
			return err
		}
	}
	ref.targets = append(ref.targets, target)
	return nil
}

func (b *execBuilder) makePacker(schemaType common.Type, reflectType reflect.Type) (packer, error) {
	t, nonNull := unwrapNonNull(schemaType)
	if !nonNull {
		if reflectType.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("%s is not a pointer", reflectType)
		}
		elemType := reflectType.Elem()
		addPtr := true
		if _, ok := t.(*schema.InputObject); ok {
			elemType = reflectType // keep pointer for input objects
			addPtr = false
		}
		elem, err := b.makeNonNullPacker(t, elemType)
		if err != nil {
			return nil, err
		}
		return &nullPacker{
			elemPacker: elem,
			valueType:  reflectType,
			addPtr:     addPtr,
		}, nil
	}

	return b.makeNonNullPacker(t, reflectType)
}

func (b *execBuilder) makeNonNullPacker(schemaType common.Type, reflectType reflect.Type) (packer, error) {
	if u, ok := reflect.New(reflectType).Interface().(Unmarshaler); ok {
		if !u.ImplementsGraphQLType(schemaType.String()) {
			return nil, fmt.Errorf("can not unmarshal %s into %s", schemaType, reflectType)
		}
		return &unmarshalerPacker{
			valueType: reflectType,
		}, nil
	}

	switch t := schemaType.(type) {
	case *schema.Scalar:
		return &valuePacker{
			valueType: reflectType,
		}, nil

	case *schema.Enum:
		want := reflect.TypeOf("")
		if reflectType != want {
			return nil, fmt.Errorf("wrong type, expected %s", want)
		}
		return &valuePacker{
			valueType: reflectType,
		}, nil

	case *schema.InputObject:
		e, err := b.makeStructPacker(t.Values, reflectType)
		if err != nil {
			return nil, err
		}
		return e, nil

	case *common.List:
		if reflectType.Kind() != reflect.Slice {
			return nil, fmt.Errorf("expected slice, got %s", reflectType)
		}
		p := &listPacker{
			sliceType: reflectType,
		}
		if err := b.assignPacker(&p.elem, t.OfType, reflectType.Elem()); err != nil {
			return nil, err
		}
		return p, nil

	case *schema.Object, *schema.Interface, *schema.Union:
		return nil, fmt.Errorf("type of kind %s can not be used as input", t.Kind())

	default:
		panic("unreachable")
	}
}

func (b *execBuilder) makeStructPacker(values common.InputValueList, typ reflect.Type) (*structPacker, error) {
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected pointer to struct, got %s", typ)
	}
	structType := typ.Elem()

	var fields []*structPackerField
	for _, v := range values {
		fe := &structPackerField{field: v}

		sf, ok := structType.FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, v.Name.Name) })
		if !ok {
			return nil, fmt.Errorf("missing argument %q", v.Name)
		}
		if sf.PkgPath != "" {
			return nil, fmt.Errorf("field %q must be exported", sf.Name)
		}
		fe.fieldIndex = sf.Index

		ft := v.Type
		if v.Default != nil {
			ft, _ = unwrapNonNull(ft)
			ft = &common.NonNull{OfType: ft}
		}

		if err := b.assignPacker(&fe.fieldPacker, ft, sf.Type); err != nil {
			return nil, fmt.Errorf("field %q: %s", sf.Name, err)
		}

		fields = append(fields, fe)
	}

	p := &structPacker{
		structType: structType,
		fields:     fields,
	}
	b.structPackers = append(b.structPackers, p)
	return p, nil
}

type structPacker struct {
	structType    reflect.Type
	defaultStruct reflect.Value
	fields        []*structPackerField
}

type structPackerField struct {
	field       *common.InputValue
	fieldIndex  []int
	fieldPacker packer
}

func (p *structPacker) pack(r *request, value interface{}) (reflect.Value, error) {
	if value == nil {
		return reflect.Value{}, errors.Errorf("got null for non-null")
	}

	values := value.(map[string]interface{})
	v := reflect.New(p.structType)
	v.Elem().Set(p.defaultStruct)
	for _, f := range p.fields {
		if value, ok := values[f.field.Name.Name]; ok {
			packed, err := f.fieldPacker.pack(r, r.resolveVar(value))
			if err != nil {
				return reflect.Value{}, err
			}
			v.Elem().FieldByIndex(f.fieldIndex).Set(packed)
		}
	}
	return v, nil
}

type listPacker struct {
	sliceType reflect.Type
	elem      packer
}

func (e *listPacker) pack(r *request, value interface{}) (reflect.Value, error) {
	list, ok := value.([]interface{})
	if !ok {
		list = []interface{}{value}
	}

	v := reflect.MakeSlice(e.sliceType, len(list), len(list))
	for i := range list {
		packed, err := e.elem.pack(r, r.resolveVar(list[i]))
		if err != nil {
			return reflect.Value{}, err
		}
		v.Index(i).Set(packed)
	}
	return v, nil
}

type nullPacker struct {
	elemPacker packer
	valueType  reflect.Type
	addPtr     bool
}

func (p *nullPacker) pack(r *request, value interface{}) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(p.valueType), nil
	}

	v, err := p.elemPacker.pack(r, value)
	if err != nil {
		return reflect.Value{}, err
	}

	if p.addPtr {
		ptr := reflect.New(p.valueType.Elem())
		ptr.Elem().Set(v)
		return ptr, nil
	}

	return v, nil
}

type valuePacker struct {
	valueType reflect.Type
}

func (p *valuePacker) pack(r *request, value interface{}) (reflect.Value, error) {
	if value == nil {
		return reflect.Value{}, errors.Errorf("got null for non-null")
	}

	if lit, ok := value.(*lexer.Literal); ok {
		value = common.UnmarshalLiteral(lit)
	}

	coerced, err := unmarshalInput(p.valueType, value)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("could not unmarshal %#v (%T) into %s: %s", value, value, p.valueType, err)
	}
	return reflect.ValueOf(coerced), nil
}

type unmarshalerPacker struct {
	valueType reflect.Type
}

func (p *unmarshalerPacker) pack(r *request, value interface{}) (reflect.Value, error) {
	if value == nil {
		return reflect.Value{}, errors.Errorf("got null for non-null")
	}

	if lit, ok := value.(*lexer.Literal); ok {
		value = common.UnmarshalLiteral(lit)
	}

	v := reflect.New(p.valueType)
	if err := v.Interface().(Unmarshaler).UnmarshalGraphQL(value); err != nil {
		return reflect.Value{}, err
	}
	return v.Elem(), nil
}

type Unmarshaler interface {
	ImplementsGraphQLType(name string) bool
	UnmarshalGraphQL(input interface{}) error
}

func unmarshalInput(typ reflect.Type, input interface{}) (interface{}, error) {
	if reflect.TypeOf(input) == typ {
		return input, nil
	}

	switch typ.Kind() {
	case reflect.Int32:
		switch input := input.(type) {
		case int:
			if input < math.MinInt32 || input > math.MaxInt32 {
				return nil, fmt.Errorf("not a 32-bit integer")
			}
			return int32(input), nil
		case float64:
			coerced := int32(input)
			if input < math.MinInt32 || input > math.MaxInt32 || float64(coerced) != input {
				return nil, fmt.Errorf("not a 32-bit integer")
			}
			return coerced, nil
		}

	case reflect.Float64:
		switch input := input.(type) {
		case int32:
			return float64(input), nil
		case int:
			return float64(input), nil
		}
	}

	return nil, fmt.Errorf("incompatible type")
}
