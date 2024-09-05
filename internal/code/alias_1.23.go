//go:build go1.23

package code

import (
	"go/types"
)

// Unalias unwraps an alias type
func Unalias(t types.Type) types.Type {
	return types.Unalias(t)
}
