//go:generate gorunpkg github.com/99designs/gqlgen --config config.yaml

package integration

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
	"github.com/99designs/gqlgen/integration/models-go"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/gqlerror"
)

func TestCustomErrorPresenter(t *testing.T) {
	resolvers := &testResolver{}
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolvers}),
		handler.ErrorPresenter(func(i context.Context, e error) *gqlerror.Error {
			if _, ok := errors.Cause(e).(*specialErr); ok {
				return &gqlerror.Error{Message: "override special error message"}
			}
			return &gqlerror.Error{Message: e.Error()}
		}),
	))
	c := client.New(srv.URL)

	t.Run("special error", func(t *testing.T) {
		resolvers.err = &specialErr{}
		var resp struct{}
		err := c.Post(`{ path { cc:child { error } } }`, &resp)

		assert.EqualError(t, err, `[{"message":"override special error message"},{"message":"override special error message"},{"message":"override special error message"},{"message":"override special error message"}]`)
	})
	t.Run("normal error", func(t *testing.T) {
		resolvers.err = fmt.Errorf("a normal error")
		var resp struct{}
		err := c.Post(`{ path { cc:child { error } } }`, &resp)

		assert.EqualError(t, err, `[{"message":"a normal error"},{"message":"a normal error"},{"message":"a normal error"},{"message":"a normal error"}]`)
	})
	t.Run("multiple errors", func(t *testing.T) {
		resolvers.queryDate = func(ctx context.Context, filter models.DateFilter) (bool, error) {
			graphql.AddErrorf(ctx, "Error 1")
			graphql.AddErrorf(ctx, "Error 2")
			graphql.AddError(ctx, &specialErr{})
			return false, nil
		}

		var resp struct{ Date bool }
		err := c.Post(`{ date(filter:{value: "asdf"}) }`, &resp)

		assert.EqualError(t, err, `[{"message":"Error 1"},{"message":"Error 2"},{"message":"override special error message"}]`)
	})
}

func TestErrorPath(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: &testResolver{err: fmt.Errorf("boom")}})))
	c := client.New(srv.URL)

	var resp struct{}
	err := c.Post(`{ path { cc:child { error } } }`, &resp)

	assert.EqualError(t, err, `[{"message":"boom","path":["path",0,"cc","error"]},{"message":"boom","path":["path",1,"cc","error"]},{"message":"boom","path":["path",2,"cc","error"]},{"message":"boom","path":["path",3,"cc","error"]}]`)
}

func TestInputDefaults(t *testing.T) {
	called := false
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: &testResolver{
		queryDate: func(ctx context.Context, filter models.DateFilter) (bool, error) {
			assert.Equal(t, "asdf", filter.Value)
			assert.Equal(t, "UTC", *filter.Timezone)
			assert.Equal(t, models.DateFilterOpEq, *filter.Op)
			called = true

			return false, nil
		},
	}})))
	c := client.New(srv.URL)

	var resp struct {
		Date bool
	}

	err := c.Post(`{ date(filter:{value: "asdf"}) }`, &resp)

	require.NoError(t, err)
	require.True(t, called)
}

func TestJsonEncoding(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: &testResolver{}})))
	c := client.New(srv.URL)

	var resp struct {
		JsonEncoding string
	}

	err := c.Post(`{ jsonEncoding }`, &resp)
	require.NoError(t, err)
	require.Equal(t, "\U000fe4ed", resp.JsonEncoding)
}

type testResolver struct {
	err       error
	queryDate func(ctx context.Context, filter models.DateFilter) (bool, error)
}
