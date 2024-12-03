//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_default.yml -stub stub_default.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_compliant.yml -stub stub_compliant.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_compliant_input_int.yml -stub stub_compliant_input_int.go

package compliant_int

import "testing"

func TestModels(t *testing.T) {
	t.Skip("TODO")
}
