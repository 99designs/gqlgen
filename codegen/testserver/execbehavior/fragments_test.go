package execbehavior

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFragmentsAndConditions has no schema of its own: it uses the interface and union of
// abstract_types.graphql.
func TestFragmentsAndConditions(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Animals = func(ctx context.Context) ([]Animal, error) {
		return []Animal{&Dog{Name: "rex", Barks: true}, &Cat{Name: "tom", Lives: 9}}, nil
	}
	resolvers.QueryResolver.Pets = func(ctx context.Context) ([]Pet, error) {
		return []Pet{&Dog{Name: "rex", Barks: true}, &Cat{Name: "tom", Lives: 9}}, nil
	}
	srv := newServer(resolvers, DirectiveRoot{})

	tests := []struct {
		name      string
		query     string
		variables map[string]any
		want      string
	}{
		{
			name: "named fragments on an interface and its implementors",
			query: `{ animals { ...AnimalFields } }
				fragment AnimalFields on Animal { __typename name ...DogFields ...CatFields }
				fragment DogFields on Dog { barks }
				fragment CatFields on Cat { lives }`,
			want: `{"data":{"animals":[
				{"__typename":"Dog","name":"rex","barks":true},
				{"__typename":"Cat","name":"tom","lives":9}
			]}}`,
		},
		{
			name:  "interface fragment on union members",
			query: `{ pets { ... on Animal { name } ... on Cat { lives } } }`,
			want:  `{"data":{"pets":[{"name":"rex"},{"name":"tom","lives":9}]}}`,
		},
		{
			name:  "same field selected twice is merged",
			query: `{ animals { name ... on Dog { name barks } ... on Animal { name } } }`,
			want:  `{"data":{"animals":[{"name":"rex","barks":true},{"name":"tom"}]}}`,
		},
		{
			name:  "aliases",
			query: `{ animals { first: name second: name ... on Dog { loud: barks } } }`,
			want: `{"data":{"animals":[
				{"first":"rex","second":"rex","loud":true},
				{"first":"tom","second":"tom"}
			]}}`,
		},
		{
			name: "skip and include with literals",
			query: `{ animals {
				name @skip(if: true)
				__typename @include(if: true)
				... on Dog @include(if: false) { barks }
				... on Cat @skip(if: false) { lives }
			} }`,
			want: `{"data":{"animals":[{"__typename":"Dog"},{"__typename":"Cat","lives":9}]}}`,
		},
		{
			name: "skip and include with variables",
			query: `query($skip: Boolean!, $include: Boolean!) { animals {
				name @skip(if: $skip)
				... DogFields @include(if: $include)
			} }
			fragment DogFields on Dog { barks }`,
			variables: map[string]any{"skip": false, "include": true},
			want:      `{"data":{"animals":[{"name":"rex","barks":true},{"name":"tom"}]}}`,
		},
		{
			name:      "a field both skipped and included is skipped",
			query:     `query($yes: Boolean!) { animals { name @skip(if: $yes) @include(if: $yes) __typename } }`,
			variables: map[string]any{"yes": true},
			want:      `{"data":{"animals":[{"__typename":"Dog"},{"__typename":"Cat"}]}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.JSONEq(t, tt.want, postVars(t, srv, tt.query, tt.variables))
		})
	}
}
