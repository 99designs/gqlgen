package graphql

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func TestAddUploadToOperations(t *testing.T) {
	key := "0"

	t.Run("fail missing all variables", func(t *testing.T) {
		file, _ := os.Open("path/to/file")
		params := &RawParams{}

		upload := Upload{
			File:        file,
			Filename:    "a.txt",
			Size:        int64(5),
			ContentType: "text/plain",
		}
		path := "variables.req.0.file"
		err := params.AddUpload(upload, key, path)
		require.EqualError(
			t,
			err,
			"input: path is missing \"variables.\" prefix, key: 0, path: variables.req.0.file",
		)
	})

	t.Run("valid variable", func(t *testing.T) {
		file, _ := os.Open("path/to/file")
		request := &RawParams{
			Variables: map[string]any{
				"file": nil,
			},
		}

		upload := Upload{
			File:        file,
			Filename:    "a.txt",
			Size:        int64(5),
			ContentType: "text/plain",
		}

		expected := &RawParams{
			Variables: map[string]any{
				"file": upload,
			},
		}

		path := "variables.file"
		err := request.AddUpload(upload, key, path)
		require.Equal(t, (*gqlerror.Error)(nil), err)

		require.Equal(t, expected, request)
	})

	t.Run("valid nested variable", func(t *testing.T) {
		file, _ := os.Open("path/to/file")
		request := &RawParams{
			Variables: map[string]any{
				"req": []any{
					map[string]any{
						"file": nil,
					},
				},
			},
		}

		upload := Upload{
			File:        file,
			Filename:    "a.txt",
			Size:        int64(5),
			ContentType: "text/plain",
		}

		expected := &RawParams{
			Variables: map[string]any{
				"req": []any{
					map[string]any{
						"file": upload,
					},
				},
			},
		}

		path := "variables.req.0.file"
		err := request.AddUpload(upload, key, path)
		require.Equal(t, (*gqlerror.Error)(nil), err)
		require.Equal(t, expected, request)
	})
}

func TestHeadersNotMarshalled(t *testing.T) {
	// Injecting headers in the body should not override the request headers.
	// Test that the Headers field is ignored when unmarshalling the body.
	params := &RawParams{}
	body := `{
		"query":"query todos {todos{id}}",
		"operationName":"todos",
		"variables":{"id":"1"},
		"extensions":{},
		"headers":{"Authorization":["Bearer token"]}
	}`
	err := json.Unmarshal([]byte(body), &params)
	require.NoError(t, err)
	require.Empty(t, params.Headers, "Headers injected in Body will override request headers")
}
