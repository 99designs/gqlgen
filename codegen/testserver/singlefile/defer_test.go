package singlefile

import (
	"context"
	"encoding/json"
	"math/rand"
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

	resolvers.QueryResolver.DeferCase1 = func(ctx context.Context) (*DeferModel, error) {
		return &DeferModel{
			ID:   "1",
			Name: "Defer test 1",
		}, nil
	}

	resolvers.QueryResolver.DeferCase2 = func(ctx context.Context) ([]*DeferModel, error) {
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

	t.Run("test deferCase1 using SSE", func(t *testing.T) {
		sse := c.SSE(context.Background(), `query testDefer {
    deferCase1 {
        id
        name
        ... on DeferModel @defer(label: "values") {
            values
        }
    }
}`)

		type response struct {
			Data struct {
				DeferCase1 struct {
					Id     string
					Name   string
					Values []string
				}
			}
			Label      string                 `json:"label"`
			Path       []interface{}          `json:"path"`
			HasNext    bool                   `json:"hasNext"`
			Errors     json.RawMessage        `json:"errors"`
			Extensions map[string]interface{} `json:"extensions"`
		}
		var resp response

		require.NoError(t, sse.Next(&resp))
		expectedInitialResponse := response{
			Data: struct {
				DeferCase1 struct {
					Id     string
					Name   string
					Values []string
				}
			}{
				DeferCase1: struct {
					Id     string
					Name   string
					Values []string
				}{
					Id:     "1",
					Name:   "Defer test 1",
					Values: nil,
				},
			},
			HasNext: true,
		}
		assert.Equal(t, expectedInitialResponse, resp)

		type valuesResponse struct {
			Data struct {
				Values []string `json:"values"`
			}
			Label      string                 `json:"label"`
			Path       []interface{}          `json:"path"`
			HasNext    bool                   `json:"hasNext"`
			Errors     json.RawMessage        `json:"errors"`
			Extensions map[string]interface{} `json:"extensions"`
		}

		var valueResp valuesResponse
		expectedResponse := valuesResponse{
			Data: struct {
				Values []string `json:"values"`
			}{
				Values: []string{"test defer 1", "test defer 2", "test defer 3"},
			},
			Label: "values",
			Path:  []interface{}{"deferCase1"},
		}

		require.NoError(t, sse.Next(&valueResp))

		assert.Equal(t, expectedResponse, valueResp)

		require.NoError(t, sse.Close())
	})

	t.Run("test deferCase2 using SSE", func(t *testing.T) {
		sse := c.SSE(context.Background(), `query testDefer {
    deferCase2 {
        id
        name
        ... on DeferModel @defer(label: "values") {
            values
        }
    }
}`)

		type response struct {
			Data struct {
				DeferCase2 []struct {
					Id     string
					Name   string
					Values []string
				}
			}
			Label      string                 `json:"label"`
			Path       []interface{}          `json:"path"`
			HasNext    bool                   `json:"hasNext"`
			Errors     json.RawMessage        `json:"errors"`
			Extensions map[string]interface{} `json:"extensions"`
		}
		var resp response

		require.NoError(t, sse.Next(&resp))
		expectedInitialResponse := response{
			Data: struct {
				DeferCase2 []struct {
					Id     string
					Name   string
					Values []string
				}
			}{
				DeferCase2: []struct {
					Id     string
					Name   string
					Values []string
				}{
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
		}
		assert.Equal(t, expectedInitialResponse, resp)

		type valuesResponse struct {
			Data struct {
				Values []string `json:"values"`
			}
			Label      string                 `json:"label"`
			Path       []interface{}          `json:"path"`
			HasNext    bool                   `json:"hasNext"`
			Errors     json.RawMessage        `json:"errors"`
			Extensions map[string]interface{} `json:"extensions"`
		}

		valuesByPath := make(map[string][]string, 2)

		for {
			var valueResp valuesResponse
			require.NoError(t, sse.Next(&valueResp))

			var kb strings.Builder
			for i, path := range valueResp.Path {
				if i != 0 {
					kb.WriteRune('.')
				}

				switch pathValue := path.(type) {
				case string:
					kb.WriteString(pathValue)
				case float64:
					kb.WriteString(strconv.FormatFloat(pathValue, 'f', -1, 64))
				default:
					t.Fatalf("unexpected path type: %T", pathValue)
				}
			}

			valuesByPath[kb.String()] = valueResp.Data.Values
			if !valueResp.HasNext {
				break
			}
		}

		assert.Equal(t, valuesByPath["deferCase2.0"], []string{"test defer 1", "test defer 2", "test defer 3"})
		assert.Equal(t, valuesByPath["deferCase2.1"], []string{"test defer 1", "test defer 2", "test defer 3"})
		assert.Equal(t, valuesByPath["deferCase2.2"], []string{"test defer 1", "test defer 2", "test defer 3"})

		for i := range resp.Data.DeferCase2 {
			resp.Data.DeferCase2[i].Values = valuesByPath["deferCase2."+strconv.FormatInt(int64(i), 10)]
		}

		expectedDeferCase2Response := response{
			Data: struct {
				DeferCase2 []struct {
					Id     string
					Name   string
					Values []string
				}
			}{
				DeferCase2: []struct {
					Id     string
					Name   string
					Values []string
				}{
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
			HasNext: true,
		}
		assert.Equal(t, expectedDeferCase2Response, resp)

		require.NoError(t, sse.Close())
	})
}
