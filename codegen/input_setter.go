package codegen

import (
	"go/types"
	"strings"

	"github.com/99designs/gqlgen/codegen/config"
)

// InputSetter describes the protobuf accessors used when unmarshaling an input field.
type InputSetter struct {
	Set         string
	Clear       string
	Dereference bool
}

func (b *builder) bindInputSetter(f *Field, getter *types.Func) (bool, error) {
	name, ok := strings.CutPrefix(getter.Name(), "Get")
	sig := getter.Signature()
	if !ok || sig.Params().Len() != 0 || sig.Results().Len() != 1 {
		return false, nil
	}

	setter := inputMutator(f.Object.Type, "Set"+name, 1)
	if setter == nil {
		return false, nil
	}
	param := setter.Signature().Params().At(0).Type()
	if !types.Identical(param, sig.Results().At(0).Type()) {
		return false, nil
	}

	// Decode nullable scalars into a pointer to distinguish null from a present zero value.
	accessor := &InputSetter{Set: setter.Name()}
	if !f.Type.NonNull {
		if !config.IsNilable(param) {
			accessor.Dereference = true
			param = types.NewPointer(param)
		}
		if clearer := inputMutator(f.Object.Type, "Clear"+name, 0); clearer != nil {
			accessor.Clear = clearer.Name()
		}
	}
	ref, err := b.Binder.TypeReference(f.Type, param)
	if err != nil {
		return false, err
	}
	f.InputSetter = accessor
	f.TypeReference = ref
	return true, nil
}

func inputMutator(t types.Type, name string, params int) *types.Func {
	obj, _, _ := types.LookupFieldOrMethod(t, true, nil, name)
	method, ok := obj.(*types.Func)
	if !ok {
		return nil
	}
	sig := method.Signature()
	if sig.Variadic() || sig.Params().Len() != params || sig.Results().Len() != 0 {
		return nil
	}
	return method
}
