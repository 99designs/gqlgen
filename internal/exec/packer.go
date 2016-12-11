package exec

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/pkg/errors"
)

type packer interface {
	pack(r *request, value interface{}) (reflect.Value, error)
}

func makePacker(s *schema.Schema, schemaType common.Type, hasDefault bool, reflectType reflect.Type) (packer, error) {
	t, nonNull := unwrapNonNull(schemaType)
	if hasDefault {
		nonNull = true
	}
	switch t := t.(type) {
	case *scalar:
		want := t.reflectType
		if !nonNull {
			want = reflect.PtrTo(want)
		}
		if reflectType != want {
			return nil, fmt.Errorf("wrong type, expected %s", want)
		}
		return &valuePacker{
			scalar:    t,
			nonNull:   nonNull,
			valueType: reflectType,
		}, nil

	case *schema.Enum:
		want := reflect.TypeOf("")
		if !nonNull {
			want = reflect.PtrTo(want)
		}
		if reflectType != want {
			return nil, fmt.Errorf("wrong type, expected %s", want)
		}
		return &valuePacker{
			scalar:    stringScalar,
			nonNull:   nonNull,
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
		elem, err := makePacker(s, t.OfType, false, reflectType.Elem())
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

		p, err := makePacker(s, f.Type, f.Default != nil, sf.Type)
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
	if v, ok := value.(common.Variable); ok {
		value = r.vars[string(v)]
	}

	values := value.(map[string]interface{})
	v := reflect.New(p.structType)
	v.Elem().Set(p.defaultStruct)
	for _, f := range p.fields {
		if value, ok := values[f.name]; ok {
			packed, err := f.fieldPacker.pack(r, value)
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
	if v, ok := value.(common.Variable); ok {
		value = r.vars[string(v)]
	}

	list := value.([]interface{})
	v := reflect.MakeSlice(e.sliceType, len(list), len(list))
	for i := range list {
		packed, err := e.elem.pack(r, list[i])
		if err != nil {
			return reflect.Value{}, err
		}
		v.Index(i).Set(packed)
	}
	return v, nil
}

type valuePacker struct {
	scalar    *scalar
	nonNull   bool
	valueType reflect.Type
}

func (p *valuePacker) pack(r *request, value interface{}) (reflect.Value, error) {
	if v, ok := value.(common.Variable); ok {
		value = r.vars[string(v)]
	}

	if value == nil {
		if p.nonNull {
			return reflect.Value{}, errors.Errorf("got null for non-null")
		}
		return reflect.Zero(p.valueType), nil
	}

	coerced, err := p.scalar.coerceInput(value)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("could not convert %#v (%T) to %s: %s", value, value, p.scalar.name, err)
	}
	if !p.nonNull {
		ptr := reflect.New(p.valueType.Elem())
		ptr.Elem().Set(reflect.ValueOf(coerced))
		return ptr, nil
	}
	return reflect.ValueOf(coerced), nil
}
