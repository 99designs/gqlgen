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
	srv.AddTransport(transport.MultipartMixed{})

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

	type deferredData response[struct {
		Values []string `json:"values"`
	}]

	type incrementalDeferredResponse struct {
		Incremental []deferredData  `json:"incremental"`
		HasNext     bool            `json:"hasNext"`
		Errors      json.RawMessage `json:"errors"`
		Extensions  map[string]any  `json:"extensions"`
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

	cases := []struct {
		name                      string
		query                     string
		expectedInitialResponse   any
		expectedDeferredResponses []deferredData
	}{
		{
			name: "defer single",
			query: `query testDefer {
	deferSingle {
		id
		name
		... @defer {
			values
		}
	}
}`,
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
			expectedDeferredResponses: []deferredData{
				{
					Data: struct {
						Values []string `json:"values"`
					}{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Path: []any{"deferSingle"},
				},
			},
		},
		{
			name: "defer single using inline fragment with type",
			query: `query testDefer {
	deferSingle {
		id
		name
		... on DeferModel @defer {
			values
		}
	}
}`,
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
			expectedDeferredResponses: []deferredData{
				{
					Data: struct {
						Values []string `json:"values"`
					}{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Path: []any{"deferSingle"},
				},
			},
		},
		{
			name: "defer single using spread fragment",
			query: `query testDefer {
	deferSingle {
		id
		name
		... DeferFragment @defer
	}
}

fragment DeferFragment on DeferModel {
	values
}
`,
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
			expectedDeferredResponses: []deferredData{
				{
					Data: struct {
						Values []string `json:"values"`
					}{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Path: []any{"deferSingle"},
				},
			},
		},
		{
			name: "defer single with label",
			query: `query testDefer {
	deferSingle {
		id
		name
		... @defer(label: "test label") {
			values
		}
	}
}`,
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
			expectedDeferredResponses: []deferredData{
				{
					Data: struct {
						Values []string `json:"values"`
					}{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "test label",
					Path:  []any{"deferSingle"},
				},
			},
		},
		{
			name: "defer single using spread fragment with label",
			query: `query testDefer {
	deferSingle {
		id
		name
		... DeferFragment @defer(label: "test label")
	}
}

fragment DeferFragment on DeferModel {
	values
}
`,
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
			expectedDeferredResponses: []deferredData{
				{
					Data: struct {
						Values []string `json:"values"`
					}{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "test label",
					Path:  []any{"deferSingle"},
				},
			},
		},
		{
			name: "defer single when if arg is true",
			query: `query testDefer {
	deferSingle {
		id
		name
		... @defer(if: true, label: "test label") {
			values
		}
	}
}`,
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
			expectedDeferredResponses: []deferredData{
				{
					Data: struct {
						Values []string `json:"values"`
					}{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "test label",
					Path:  []any{"deferSingle"},
				},
			},
		},
		{
			name: "defer single when if arg is false",
			query: `query testDefer {
	deferSingle {
		id
		name
		... @defer(if: false) {
			values
		}
	}
}`,
			expectedInitialResponse: response[struct {
				DeferSingle deferModel
			}]{
				Data: struct {
					DeferSingle deferModel
				}{
					DeferSingle: deferModel{
						Id:     "1",
						Name:   "Defer test 1",
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
				},
			},
		},
		{
			name: "defer multiple",
			query: `query testDefer {
	deferMultiple {
		id
		name
		... @defer (label: "test label") {
			values
		}
	}
}`,
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
			expectedDeferredResponses: []deferredData{
				{
					Data: struct {
						Values []string `json:"values"`
					}{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "test label",
					Path:  []any{"deferMultiple", float64(0)},
				},
				{
					Data: struct {
						Values []string `json:"values"`
					}{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "test label",
					Path:  []any{"deferMultiple", float64(1)},
				},
				{
					Data: struct {
						Values []string `json:"values"`
					}{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "test label",
					Path:  []any{"deferMultiple", float64(2)},
				},
			},
		},
		{
			name: "defer multiple when if arg is false",
			query: `query testDefer {
	deferMultiple {
		id
		name
		... @defer(label: "test label", if: false) {
			values
		}
	}
}`,
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
							Values: []string{"test defer 1", "test defer 2", "test defer 3"},
						},
						{
							Id:     "2",
							Name:   "Defer test 2",
							Values: []string{"test defer 1", "test defer 2", "test defer 3"},
						},
						{
							Id:     "3",
							Name:   "Defer test 3",
							Values: []string{"test defer 1", "test defer 2", "test defer 3"},
						},
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/over SSE", func(t *testing.T) {
			resT := reflect.TypeOf(tc.expectedInitialResponse)
			resE := reflect.New(resT).Elem()
			resp := resE.Interface()

			read := c.SSE(context.Background(), tc.query)
			require.NoError(t, read.Next(&resp))
			assert.Equal(t, tc.expectedInitialResponse, resp)

			// If there are no deferred responses, we can stop here.
			if !resE.FieldByName("HasNext").Bool() && len(tc.expectedDeferredResponses) == 0 {
				return
			}

			deferredResponses := make([]deferredData, 0)
			for {
				var valueResp deferredData
				require.NoError(t, read.Next(&valueResp))

				if !valueResp.HasNext {
					deferredResponses = append(deferredResponses, valueResp)
					break
				}

				// Remove HasNext from comparison: we don't know the order they will be
				// delivered in, and so this can't be known in the setup. But if HasNext
				// does not work right we will either error out or get too few
				// responses, so it's still checked.
				valueResp.HasNext = false
				deferredResponses = append(deferredResponses, valueResp)
			}
			require.NoError(t, read.Close())

			slices.SortFunc(deferredResponses, func(a, b deferredData) int {
				return cmp.Compare(pathStringer(a.Path), pathStringer(b.Path))
			})
			assert.Equal(t, tc.expectedDeferredResponses, deferredResponses)
		})

		t.Run(tc.name+"/over multipart HTTP", func(t *testing.T) {
			resT := reflect.TypeOf(tc.expectedInitialResponse)
			resE := reflect.New(resT).Elem()
			resp := resE.Interface()

			read := c.IncrementalHTTP(context.Background(), tc.query)
			require.NoError(t, read.Next(&resp))
			assert.Equal(t, tc.expectedInitialResponse, resp)

			// If there are no deferred responses, we can stop here.
			if !reflect.ValueOf(resp).FieldByName("HasNext").Bool() && len(tc.expectedDeferredResponses) == 0 {
				return
			}

			deferredIncrementalData := make([]deferredData, 0)
			for {
				var valueResp incrementalDeferredResponse
				require.NoError(t, read.Next(&valueResp))
				assert.Empty(t, valueResp.Errors)
				assert.Empty(t, valueResp.Extensions)

				// Extract the incremental data from the response.
				//
				// FIXME: currently the HasNext field does not describe the state of the
				// delivery as bounded by the associated path, but rather the state of
				// the operation as a whole. This makes it impossible to determine it
				// from the response, so we can not define it ahead of time.
				//
				// It is also questionable that the incremental data objects should
				// include hasNext, so for now we remove them from assertion. Once we
				// align on the spec we must update this test, as the status of the
				// path-bounded delivery should be determinative and can be asserted.
				for _, incr := range valueResp.Incremental {
					incr.HasNext = false
					deferredIncrementalData = append(deferredIncrementalData, incr)
				}

				if !valueResp.HasNext {
					break
				}
			}
			require.NoError(t, read.Close())

			slices.SortFunc(deferredIncrementalData, func(a, b deferredData) int {
				return cmp.Compare(pathStringer(a.Path), pathStringer(b.Path))
			})
			assert.Equal(t, tc.expectedDeferredResponses, deferredIncrementalData)
		})
	}
}
