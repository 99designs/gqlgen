package followschema

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
)

func TestInput(t *testing.T) {
	resolvers := &Stub{}
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers}))
	c := client.New(srv)

	t.Run("when function errors on directives", func(t *testing.T) {
		resolvers.QueryResolver.InputSlice = func(ctx context.Context, arg []string) (b bool, e error) {
			return true, nil
		}

		var resp struct {
			DirectiveArg *string
		}

		err := c.Post(`query { inputSlice(arg: ["ok", 1, 2, "ok"]) }`, &resp)

		require.EqualError(t, err, `http 422: {"errors":[{"message":"String cannot represent a non string value: 1","locations":[{"line":1,"column":32}],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}},{"message":"String cannot represent a non string value: 2","locations":[{"line":1,"column":35}],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`)
		require.Nil(t, resp.DirectiveArg)
	})

	t.Run("when input slice nullable", func(t *testing.T) {
		resolvers.QueryResolver.InputNullableSlice = func(ctx context.Context, arg []string) (b bool, e error) {
			return arg == nil, nil
		}

		var resp struct {
			InputNullableSlice bool
		}
		var err error
		err = c.Post(`query { inputNullableSlice(arg: null) }`, &resp)
		require.NoError(t, err)
		require.True(t, resp.InputNullableSlice)

		err = c.Post(`query { inputNullableSlice(arg: []) }`, &resp)
		require.NoError(t, err)
		require.False(t, resp.InputNullableSlice)
	})

	t.Run("coerce single value to slice", func(t *testing.T) {
		check := func(ctx context.Context, arg []string) (b bool, e error) {
			return len(arg) == 1 && arg[0] == "coerced", nil
		}
		resolvers.QueryResolver.InputSlice = check
		resolvers.QueryResolver.InputNullableSlice = check

		var resp struct {
			Coerced bool
		}
		var err error
		err = c.Post(`query { coerced: inputSlice(arg: "coerced") }`, &resp)
		require.NoError(t, err)
		require.True(t, resp.Coerced)

		err = c.Post(`query { coerced: inputNullableSlice(arg: "coerced") }`, &resp)
		require.NoError(t, err)
		require.True(t, resp.Coerced)
	})
}

func TestInputOmittable(t *testing.T) {
	resolvers := &Stub{}
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers}))
	c := client.New(srv)

	t.Run("id field", func(t *testing.T) {
		resolvers.QueryResolver.InputOmittable = func(ctx context.Context, arg OmittableInput) (string, error) {
			value, isSet := arg.ID.ValueOK()
			if !isSet {
				return "<unset>", nil
			}

			if value == nil {
				return "<nil>", nil
			}

			return *value, nil
		}

		var resp struct {
			InputOmittable string
		}
		var err error

		err = c.Post(`query { inputOmittable(arg: {}) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<unset>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { id: null }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<nil>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { id: "foo" }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "foo", resp.InputOmittable)
	})

	t.Run("bool field", func(t *testing.T) {
		resolvers.QueryResolver.InputOmittable = func(ctx context.Context, arg OmittableInput) (string, error) {
			value, isSet := arg.Bool.ValueOK()
			if !isSet {
				return "<unset>", nil
			}

			if value == nil {
				return "<nil>", nil
			}

			return strconv.FormatBool(*value), nil
		}

		var resp struct {
			InputOmittable string
		}
		var err error

		err = c.Post(`query { inputOmittable(arg: {}) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<unset>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { bool: null }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<nil>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { bool: false }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "false", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { bool: true }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "true", resp.InputOmittable)
	})

	t.Run("str field", func(t *testing.T) {
		resolvers.QueryResolver.InputOmittable = func(ctx context.Context, arg OmittableInput) (string, error) {
			value, isSet := arg.Str.ValueOK()
			if !isSet {
				return "<unset>", nil
			}

			if value == nil {
				return "<nil>", nil
			}

			return *value, nil
		}

		var resp struct {
			InputOmittable string
		}
		var err error

		err = c.Post(`query { inputOmittable(arg: {}) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<unset>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { str: null }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<nil>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { str: "bar" }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "bar", resp.InputOmittable)
	})

	t.Run("int field", func(t *testing.T) {
		resolvers.QueryResolver.InputOmittable = func(ctx context.Context, arg OmittableInput) (string, error) {
			value, isSet := arg.Int.ValueOK()
			if !isSet {
				return "<unset>", nil
			}

			if value == nil {
				return "<nil>", nil
			}

			return strconv.Itoa(*value), nil
		}

		var resp struct {
			InputOmittable string
		}
		var err error

		err = c.Post(`query { inputOmittable(arg: {}) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<unset>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { int: null }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<nil>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { int: 42 }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "42", resp.InputOmittable)
	})

	t.Run("time field", func(t *testing.T) {
		resolvers.QueryResolver.InputOmittable = func(ctx context.Context, arg OmittableInput) (string, error) {
			value, isSet := arg.Time.ValueOK()
			if !isSet {
				return "<unset>", nil
			}

			if value == nil {
				return "<nil>", nil
			}

			return value.UTC().Format(time.RFC3339), nil
		}

		var resp struct {
			InputOmittable string
		}
		var err error

		err = c.Post(`query { inputOmittable(arg: {}) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<unset>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { time: null }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<nil>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { time: "2011-04-05T16:01:33Z" }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "2011-04-05T16:01:33Z", resp.InputOmittable)
	})

	t.Run("enum field", func(t *testing.T) {
		resolvers.QueryResolver.InputOmittable = func(ctx context.Context, arg OmittableInput) (string, error) {
			value, isSet := arg.Enum.ValueOK()
			if !isSet {
				return "<unset>", nil
			}

			if value == nil {
				return "<nil>", nil
			}

			return value.String(), nil
		}

		var resp struct {
			InputOmittable string
		}
		var err error

		err = c.Post(`query { inputOmittable(arg: {}) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<unset>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { enum: null }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<nil>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { enum: OK }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "OK", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { enum: ERROR }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "ERROR", resp.InputOmittable)
	})

	t.Run("scalar field", func(t *testing.T) {
		resolvers.QueryResolver.InputOmittable = func(ctx context.Context, arg OmittableInput) (string, error) {
			value, isSet := arg.Scalar.ValueOK()
			if !isSet {
				return "<unset>", nil
			}

			if value == nil {
				return "<nil>", nil
			}

			return value.str, nil
		}

		var resp struct {
			InputOmittable string
		}
		var err error

		err = c.Post(`query { inputOmittable(arg: {}) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<unset>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { scalar: null }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<nil>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { scalar: "baz" }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "baz", resp.InputOmittable)
	})

	t.Run("object field", func(t *testing.T) {
		resolvers.QueryResolver.InputOmittable = func(ctx context.Context, arg OmittableInput) (string, error) {
			value, isSet := arg.Object.ValueOK()
			if !isSet {
				return "<unset>", nil
			}

			if value == nil {
				return "<nil>", nil
			}

			return strconv.Itoa(value.Inner.ID), nil
		}

		var resp struct {
			InputOmittable string
		}
		var err error

		err = c.Post(`query { inputOmittable(arg: {}) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<unset>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { object: null }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "<nil>", resp.InputOmittable)

		err = c.Post(`query { inputOmittable(arg: { object: { inner: { id: 21 } } }) }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "21", resp.InputOmittable)
	})
}
