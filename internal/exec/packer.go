package exec

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/schema"
)

type packer interface {
	pack(r *request, value interface{}) (reflect.Value, error)
}

func makePacker(s *schema.Schema, schemaType common.Type, reflectType reflect.Type) (packer, error) {
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
		elem, err := makeNonNullPacker(s, t, elemType)
		if err != nil {
			return nil, err
		}
		return &nullPacker{
			elemPacker: elem,
			valueType:  reflectType,
			addPtr:     addPtr,
		}, nil
	}

	return makeNonNullPacker(s, t, reflectType)
}

func makeNonNullPacker(s *schema.Schema, schemaType common.Type, reflectType reflect.Type) (packer, error) {
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
		e, err := makeStructPacker(s, &t.InputMap, reflectType)
		if err != nil {
			return nil, err
		}
		return e, nil

	case *common.List:
		if reflectType.Kind() != reflect.Slice {
			return nil, fmt.Errorf("expected slice, got %s", reflectType)
		}
		elem, err := makePacker(s, t.OfType, reflectType.Elem())
		if err != nil {
			return nil, err
		}
		return &listPacker{
			sliceType: reflectType,
			elem:      elem,
		}, nil

	case *schema.Object, *schema.Interface, *schema.Union:
		return nil, fmt.Errorf("type of kind %s can not be used as input", t.Kind())

	default:
		panic("unreachable")
	}
}

func makeStructPacker(s *schema.Schema, obj *common.InputMap, typ reflect.Type) (*structPacker, error) {
	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected pointer to struct, got %s", typ)
	}
	structType := typ.Elem()

	var fields []*structPackerField
	defaultStruct := reflect.New(structType).Elem()
	for _, f := range obj.Fields {
		fe := &structPackerField{
			name: f.Name,
		}

		sf, ok := structType.FieldByNameFunc(func(n string) bool { return strings.EqualFold(n, f.Name) })
		if !ok {
			return nil, fmt.Errorf("missing argument %q", f.Name)
		}
		if sf.PkgPath != "" {
			return nil, fmt.Errorf("field %q must be exported", sf.Name)
		}
		fe.fieldIndex = sf.Index

		ft := f.Type
		if f.Default != nil {
			ft, _ = unwrapNonNull(ft)
			ft = &common.NonNull{OfType: ft}
		}

		p, err := makePacker(s, ft, sf.Type)
		if err != nil {
			return nil, fmt.Errorf("field %q: %s", sf.Name, err)
		}
		fe.fieldPacker = p

		if f.Default != nil {
			v, err := fe.fieldPacker.pack(nil, f.Default)
			if err != nil {
				return nil, err
			}
			defaultStruct.FieldByIndex(fe.fieldIndex).Set(v)
		}

		fields = append(fields, fe)
	}

	return &structPacker{
		structType:    structType,
		defaultStruct: defaultStruct,
		fields:        fields,
	}, nil
}

type structPacker struct {
	structType    reflect.Type
	defaultStruct reflect.Value
	fields        []*structPackerField
}

type structPackerField struct {
	name        string
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
		if value, ok := values[f.name]; ok {
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

var int32Type = reflect.TypeOf(int32(0))
var float64Type = reflect.TypeOf(float64(0))
var stringType = reflect.TypeOf("")
var boolType = reflect.TypeOf(false)

func unmarshalInput(typ reflect.Type, input interface{}) (interface{}, error) {
	if reflect.TypeOf(input) == typ {
		return input, nil
	}

	switch typ {
	case int32Type:
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

	case float64Type:
		switch input := input.(type) {
		case int32:
			return float64(input), nil
		case int:
			return float64(input), nil
		}
	}

	return nil, fmt.Errorf("incompatible type")
}
