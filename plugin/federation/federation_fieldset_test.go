package federation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/plugin/federation/fieldset"
)

func TestEntityResolverFieldSet(t *testing.T) {
	cases := []struct {
		fields string // @key(fields:)
		want   string
	}{
		{fields: "id", want: "id"},
		{fields: "_id", want: "_id"},
		{fields: "id owner { id }", want: "id owner { id }"},
		{fields: "id  owner{id}", want: "id owner { id }"},
		{fields: "owner { id name } id", want: "owner { id name } id"},
		{fields: "a { b { c } d }", want: "a { b { c } d }"},
	}

	for _, tc := range cases {
		t.Run(tc.fields, func(t *testing.T) {
			var resolver EntityResolver
			for _, field := range fieldset.New(tc.fields, nil) {
				resolver.KeyFields = append(resolver.KeyFields, &KeyField{Field: field})
			}

			require.Equal(t, tc.want, resolver.FieldSet())
		})
	}
}
