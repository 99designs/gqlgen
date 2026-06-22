//go:build tools

// Package tools pins the gqlgen code-generation command as a module dependency
// so that `go run github.com/99designs/gqlgen` (invoked by the //go:generate
// directive in generate.go) resolves with a complete go.sum.
//
// This file is excluded from normal builds by the `tools` build tag because it
// imports a main package, which cannot be imported by ordinary code. The
// //go:generate directive lives in generate.go instead, since `go generate`
// ignores build-tagged files.
package tools

import (
	_ "github.com/99designs/gqlgen"
)
