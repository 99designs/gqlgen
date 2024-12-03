//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_default.yml -stub generated_default/stub.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_compliant.yml -stub generated_compliant/stub.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_compliant_input_int.yml -stub generated_compliant_input_int/stub.go

package compliant_int

import "testing"

func TestModels(t *testing.T) {
	t.Skip("TODO")
}
