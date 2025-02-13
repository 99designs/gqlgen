//go:build private

// This file is excluded from the build unless the "private" build tag is set.
// This is used to test loading private packages.
// See internal/code/packages_test.go for more details.
package p

import (
	"github.com/john-markham/gqlgen/internal/code/testdata/b"
)

var P = b.C + " P"
