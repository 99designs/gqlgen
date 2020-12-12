package client_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/stretchr/testify/require"
)

func TestWithFiles(t *testing.T) {
	tempFile1, _ := ioutil.TempFile(os.TempDir(), "tempFile")
	tempFile2, _ := ioutil.TempFile(os.TempDir(), "tempFile")
	tempFile3, _ := ioutil.TempFile(os.TempDir(), "tempFile")
	defer os.Remove(tempFile1.Name())
	defer os.Remove(tempFile2.Name())
	defer os.Remove(tempFile3.Name())
	tempFile1.WriteString(`The quick brown fox jumps over the lazy dog`)
	tempFile2.WriteString(`hello world`)
	tempFile3.WriteString(`La-Li-Lu-Le-Lo`)

	t.Run("with one file", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := ioutil.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="operations"`)
			require.Contains(t, string(bodyBytes), `{"query":"{ id }","variables":{"file":{}}}`)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="map"`)
			require.Contains(t, string(bodyBytes), `{"0":["variables.file"]}`)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="0"; filename=`)
			require.Contains(t, string(bodyBytes), `Content-Type: application/octet-stream`)
			require.Contains(t, string(bodyBytes), `The quick brown fox jumps over the lazy dog`)

			w.Write([]byte(`{}`))
		})

		c := client.New(h)

		var resp struct{}
		c.MustPost("{ id }", &resp,
			client.Var("file", tempFile1),
			client.WithFiles(),
		)
	})

	t.Run("with multiple files", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := ioutil.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="operations"`)
			require.Contains(t, string(bodyBytes), `{"query":"{ id }","variables":{"input":{"files":[{},{}]}}}`)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="map"`)
			require.Contains(t, string(bodyBytes), `{"0":["variables.input.files.0"],"1":["variables.input.files.1"]}`)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="0"; filename=`)
			require.Contains(t, string(bodyBytes), `Content-Type: application/octet-stream`)
			require.Contains(t, string(bodyBytes), `The quick brown fox jumps over the lazy dog`)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="1"; filename=`)
			require.Contains(t, string(bodyBytes), `hello world`)

			w.Write([]byte(`{}`))
		})

		c := client.New(h)

		var resp struct{}
		c.MustPost("{ id }", &resp,
			client.Var("input", map[string]interface{}{
				"files": []*os.File{tempFile1, tempFile2},
			}),
			client.WithFiles(),
		)
	})

	t.Run("with multiple files across multiple variables", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := ioutil.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="operations"`)
			require.Contains(t, string(bodyBytes), `{"query":"{ id }","variables":{"req":{"files":[{},{}],"foo":{"bar":{}}}}}`)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="map"`)
			require.Contains(t, string(bodyBytes), `{"0":["variables.req.files.0"],"1":["variables.req.files.1"],"2":["variables.req.foo.bar"]}`)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="0"; filename=`)
			require.Contains(t, string(bodyBytes), `Content-Type: application/octet-stream`)
			require.Contains(t, string(bodyBytes), `The quick brown fox jumps over the lazy dog`)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="1"; filename=`)
			require.Contains(t, string(bodyBytes), `hello world`)
			require.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="2"; filename=`)
			require.Contains(t, string(bodyBytes), `La-Li-Lu-Le-Lo`)

			w.Write([]byte(`{}`))
		})

		c := client.New(h)

		var resp struct{}
		c.MustPost("{ id }", &resp,
			client.Var("req", map[string]interface{}{
				"files": []*os.File{tempFile1, tempFile2},
				"foo": map[string]interface{}{
					"bar": tempFile3,
				},
			}),
			client.WithFiles(),
		)
	})
}
