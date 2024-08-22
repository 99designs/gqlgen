//go:build !go1.23

package codegen

import (
	"fmt"
	"go/types"
	"reflect"
)

func (b *builder) findBindFieldTarget(in types.Type, name string) (types.Object, error) {
	switch t := in.(type) {
	case *types.Named:
		return b.findBindFieldTarget(t.Underlying(), name)
	case *types.Struct:
		var found types.Object
		for i := 0; i < t.NumFields(); i++ {
			field := t.Field(i)
			if !field.Exported() || !equalFieldName(field.Name(), name) {
				continue
			}

			if found != nil {
				return nil, fmt.Errorf("found more than one matching field to bind for %s", name)
			}

			found = field
		}

		return found, nil
	}

	return nil, nil
}

func (b *builder) findBindEmbedsTarget(in types.Type, name string) (types.Object, error) {
	switch t := in.(type) {
	case *types.Named:
		return b.findBindEmbedsTarget(t.Underlying(), name)
	case *types.Struct:
		return b.findBindStructEmbedsTarget(t, name)
	case *types.Interface:
		return b.findBindInterfaceEmbedsTarget(t, name)
	}

	return nil, nil
}

func (b *builder) findBindStructTagTarget(in types.Type, name string) (types.Object, error) {
	if b.Config.StructTag == "" {
		return nil, nil
	}

	switch t := in.(type) {
	case *types.Named:
		return b.findBindStructTagTarget(t.Underlying(), name)
	case *types.Struct:
		var found types.Object
		for i := 0; i < t.NumFields(); i++ {
			field := t.Field(i)
			if !field.Exported() || field.Embedded() {
				continue
			}
			tags := reflect.StructTag(t.Tag(i))
			if val, ok := tags.Lookup(b.Config.StructTag); ok && equalFieldName(val, name) {
				if found != nil {
					return nil, fmt.Errorf("tag %s is ambiguous; multiple fields have the same tag value of %s", b.Config.StructTag, val)
				}

				found = field
			}
		}

		return found, nil
	}

	return nil, nil
}

func (b *builder) findBindMethodTarget(in types.Type, name string) (types.Object, error) {
	switch t := in.(type) {
	case *types.Named:
		if _, ok := t.Underlying().(*types.Interface); ok {
			return b.findBindMethodTarget(t.Underlying(), name)
		}

		return b.findBindMethoderTarget(t.Method, t.NumMethods(), name)
	case *types.Interface:
		// FIX-ME: Should use ExplicitMethod here? What's the difference?
		return b.findBindMethoderTarget(t.Method, t.NumMethods(), name)
	}

	return nil, nil
}
