package execbehavior

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSkipRuntimeDirective(t *testing.T) {
	_, ok := reflect.TypeFor[DirectiveRoot]().FieldByName("GenerationOnly")
	require.False(t, ok, "a skip_runtime directive has no implementation")

	resolvers := &Stub{}
	resolvers.QueryResolver.GenerationOnly = func(ctx context.Context, arg string, input GenerationOnlyInput) (string, error) {
		return arg + " " + input.Value, nil
	}
	srv := newServer(resolvers, DirectiveRoot{})

	require.JSONEq(
		t,
		`{"data":{"generationOnly":"a b"}}`,
		post(t, srv, `{ generationOnly(arg: "a", input: {value: "b"}) }`),
	)
}
