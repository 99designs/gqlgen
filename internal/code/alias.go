package code

import (
	"go/types"
)

// Unalias unwraps an alias type
func Unalias(t types.Type) types.Type {
	if p, ok := t.(*types.Pointer); ok {
		// If the type come from auto-binding,
		// it will be a pointer to an alias type.
		// (e.g: `type Cursor = entgql.Cursor[int]`)
		// *ent.Cursor is the type we got from auto-binding.
		return types.NewPointer(Unalias(p.Elem()))
	}
	return types.Unalias(t)
}
