//go:generate gorunpkg github.com/vektah/gqlgen --config config.yaml

package test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"remote_api"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/client"
	"github.com/vektah/gqlgen/graphql"
	"github.com/vektah/gqlgen/handler"
	"github.com/vektah/gqlgen/test/models-go"
)

func TestCustomErrorPresenter(t *testing.T) {
	resolvers := &testResolvers{}
	srv := httptest.NewServer(handler.GraphQL(MakeExecutableSchema(resolvers),
		handler.ErrorPresenter(func(i context.Context, e error) *graphql.Error {
			if _, ok := errors.Cause(e).(*specialErr); ok {
				return &graphql.Error{Message: "override special error message"}
			}
			return &graphql.Error{Message: e.Error()}
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
	srv := httptest.NewServer(handler.GraphQL(MakeExecutableSchema(&testResolvers{err: fmt.Errorf("boom")})))
	c := client.New(srv.URL)

	var resp struct{}
	err := c.Post(`{ path { cc:child { error } } }`, &resp)

	assert.EqualError(t, err, `[{"message":"boom","path":["path",0,"cc","error"]},{"message":"boom","path":["path",1,"cc","error"]},{"message":"boom","path":["path",2,"cc","error"]},{"message":"boom","path":["path",3,"cc","error"]}]`)
}

func TestInputDefaults(t *testing.T) {
	called := false
	srv := httptest.NewServer(handler.GraphQL(MakeExecutableSchema(&testResolvers{
		queryDate: func(ctx context.Context, filter models.DateFilter) (bool, error) {
			assert.Equal(t, "asdf", filter.Value)
			assert.Equal(t, "UTC", *filter.Timezone)
			assert.Equal(t, models.DateFilterOpEq, *filter.Op)
			called = true

			return false, nil
		},
	})))
	c := client.New(srv.URL)

	var resp struct {
		Date bool
	}

	err := c.Post(`{ date(filter:{value: "asdf"}) }`, &resp)

	require.NoError(t, err)
	require.True(t, called)
}

func TestJsonEncoding(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(MakeExecutableSchema(&testResolvers{})))
	c := client.New(srv.URL)

	var resp struct {
		JsonEncoding string
	}

	err := c.Post(`{ jsonEncoding }`, &resp)
	require.NoError(t, err)
	require.Equal(t, "\U000fe4ed", resp.JsonEncoding)
}

type testResolvers struct {
	err       error
	queryDate func(ctx context.Context, filter models.DateFilter) (bool, error)
}

func (r *testResolvers) Query_jsonEncoding(ctx context.Context) (string, error) {
	return "\U000fe4ed", nil
}

func (r *testResolvers) Query_viewer(ctx context.Context) (*models.Viewer, error) {
	return &models.Viewer{
		User: &remote_api.User{Name: "Bob"},
	}, nil
}

func (r *testResolvers) Query_date(ctx context.Context, filter models.DateFilter) (bool, error) {
	return r.queryDate(ctx, filter)
}

func (r *testResolvers) Query_path(ctx context.Context) ([]*models.Element, error) {
	return []*models.Element{{1}, {2}, {3}, {4}}, nil
}

func (r *testResolvers) Element_child(ctx context.Context, obj *models.Element) (models.Element, error) {
	return models.Element{obj.ID * 10}, nil
}

func (r *testResolvers) Element_error(ctx context.Context, obj *models.Element) (bool, error) {
	// A silly hack to make the result order stable
	time.Sleep(time.Duration(obj.ID) * 10 * time.Millisecond)

	return false, r.err
}

func (r *testResolvers) Element_mismatched(ctx context.Context, obj *models.Element) ([]bool, error) {
	return []bool{true}, nil
}

func (r *testResolvers) User_likes(ctx context.Context, obj *remote_api.User) ([]string, error) {
	return obj.Likes, nil
}

type specialErr struct{}

func (*specialErr) Error() string {
	return "original special error message"
}
