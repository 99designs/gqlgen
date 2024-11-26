package singlefile

import (
	"cmp"
	"context"
	"encoding/json"
	"math/rand"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestDefer(t *testing.T) {
	resolvers := &Stub{}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.SSE{})

	c := client.New(srv)

	resolvers.QueryResolver.DeferSingle = func(ctx context.Context) (*DeferModel, error) {
		return &DeferModel{
			ID:   "1",
			Name: "Defer test 1",
		}, nil
	}

	resolvers.QueryResolver.DeferMultiple = func(ctx context.Context) ([]*DeferModel, error) {
		return []*DeferModel{
			{
				ID:   "1",
				Name: "Defer test 1",
			},
			{
				ID:   "2",
				Name: "Defer test 2",
			},
			{
				ID:   "3",
				Name: "Defer test 3",
			},
		}, nil
	}

	resolvers.DeferModelResolver.Values = func(ctx context.Context, obj *DeferModel) ([]string, error) {
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		return []string{
			"test defer 1",
			"test defer 2",
			"test defer 3",
		}, nil
	}

	deferSingleQuery := `query testDefer {
	    deferSingle {
	        id
	        name
	        ... @defer(label: "values") {
	            values
	        }
	    }
	}`
	deferMultipleQuery := `query testDefer {
	    deferMultiple {
	        id
	        name
	        ... @defer(label: "values") {
	            values
	        }
	    }
	}`

	type deferModel struct {
		Id     string
		Name   string
		Values []string
	}
	type response[T any] struct {
		Data       T
		Label      string          `json:"label"`
		Path       []any           `json:"path"`
		HasNext    bool            `json:"hasNext"`
		Errors     json.RawMessage `json:"errors"`
		Extensions map[string]any  `json:"extensions"`
	}
	type sseDeferredResponse struct {
		Data struct {
			Values []string `json:"values"`
		}
		Label      string          `json:"label"`
		Path       []any           `json:"path"`
		HasNext    bool            `json:"hasNext"`
		Errors     json.RawMessage `json:"errors"`
		Extensions map[string]any  `json:"extensions"`
	}

	pathStringer := func(path []any) string {
		var kb strings.Builder
		for i, part := range path {
			if i != 0 {
				kb.WriteRune('.')
			}

			switch pathValue := part.(type) {
			case string:
				kb.WriteString(pathValue)
			case float64:
				kb.WriteString(strconv.FormatFloat(pathValue, 'f', -1, 64))
			default:
				t.Fatalf("unexpected path type: %T", pathValue)
			}
		}
		return kb.String()
	}

	t.Run("using SSE", func(t *testing.T) {
		cases := []struct {
			name                      string
			query                     string
			expectedInitialResponse   interface{}
			expectedDeferredResponses []sseDeferredResponse
		}{
			{
				name:  "defer single",
				query: deferSingleQuery,
				expectedInitialResponse: response[struct {
					DeferSingle deferModel
				}]{
					Data: struct {
						DeferSingle deferModel
					}{
						DeferSingle: deferModel{
							Id:     "1",
							Name:   "Defer test 1",
							Values: nil,
						},
					},
					HasNext: true,
				},
				expectedDeferredResponses: []sseDeferredResponse{
					{
						Data: struct {
							Values []string `json:"values"`
						}{
							Values: []string{"test defer 1", "test defer 2", "test defer 3"},
						},
						Label: "values",
						Path:  []any{"deferSingle"},
					},
				},
			},
			{
				name:  "defer multiple",
				query: deferMultipleQuery,
				expectedInitialResponse: response[struct {
					DeferMultiple []deferModel
				}]{
					Data: struct {
						DeferMultiple []deferModel
					}{
						DeferMultiple: []deferModel{
							{
								Id:     "1",
								Name:   "Defer test 1",
								Values: nil,
							},
							{
								Id:     "2",
								Name:   "Defer test 2",
								Values: nil,
							},
							{
								Id:     "3",
								Name:   "Defer test 3",
								Values: nil,
							},
						},
					},
					HasNext: true,
				},
				expectedDeferredResponses: []sseDeferredResponse{
					{
						Data: struct {
							Values []string `json:"values"`
						}{
							Values: []string{"test defer 1", "test defer 2", "test defer 3"},
						},
						Label: "values",
						Path:  []any{"deferMultiple", float64(0)},
					},
					{
						Data: struct {
							Values []string `json:"values"`
						}{
							Values: []string{"test defer 1", "test defer 2", "test defer 3"},
						},
						Label: "values",
						Path:  []any{"deferMultiple", float64(1)},
					},
					{
						Data: struct {
							Values []string `json:"values"`
						}{
							Values: []string{"test defer 1", "test defer 2", "test defer 3"},
						},
						Label: "values",
						Path:  []any{"deferMultiple", float64(2)},
					},
				},
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				initRespT := reflect.TypeOf(tc.expectedInitialResponse)

				sse := c.SSE(context.Background(), tc.query)
				resp := reflect.New(initRespT).Elem().Interface()
				require.NoError(t, sse.Next(&resp))
				assert.Equal(t, tc.expectedInitialResponse, resp)

				deferredResponses := make([]sseDeferredResponse, 0)
				for {
					var valueResp sseDeferredResponse
					require.NoError(t, sse.Next(&valueResp))

					if !valueResp.HasNext {
						deferredResponses = append(deferredResponses, valueResp)
						break
					} else {
						// Remove HasNext from comparison: we don't know the order they will be
						// delivered in, and so this can't be known in the setup. But if HasNext
						// does not work right we will either error out or get too few
						// responses, so it's still checked.
						valueResp.HasNext = false
						deferredResponses = append(deferredResponses, valueResp)
					}
				}
				require.NoError(t, sse.Close())

				slices.SortFunc(deferredResponses, func(a, b sseDeferredResponse) int {
					return cmp.Compare(pathStringer(a.Path), pathStringer(b.Path))
				})
				assert.Equal(t, tc.expectedDeferredResponses, deferredResponses)
			})
		}
	})
}
