package codegen

import (
	"go/types"

	"github.com/99designs/gqlgen/codegen/config"
)

func (b *builder) buildTypes() (map[string]*config.TypeReference, error) {
	ret := map[string]*config.TypeReference{}

	for _, ref := range b.Binder.References {
		for {
			ret[ref.GO.String()+ref.GQL.Name()] = ref

			if p, isPtr := ref.GO.(*types.Pointer); isPtr {
				ref = &config.TypeReference{
					GO:          p.Elem(),
					GQL:         ref.GQL,
					Definition:  ref.Definition,
					Unmarshaler: ref.Unmarshaler,
					Marshaler:   ref.Marshaler,
				}
			} else if s, isSlice := ref.GO.(*types.Slice); isSlice {
				ref = &config.TypeReference{
					GO:          s.Elem(),
					GQL:         ref.GQL,
					Definition:  ref.Definition,
					Unmarshaler: ref.Unmarshaler,
					Marshaler:   ref.Marshaler,
				}
			} else {
				break
			}
		}
	}
	return ret, nil
}
