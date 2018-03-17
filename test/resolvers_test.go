//go:generate gorunpkg github.com/vektah/gqlgen -out generated.go -typemap types.json

package test

import "testing"

func TestCompiles(t *testing.T) {}
