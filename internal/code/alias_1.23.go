//go:build go1.23

package code

import (
	"go/types"
)

// Unalias unwraps an alias type
func Unalias(t types.Type) types.Type {
	if p, ok := t.(*types.Pointer); ok {
		return types.NewPointer(Unalias(p.Elem()))
	}
	return types.Unalias(t)
}
