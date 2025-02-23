package singlefile

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
)

func TestEnumsResolver(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.EnumInInput = func(ctx context.Context, input *InputWithEnumValue) (EnumTest, error) {
		return input.Enum, nil
	}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("input with valid enum value", func(t *testing.T) {
		var resp struct {
			EnumInInput EnumTest
		}
		c.MustPost(`query {
			enumInInput(input: {enum: OK})
		}
		`, &resp)
		require.Equal(t, EnumTestOk, resp.EnumInInput)
	})

	t.Run("input with invalid enum value", func(t *testing.T) {
		var resp struct {
			EnumInInput EnumTest
		}
		err := c.Post(`query {
			enumInInput(input: {enum: INVALID})
		}
		`, &resp)
		require.EqualError(t, err, `http 400: {"errors":[{"message":"Value \"INVALID\" does not exist in \"EnumTest!\" enum.","locations":[{"line":2,"column":30}],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`)
	})

	t.Run("input with invalid enum value via vars", func(t *testing.T) {
		var resp struct {
			EnumInInput EnumTest
		}
		err := c.Post(`query ($input: InputWithEnumValue) {
			enumInInput(input: $input)
		}
		`, &resp, client.Var("input", map[string]any{"enum": "INVALID"}))
		require.EqualError(t, err, `http 400: {"errors":[{"message":"INVALID is not a valid EnumTest","path":["variable","input","enum"],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`)
	})
}
