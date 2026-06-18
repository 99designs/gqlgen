package singlefile

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestDefer(t *testing.T) {
	t.Parallel()

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

	resolvers.DeferModelResolver.OtherResolvedValue = func(ctx context.Context, obj *DeferModel) (string, error) {
		time.Sleep(time.Second)
		return "otherResolvedValue", nil
	}
	resolvers.DeferModelResolver.Values = func(ctx context.Context, obj *DeferModel) ([]string, error) {
		time.Sleep(time.Second * 4)
		return []string{
			"test defer 1",
			"test defer 2",
			"test defer 3",
		}, nil
	}

	type (
		deferModel struct {
			ID                 string
			Name               string
			Values             []string
			OtherResolvedValue string
		}

		response[T any] struct {
			Data       T
			Label      string          `json:"label"`
			Path       []any           `json:"path"`
			HasNext    bool            `json:"hasNext"`
			Errors     json.RawMessage `json:"errors"`
			Extensions map[string]any  `json:"extensions"`
		}

		deferSingleData struct {
			DeferSingle deferModel
		}
		deferMultipleData struct {
			DeferMultiple []deferModel
		}

		deferredData response[deferModel]

		incrementalDeferredResponse struct {
			Incremental []deferredData  `json:"incremental"`
			HasNext     bool            `json:"hasNext"`
			Errors      json.RawMessage `json:"errors"`
			Extensions  map[string]any  `json:"extensions"`
		}
	)

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

	type testCase struct {
		name                      string
		query                     string
		expectedInitialResponse   any
		expectedDeferredResponses []deferredData
		assertResponses           func(t *testing.T, tc *testCase, actual []deferredData)
	}

	cases := []testCase{
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
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID:     "1",
						Name:   "Defer test 1",
						Values: nil,
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
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
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID:     "1",
						Name:   "Defer test 1",
						Values: nil,
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
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
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID:     "1",
						Name:   "Defer test 1",
						Values: nil,
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
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
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID:     "1",
						Name:   "Defer test 1",
						Values: nil,
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
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
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID:     "1",
						Name:   "Defer test 1",
						Values: nil,
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
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
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID:     "1",
						Name:   "Defer test 1",
						Values: nil,
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
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
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID:     "1",
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
			expectedInitialResponse: response[deferMultipleData]{
				Data: deferMultipleData{
					DeferMultiple: []deferModel{
						{
							ID:     "1",
							Name:   "Defer test 1",
							Values: nil,
						},
						{
							ID:     "2",
							Name:   "Defer test 2",
							Values: nil,
						},
						{
							ID:     "3",
							Name:   "Defer test 3",
							Values: nil,
						},
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "test label",
					Path:  []any{"deferMultiple", float64(0)},
				},
				{
					Data: deferModel{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "test label",
					Path:  []any{"deferMultiple", float64(1)},
				},
				{
					Data: deferModel{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "test label",
					Path:  []any{"deferMultiple", float64(2)},
				},
			},
			assertResponses: func(t *testing.T, tc *testCase, actual []deferredData) {
				slices.SortFunc(actual, func(a, b deferredData) int {
					return strings.Compare(pathStringer(a.Path), pathStringer(b.Path))
				})
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
			expectedInitialResponse: response[deferMultipleData]{
				Data: deferMultipleData{
					DeferMultiple: []deferModel{
						{
							ID:     "1",
							Name:   "Defer test 1",
							Values: []string{"test defer 1", "test defer 2", "test defer 3"},
						},
						{
							ID:     "2",
							Name:   "Defer test 2",
							Values: []string{"test defer 1", "test defer 2", "test defer 3"},
						},
						{
							ID:     "3",
							Name:   "Defer test 3",
							Values: []string{"test defer 1", "test defer 2", "test defer 3"},
						},
					},
				},
			},
		},
		{
			name: "defer overlapping selection sets, separate defer fragments",
			query: `query testDefer {
	deferSingle {
		id
		... @defer(label: "otherResolvedValueAndName") {
			name
			otherResolvedValue
		}
		... @defer(label: "otherResolvedValueAndValues") {
			otherResolvedValue
			values
		}
	}
}`,
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID:   "1",
						Name: "Defer test 1",
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
						OtherResolvedValue: "otherResolvedValue",
					},
					Label:   "otherResolvedValueAndName",
					Path:    []any{"deferSingle"},
					HasNext: true,
				},
				{
					Data: deferModel{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "otherResolvedValueAndValues",
					Path:  []any{"deferSingle"},
				},
			},
		},
		{
			name: "defer overlapping selection sets, across concrete and interface type",
			query: `query testDefer {
	deferSingle {
		id
		... on DeferModel @defer(label: "otherResolvedValueAndName") {
			name
			otherResolvedValue
		}
		... on DeferModelInterface @defer(label: "otherResolvedValueAndValues") {
			otherResolvedValue
			values
		}
	}
}`,
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID:   "1",
						Name: "Defer test 1",
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
						OtherResolvedValue: "otherResolvedValue",
					},
					Label:   "otherResolvedValueAndName",
					Path:    []any{"deferSingle"},
					HasNext: true,
				},
				{
					Data: deferModel{
						Values: []string{"test defer 1", "test defer 2", "test defer 3"},
					},
					Label: "otherResolvedValueAndValues",
					Path:  []any{"deferSingle"},
				},
			},
		},
		{
			name: "field selected in both defer and non-defer context is not deferred",
			query: `query testDefer {
	deferSingle {
		id
		values
		... @defer {
			otherResolvedValue
			values
		}
	}
}`,
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID: "1",
						Values: []string{
							"test defer 1",
							"test defer 2",
							"test defer 3",
						},
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
						OtherResolvedValue: "otherResolvedValue",
					},
					Path:    []any{"deferSingle"},
					HasNext: false,
				},
			},
		},
		{
			name: "nested deferred fragments work",
			query: `query testDefer {
	deferSingle {
		id
		...ParentFragment @defer(label: "parent")
	}
}

fragment ParentFragment on DeferModel {
	...ChildFragment @defer(label: "child")
}

fragment ChildFragment on DeferModel {
	otherResolvedValue
}
`,
			expectedInitialResponse: response[deferSingleData]{
				Data: deferSingleData{
					DeferSingle: deferModel{
						ID: "1",
					},
				},
				HasNext: true,
			},
			expectedDeferredResponses: []deferredData{
				{
					Data: deferModel{
						OtherResolvedValue: "otherResolvedValue",
					},
					Label:   "child",
					Path:    []any{"deferSingle"},
					HasNext: true,
				},
				{
					Data:    deferModel{},
					Label:   "parent",
					Path:    []any{"deferSingle"},
					HasNext: false,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/over SSE", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				resT := reflect.TypeOf(tc.expectedInitialResponse)
				resE := reflect.New(resT).Elem()
				resp := resE.Interface()

				read := c.SSE(context.Background(), tc.query)
				synctest.Wait()
				require.NoError(t, read.Next(&resp))
				assert.Equal(t, tc.expectedInitialResponse, resp, "expected initial response to match")

				// If there are no deferred responses, we can stop here.
				if !resE.FieldByName("HasNext").Bool() && len(tc.expectedDeferredResponses) == 0 {
					return
				}

				deferredResponses := make([]deferredData, 0)
				for {
					var valueResp deferredData
					synctest.Wait()
					require.NoError(t, read.Next(&valueResp))
					deferredResponses = append(deferredResponses, valueResp)
					if !valueResp.HasNext {
						break
					}

				}
				require.NoError(t, read.Close(), "expected to close reader")

				if tc.assertResponses != nil {
					tc.assertResponses(t, &tc, deferredResponses)
				} else {
					assert.Equal(t, tc.expectedDeferredResponses, deferredResponses, "expected deferred responses to match")
				}
			})
		})

		t.Run(tc.name+"/over multipart HTTP", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				resT := reflect.TypeOf(tc.expectedInitialResponse)
				resE := reflect.New(resT).Elem()
				resp := resE.Interface()

				read := c.IncrementalHTTP(context.Background(), tc.query)

				synctest.Wait()
				require.NoError(t, read.Next(&resp))
				assert.Equal(t, tc.expectedInitialResponse, resp)

				// If there are no deferred responses, we can stop here.
				if !reflect.ValueOf(resp).FieldByName("HasNext").Bool() &&
					len(tc.expectedDeferredResponses) == 0 {
					return
				}

				deferredIncrementalData := make([]deferredData, 0)
				for {
					var valueResp incrementalDeferredResponse
					synctest.Wait()
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
					deferredIncrementalData = append(deferredIncrementalData, valueResp.Incremental...)

					if !valueResp.HasNext {
						break
					}
				}
				require.NoError(t, read.Close())
				if tc.assertResponses != nil {
					tc.assertResponses(t, &tc, deferredIncrementalData)
				} else {
					assert.Equal(t, tc.expectedDeferredResponses, deferredIncrementalData, "expected deferred responses to match")
				}
			})
		})
	}
}
