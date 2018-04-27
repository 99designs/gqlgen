//go:generate gorunpkg github.com/vektah/gqlgen -out generated.go -typemap types.json -models models/generated.go

package test

import (
	"context"
	fmt "fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlgen/client"
	"github.com/vektah/gqlgen/graphql"
	"github.com/vektah/gqlgen/handler"
	"github.com/vektah/gqlgen/test/introspection"
	invalid_identifier "github.com/vektah/gqlgen/test/invalid-identifier"
	"github.com/vektah/gqlgen/test/models"
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
		resolvers.nestedOutputsErr = &specialErr{}
		var resp struct{}
		err := c.Post(`query { nestedOutputs { inner { id } } }`, &resp)

		assert.EqualError(t, err, `[{"message":"override special error message"}]`)
	})
	t.Run("normal error", func(t *testing.T) {
		resolvers.nestedOutputsErr = fmt.Errorf("a normal error")
		var resp struct{}
		err := c.Post(`query { nestedOutputs { inner { id } } }`, &resp)

		assert.EqualError(t, err, `[{"message":"a normal error"}]`)
	})
}

func TestErrorPath(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(MakeExecutableSchema(&testResolvers{})))
	c := client.New(srv.URL)

	var resp struct{}
	err := c.Post(`{ path { cc:child { error(message: "boom") } } }`, &resp)

	assert.EqualError(t, err, `[{"message":"boom","path":["path",0,"cc","error"]},{"message":"boom","path":["path",1,"cc","error"]},{"message":"boom","path":["path",2,"cc","error"]},{"message":"boom","path":["path",3,"cc","error"]}]`)
}

type testResolvers struct {
	inner             models.InnerObject
	innerErr          error
	nestedInputs      *bool
	nestedInputsErr   error
	nestedOutputs     [][]models.OuterObject
	nestedOutputsErr  error
	invalidIdentifier *invalid_identifier.InvalidIdentifier
}

func (r *testResolvers) Query_shapes(ctx context.Context) ([]Shape, error) {
	panic("implement me")
}

func (r *testResolvers) Query_recursive(ctx context.Context, input *RecursiveInputSlice) (*bool, error) {
	panic("implement me")
}

func (r *testResolvers) Query_mapInput(ctx context.Context, input *map[string]interface{}) (*bool, error) {
	panic("implement me")
}

func (r *testResolvers) Query_collision(ctx context.Context) (*introspection.It, error) {
	panic("implement me")
}

func (r *testResolvers) OuterObject_inner(ctx context.Context, obj *models.OuterObject) (models.InnerObject, error) {
	return r.inner, r.innerErr
}

func (r *testResolvers) Query_nestedInputs(ctx context.Context, input [][]models.OuterInput) (*bool, error) {
	return r.nestedInputs, r.nestedInputsErr
}

func (r *testResolvers) Query_nestedOutputs(ctx context.Context) ([][]models.OuterObject, error) {
	return r.nestedOutputs, r.nestedOutputsErr
}

func (r *testResolvers) Query_invalidIdentifier(ctx context.Context) (*invalid_identifier.InvalidIdentifier, error) {
	return r.invalidIdentifier, nil
}

func (r *testResolvers) Query_path(ctx context.Context) ([]Element, error) {
	return []Element{{1}, {2}, {3}, {4}}, nil
}

func (r *testResolvers) Element_child(ctx context.Context, obj *Element) (Element, error) {
	return Element{obj.ID * 10}, nil
}

func (r *testResolvers) Element_error(ctx context.Context, obj *Element, message *string) (bool, error) {
	// A silly hack to make the result order stable
	time.Sleep(time.Duration(obj.ID) * 10 * time.Millisecond)

	if message != nil {
		return true, errors.New(*message)
	}
	return false, nil
}

type specialErr struct{}

func (*specialErr) Error() string {
	return "original special error message"
}
