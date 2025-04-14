package code

import (
	"go/types"
	"strings"
)

// Unalias unwraps an alias type
func Unalias(t types.Type) types.Type {
	if alias, ok := t.(*types.Alias); ok {
		// If the type is an alias type, it must first check if the non-alias
		// type is in an internal package. Only the last type in the alias
		// chain is provided as the RHS.
		if isAliasInternal(t.String(), unalias(t).String()) {
			return types.NewNamed(alias.Obj(), alias.Underlying(), nil)
		}
	}
	return unalias(t)
}

func unalias(t types.Type) types.Type {
	if p, ok := t.(*types.Pointer); ok {
		// If the type come from auto-binding,
		// it will be a pointer to an alias type.
		// (e.g: `type Cursor = entgql.Cursor[int]`)
		// *ent.Cursor is the type we got from auto-binding.
		return types.NewPointer(Unalias(p.Elem()))
	}
	return types.Unalias(t)
}

// isAliasInternal checks if an alias type path is declared for a type within
// an internal package. A best-effort attempt is made to mirror the Go internal
// visibility rules by finding the root for the rhs, and checking to ensure
// that the types both share the same root.
func isAliasInternal(lhs, rhs string) bool {
	idx := strings.LastIndex(lhs, "internal")
	if idx != -1 {
		// If the alias type contains an internal package, there is no reason
		// to continue.
		return false
	}
	idx = strings.LastIndex(rhs, "internal")
	if idx < 0 {
		return false
	}
	root := rhs[:idx]
	switch {
	// The alias type path is checked against the root of the non-alias type to
	// ensure the types being aliased share the same root.
	case strings.HasPrefix(lhs, root):
		return true
	default:
		return false
	}
}
