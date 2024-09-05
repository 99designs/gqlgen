//go:build !go1.23

package code

import (
	"go/types"
)

// Unalias unwraps an alias type
// TODO: Drop this function when we drop support for go1.22
func Unalias(t types.Type) types.Type {
	return t // No-op
}
