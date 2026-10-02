package returnpointersinunmarshalinput

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestInputObjectDirectiveWithPointers(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.CheckedInput = func(ctx context.Context, input *CheckedInput) (string, error) {
		if input == nil {
			return "nil input", nil
		}
		return input.Mode + ":" + input.Value, nil
	}
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			InputCheck: func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
				res, err := next(ctx)
				if err != nil {
					return nil, err
				}
				in, ok := res.(*CheckedInput)
				if !ok {
					return nil, errors.New("inputCheck did not receive a pointer")
				}
				switch in.Mode {
				case "replace":
					return &CheckedInput{Mode: in.Mode, Value: "replaced " + in.Value}, nil
				case "typed nil":
					return (*CheckedInput)(nil), nil
				case "nil":
					return nil, nil
				case "value":
					return *in, nil
				}
				return in, nil
			},
		},
	}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	query := func(mode string) (string, error) {
		var resp struct{ CheckedInput string }
		err := c.Post(
			`query($mode: String!) { checkedInput(input: { mode: $mode, value: "v" }) }`,
			&resp,
			client.Var("mode", mode),
		)
		return resp.CheckedInput, err
	}

	got, err := query("pass")
	require.NoError(t, err)
	require.Equal(t, "pass:v", got)

	got, err = query("replace")
	require.NoError(t, err)
	require.Equal(t, "replace:replaced v", got)

	// A typed nil pointer replaces the input with nil.
	got, err = query("typed nil")
	require.NoError(t, err)
	require.Equal(t, "nil input", got)

	// An untyped nil is not a *CheckedInput.
	_, err = query("nil")
	require.Equal(
		t,
		[]string{"unexpected type <nil> from INPUT_OBJECT directive, should be *CheckedInput"},
		messages(t, err),
	)

	_, err = query("value")
	require.Equal(
		t,
		[]string{
			"unexpected type returnpointersinunmarshalinput.CheckedInput from INPUT_OBJECT directive, should be *CheckedInput",
		},
		messages(t, err),
	)
}

// messages returns the messages of the GraphQL errors that the client returned as err.
func messages(t *testing.T, err error) []string {
	t.Helper()
	require.Error(t, err)
	var errs []struct{ Message string }
	require.NoError(t, json.Unmarshal([]byte(err.Error()), &errs))
	res := make([]string, len(errs))
	for i, e := range errs {
		res[i] = e.Message
	}
	return res
}
