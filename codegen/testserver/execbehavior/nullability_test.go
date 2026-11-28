package execbehavior

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNonNullViolations(t *testing.T) {
	s := "s"
	resolvers := &Stub{}
	resolvers.QueryResolver.Nullability = func(ctx context.Context, valid bool) (*Nullability, error) {
		if valid {
			return &Nullability{
				Optional:       &s,
				Required:       &s,
				RequiredElems:  []*string{&s},
				RequiredList:   []*string{&s, nil},
				Nested:         [][]*string{{&s}},
				RequiredObject: &Dog{Name: "rex"},
			}, nil
		}
		return &Nullability{
			RequiredDogs:  []*Dog{{Name: "rex"}, nil},
			RequiredElems: []*string{&s, nil},
			Nested:        [][]*string{{&s}, {nil}},
		}, nil
	}
	srv := newServer(resolvers, DirectiveRoot{})
	// nilField is the error of a nil in the non-null field of Nullability named field.
	nilField := func(field, typ string) string {
		return `{"message":"cannot return null for non-null field Nullability.` + field +
			` (` + typ + `): the model struct field was nil and the field has no resolver",` +
			`"path":["nullability","` + field + `"]}`
	}

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name: "valid values",
			query: `{ nullability(valid: true) {
				optional required requiredElems requiredList nested requiredObject { name }
			} }`,
			want: `{"data":{"nullability":{
				"optional":"s","required":"s","requiredElems":["s"],"requiredList":["s",null],
				"nested":[["s"]],"requiredObject":{"name":"rex"}
			}}}`,
		},
		{
			name:  "nil in a nullable field",
			query: `{ nullability(valid: false) { optional } }`,
			want:  `{"data":{"nullability":{"optional":null}}}`,
		},
		{
			name:  "nil in a non-null field nulls the parent",
			query: `{ nullability(valid: false) { optional required } }`,
			want: `{"data":{"nullability":null},"errors":[
				` + nilField("required", "String!") + `]}`,
		},
		// KNOWN BUG: lists of scalars marshal their elements without a path of their own,
		// so the error of a null element names the list, as if the field itself were nil.
		// Expected: the path ends with the index of the element, because the spec asks the
		// path of an error to point to the list item that caused it.
		// Issue: https://github.com/99designs/gqlgen/issues/4459
		{
			name:  "nil element of a list of non-null elements nulls the list",
			query: `{ nullability(valid: false) { optional requiredElems } }`,
			want: `{"data":{"nullability":{"optional":null,"requiredElems":null}},"errors":[
				` + nilField("requiredElems", "String!") + `]}`,
		},
		{
			// A nil slice in a non-null list position is marshaled as an empty list, while
			// in a nullable position it is marshaled as null. TestSlices of
			// codegen/testserver/singlefile requires the same.
			name:  "nil non-null list",
			query: `{ nullability(valid: false) { requiredList } }`,
			want:  `{"data":{"nullability":{"requiredList":[]}}}`,
		},
		{
			// KNOWN BUG: as for the list of scalars above, the error of the null element
			// names the outer list.
			// Expected: the path ends with the indices of the element, because the spec
			// asks the path of an error to point to the list item that caused it.
			// Issue: https://github.com/99designs/gqlgen/issues/4459
			name:  "nil in an inner list nulls the outer list",
			query: `{ nullability(valid: false) { optional nested } }`,
			want: `{"data":{"nullability":{"optional":null,"nested":null}},"errors":[
				` + nilField("nested", "String!") + `]}`,
		},
		{
			name:  "nil element of a list of non-null objects names the element",
			query: `{ nullability(valid: false) { optional requiredDogs { name } } }`,
			want: `{"data":{"nullability":{"optional":null,"requiredDogs":null}},"errors":[
				{"message":"cannot return null for non-null element of field ` +
				`Nullability.requiredDogs (Dog!): the list element was nil",` +
				`"path":["nullability","requiredDogs",1]}]}`,
		},
		{
			name:  "nil non-null object nulls the parent",
			query: `{ nullability(valid: false) { optional requiredObject { name } } }`,
			want: `{"data":{"nullability":null},"errors":[
				` + nilField("requiredObject", "Dog!") + `]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.JSONEq(t, tt.want, post(t, srv, tt.query))
		})
	}
}
