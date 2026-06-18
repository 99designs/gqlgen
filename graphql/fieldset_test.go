package graphql

import (
	"bytes"
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestFieldSet_MarshalGQL(t *testing.T) {
	t.Run("Should_Deduplicate_Keys", func(t *testing.T) {
		fs := NewFieldSet([]CollectedField{
			{Field: &ast.Field{Alias: "__typename"}},
			{Field: &ast.Field{Alias: "__typename"}},
		})
		fs.Values[0] = MarshalString("A")
		fs.Values[1] = MarshalString("A")

		b := bytes.NewBuffer(nil)
		fs.MarshalGQL(b)

		assert.JSONEq(t, "{\"__typename\":\"A\"}", b.String())
	})
}

func addConcurrentFieldAndReturnIndex(t *testing.T, fieldSet *FieldSet, field *ast.Field, resolver func(context.Context) Marshaler) int {
	t.Helper()
	fieldSet.AddField(CollectedField{Field: field})
	i := len(fieldSet.Values) - 1
	fieldSet.Concurrently(i, resolver)
	return i
}

func TestFieldSetView(t *testing.T) {
	t.Parallel()
	t.Run("properly_yields_values_and_takes_them", func(t *testing.T) {
		synctest.Test(
			t,
			func(t *testing.T) {
				// Arrange
				fieldSet := NewFieldSet(nil)
				view1 := fieldSet.NewView()
				view2 := fieldSet.NewView()
				view3 := fieldSet.NewView()

				slowestField := addConcurrentFieldAndReturnIndex(
					t,
					fieldSet,
					&ast.Field{
						Alias: "slowestField",
					},
					func(ctx context.Context) Marshaler {
						time.Sleep(time.Second * 3)
						return MarshalString("slowestFieldValue")
					},
				)

				secondSlowestField := addConcurrentFieldAndReturnIndex(
					t,
					fieldSet,
					&ast.Field{Alias: "secondSlowestField"},
					func(ctx context.Context) Marshaler {
						time.Sleep(time.Second * 2)
						return MarshalString("secondSlowestFieldValue")
					},
				)

				fastestField := addConcurrentFieldAndReturnIndex(
					t,
					fieldSet,
					&ast.Field{Alias: "fastestField"},
					func(ctx context.Context) Marshaler {
						time.Sleep(time.Second * 1)
						return MarshalString("fastestFieldValue")
					},
				)

				view1.AddIndices(slowestField, fastestField)
				view2.AddIndices(fastestField)
				view3.AddIndices(fastestField, secondSlowestField)

				resultCh := make(chan *FieldSetView)
				view1.SetOnComplete(func(ctx context.Context) {
					resultCh <- view1
				})
				view2.SetOnComplete(func(ctx context.Context) {
					resultCh <- view2
				})
				view3.SetOnComplete(func(ctx context.Context) {
					resultCh <- view3
				})

				// Act
				go fieldSet.Dispatch(t.Context())

				// Assert
				expectedResults := []string{
					`{"fastestField":"fastestFieldValue"}`,
					`{"secondSlowestField":"secondSlowestFieldValue"}`,
					`{"slowestField":"slowestFieldValue"}`,
				}
				for _, expected := range expectedResults {
					synctest.Wait()
					view := <-resultCh
					var buf bytes.Buffer
					view.MarshalGQL(&buf)
					assert.JSONEq(t, expected, buf.String())
				}
			},
		)
	})
}
