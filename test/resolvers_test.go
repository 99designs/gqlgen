//go:generate gorunpkg github.com/vektah/gqlgen -out generated.go -typemap types.json -models models/generated.go

package test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlgen/client"
	"github.com/vektah/gqlgen/graphql"
	"github.com/vektah/gqlgen/handler"
)

func TestCustomErrorPresenter(t *testing.T) {
	resolvers := &testResolvers{}
	srv := httptest.NewServer(handler.GraphQL(MakeExecutableSchema(resolvers),
		handler.ErrorPresenter(func(i context.Context, e error) error {
			if _, ok := errors.Cause(e).(*specialErr); ok {
				return &graphql.ResolverError{Message: "override special error message"}
			}
			return &graphql.ResolverError{Message: e.Error()}
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
}

func TestErrorPath(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(MakeExecutableSchema(&testResolvers{fmt.Errorf("boom")})))
	c := client.New(srv.URL)

	var resp struct{}
	err := c.Post(`{ path { cc:child { error } } }`, &resp)

	assert.EqualError(t, err, `[{"message":"boom","path":["path",0,"cc","error"]},{"message":"boom","path":["path",1,"cc","error"]},{"message":"boom","path":["path",2,"cc","error"]},{"message":"boom","path":["path",3,"cc","error"]}]`)
}

type testResolvers struct {
	err error
}

func (r *testResolvers) Query_path(ctx context.Context) ([]Element, error) {
	return []Element{{1}, {2}, {3}, {4}}, nil
}

func (r *testResolvers) Element_child(ctx context.Context, obj *Element) (Element, error) {
	return Element{obj.ID * 10}, nil
}

func (r *testResolvers) Element_error(ctx context.Context, obj *Element) (bool, error) {
	// A silly hack to make the result order stable
	time.Sleep(time.Duration(obj.ID) * 10 * time.Millisecond)

	return false, r.err
}

type specialErr struct{}

func (*specialErr) Error() string {
	return "original special error message"
}
