package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvalidPackagenames(t *testing.T) {
	err := generate("invalid-packagename", `
		type Query {
			invalidIdentifier: InvalidIdentifier
		}
		type InvalidIdentifier {
			id: Int!
		}
	`, map[string]string{
		"InvalidIdentifier": "github.com/vektah/gqlgen/codegen/testdata/invalid-packagename.InvalidIdentifier",
	})

	require.NoError(t, err)
}

func TestImportCollisions(t *testing.T) {
	err := generate("complexinput", `
		type Query {
			collision: It
		}
		type It {
			id: ID!
		}

	`, map[string]string{
		"It": "github.com/vektah/gqlgen/codegen/testdata/introspection.It",
	})

	require.NoError(t, err)
}

func TestDeterministicDecollisioning(t *testing.T) {
	a := Imports{
		imports: []*Import{
			{Name: "types", Path: "foobar/types"},
			{Name: "types", Path: "bazfoo/types"},
		},
	}.finalize()

	b := Imports{
		imports: []*Import{
			{Name: "types", Path: "bazfoo/types"},
			{Name: "types", Path: "foobar/types"},
		},
	}.finalize()

	require.EqualValues(t, a, b)
}
