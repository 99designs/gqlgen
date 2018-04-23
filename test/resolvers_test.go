//go:generate gorunpkg github.com/vektah/gqlgen -out generated.go -typemap types.json -models models/generated.go

package test

import (
	"context"
	fmt "fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/graphql"
	gqlerrors "github.com/vektah/gqlgen/neelance/errors"
	"github.com/vektah/gqlgen/neelance/query"
	"github.com/vektah/gqlgen/test/introspection"
	invalid_identifier "github.com/vektah/gqlgen/test/invalid-identifier"
	"github.com/vektah/gqlgen/test/models"
)

func TestCompiles(t *testing.T) {}

func TestErrorConverter(t *testing.T) {
	resolvers := &testResolvers{}
	s := MakeExecutableSchema(resolvers)

	doc, errs := query.Parse(`query { nestedOutputs { inner { id } } } `)
	require.Nil(t, errs)

	t.Run("with", func(t *testing.T) {
		testConvErr := func(e error) string {
			if _, ok := errors.Cause(e).(*specialErr); ok {
				return "override special error message"
			}
			return e.Error()
		}
		t.Run("special error", func(t *testing.T) {
			resolvers.nestedOutputsErr = &specialErr{}

			resp := s.Query(mkctx(doc, testConvErr), doc.Operations[0])
			require.Len(t, resp.Errors, 1)
			assert.Equal(t, "override special error message", resp.Errors[0].Message)
		})
		t.Run("normal error", func(t *testing.T) {
			resolvers.nestedOutputsErr = fmt.Errorf("a normal error")

			resp := s.Query(mkctx(doc, testConvErr), doc.Operations[0])
			require.Len(t, resp.Errors, 1)
			assert.Equal(t, "a normal error", resp.Errors[0].Message)
		})
	})

	t.Run("without", func(t *testing.T) {
		t.Run("special error", func(t *testing.T) {
			resolvers.nestedOutputsErr = &specialErr{}

			resp := s.Query(mkctx(doc, nil), doc.Operations[0])
			require.Len(t, resp.Errors, 1)
			assert.Equal(t, "original special error message", resp.Errors[0].Message)
		})
		t.Run("normal error", func(t *testing.T) {
			resolvers.nestedOutputsErr = fmt.Errorf("a normal error")

			resp := s.Query(mkctx(doc, nil), doc.Operations[0])
			require.Len(t, resp.Errors, 1)
			assert.Equal(t, "a normal error", resp.Errors[0].Message)
		})
	})
}

func mkctx(doc *query.Document, errFn func(e error) string) context.Context {
	return graphql.WithRequestContext(context.Background(), &graphql.RequestContext{
		Doc: doc,
		ResolverMiddleware: func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
			return next(ctx)
		},
		RequestMiddleware: func(ctx context.Context, next func(ctx context.Context) []byte) []byte {
			return next(ctx)
		},
		Builder: gqlerrors.Builder{
			ErrorMessageFn: errFn,
		},
	})
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

type specialErr struct{}

func (*specialErr) Error() string {
	return "original special error message"
}
