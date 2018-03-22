//go:generate gorunpkg github.com/vektah/gqlgen -out generated.go -typemap types.json

package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlgen/neelance/query"
	"github.com/vektah/gqlgen/test/introspection"
)

func TestCompiles(t *testing.T) {}

func TestErrorConverter(t *testing.T) {
	t.Run("with", func(t *testing.T) {
		testConvErr := func(e error) string {
			if _, ok := errors.Cause(e).(*specialErr); ok {
				return "override special error message"
			}
			return e.Error()
		}
		t.Run("special error", func(t *testing.T) {
			s := MakeExecutableSchema(&testResolvers{
				nestedOutputsErr: &specialErr{},
			}, WithErrorConverter(testConvErr))
			ctx := context.Background()
			doc, errs := query.Parse(`query { nestedOutputs { inner { id } } } `)
			require.Nil(t, errs)
			resp := s.Query(ctx, doc, nil, doc.Operations[0], nil)
			require.Len(t, resp.Errors, 1)
			assert.Equal(t, "override special error message", resp.Errors[0].Message)
		})
		t.Run("normal error", func(t *testing.T) {
			s := MakeExecutableSchema(&testResolvers{
				nestedOutputsErr: fmt.Errorf("a normal error"),
			}, WithErrorConverter(testConvErr))
			ctx := context.Background()
			doc, errs := query.Parse(`query { nestedOutputs { inner { id } } } `)
			require.Nil(t, errs)
			resp := s.Query(ctx, doc, nil, doc.Operations[0], nil)
			require.Len(t, resp.Errors, 1)
			assert.Equal(t, "a normal error", resp.Errors[0].Message)
		})
	})

	t.Run("without", func(t *testing.T) {
		t.Run("special error", func(t *testing.T) {
			s := MakeExecutableSchema(&testResolvers{
				nestedOutputsErr: &specialErr{},
			})
			ctx := context.Background()
			doc, errs := query.Parse(`query { nestedOutputs { inner { id } } } `)
			require.Nil(t, errs)
			resp := s.Query(ctx, doc, nil, doc.Operations[0], nil)
			require.Len(t, resp.Errors, 1)
			assert.Equal(t, "original special error message", resp.Errors[0].Message)
		})
		t.Run("normal error", func(t *testing.T) {
			s := MakeExecutableSchema(&testResolvers{
				nestedOutputsErr: fmt.Errorf("a normal error"),
			})
			ctx := context.Background()
			doc, errs := query.Parse(`query { nestedOutputs { inner { id } } } `)
			require.Nil(t, errs)
			resp := s.Query(ctx, doc, nil, doc.Operations[0], nil)
			require.Len(t, resp.Errors, 1)
			assert.Equal(t, "a normal error", resp.Errors[0].Message)
		})
	})
}

type testResolvers struct {
	inner            InnerObject
	innerErr         error
	nestedInputs     *bool
	nestedInputsErr  error
	nestedOutputs    [][]OuterObject
	nestedOutputsErr error
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

func (r *testResolvers) OuterObject_inner(ctx context.Context, obj *OuterObject) (InnerObject, error) {
	return r.inner, r.innerErr
}

func (r *testResolvers) Query_nestedInputs(ctx context.Context, input [][]OuterInput) (*bool, error) {
	return r.nestedInputs, r.nestedInputsErr
}

func (r *testResolvers) Query_nestedOutputs(ctx context.Context) ([][]OuterObject, error) {
	return r.nestedOutputs, r.nestedOutputsErr
}

type specialErr struct{}

func (*specialErr) Error() string {
	return "original special error message"
}
