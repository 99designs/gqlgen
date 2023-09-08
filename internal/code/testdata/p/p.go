//go:build private
// +build private

// This file is excluded from the build unless the "private" build tag is set.
// This is used to test loading private packages.
// See internal/code/packages_test.go for more details.
package p

import (
	"github.com/99designs/gqlgen/internal/code/testdata/b"
)

var P = b.C + " P"
