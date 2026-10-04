package execbehavior

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInputUnmarshaler(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Coordinates = func(ctx context.Context, at Coordinates) (string, error) {
		return fmt.Sprintf("%g/%g", at.Lat, at.Lng), nil
	}
	resolvers.QueryResolver.NullableCoordinates = func(ctx context.Context, at *Coordinates) (string, error) {
		if at == nil {
			return "nil", nil
		}
		return fmt.Sprintf("%g/%g", at.Lat, at.Lng), nil
	}
	srv := newServer(resolvers, DirectiveRoot{})

	query := `query($at: Coordinates) { nullableCoordinates(at: $at) }`
	for _, tc := range []struct {
		name  string
		query string
		vars  map[string]any
		want  string
	}{
		{
			name:  "literal",
			query: `{ coordinates(at: {lat: 35.5, lng: 139}) }`,
			want:  `{"data":{"coordinates":"35.5/139"}}`,
		},
		{
			name:  "rejected by UnmarshalGQL",
			query: `{ coordinates(at: {lat: 100, lng: 0}) }`,
			want: `{"data":null,"errors":[` +
				`{"message":"latitude out of range","path":["coordinates","at"]}]}`,
		},
		{
			name:  "null",
			query: `{ nullableCoordinates(at: null) }`,
			want:  `{"data":{"nullableCoordinates":"nil"}}`,
		},
		{
			name:  "absent",
			query: `{ nullableCoordinates }`,
			want:  `{"data":{"nullableCoordinates":"nil"}}`,
		},
		{
			name:  "variable",
			query: query,
			vars:  map[string]any{"at": map[string]any{"lat": 1.5, "lng": 2}},
			want:  `{"data":{"nullableCoordinates":"1.5/2"}}`,
		},
		{
			// KNOWN BUG: an invalid variable fails only the field that uses it.
			// Expected: a request error, as in arguments_test.go.
			// Issue: https://github.com/99designs/gqlgen/issues/4460
			name:  "variable rejected by UnmarshalGQL",
			query: query,
			vars:  map[string]any{"at": map[string]any{"lat": -91, "lng": 0}},
			want: `{"data":null,"errors":[` +
				`{"message":"latitude out of range","path":["nullableCoordinates","at"]}]}`,
		},
		{
			name:  "null variable",
			query: query,
			vars:  map[string]any{"at": nil},
			want:  `{"data":{"nullableCoordinates":"nil"}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.JSONEq(t, tc.want, postVars(t, srv, tc.query, tc.vars))
		})
	}
}
