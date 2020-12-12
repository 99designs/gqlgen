package client_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/stretchr/testify/require"
)

func TestWithFiles(t *testing.T) {
	tempFile1, _ := ioutil.TempFile(os.TempDir(), "tempFile1")
	tempFile2, _ := ioutil.TempFile(os.TempDir(), "tempFile2")
	tempFile3, _ := ioutil.TempFile(os.TempDir(), "tempFile3")
	defer os.Remove(tempFile1.Name())
	defer os.Remove(tempFile2.Name())
	defer os.Remove(tempFile3.Name())
	tempFile1.WriteString(`The quick brown fox jumps over the lazy dog`)
	tempFile2.WriteString(`hello world`)
	tempFile3.WriteString(`La-Li-Lu-Le-Lo`)

	t.Run("with one file", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(mediaType, "multipart/"))

			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				require.NoError(t, err)

				slurp, err := ioutil.ReadAll(p)
				require.NoError(t, err)

				contentDisposition := p.Header.Get("Content-Disposition")
				fmt.Printf("Part %q: %q\n", contentDisposition, slurp)

				if contentDisposition == `form-data; name="operations"` {
					require.Equal(t, []byte(`{"query":"{ id }","variables":{"file":{}}}`), slurp)
				}
				if contentDisposition == `form-data; name="map"` {
					require.Equal(t, []byte(`{"0":["variables.file"]}`), slurp)
				}
				if regexp.MustCompile(`form-data; name="0"; filename=.*`).MatchString(contentDisposition) {
					require.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					require.Equal(t, []byte(`The quick brown fox jumps over the lazy dog`), slurp)
				}
			}
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
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(mediaType, "multipart/"))

			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				require.NoError(t, err)

				slurp, err := ioutil.ReadAll(p)
				require.NoError(t, err)

				contentDisposition := p.Header.Get("Content-Disposition")
				fmt.Printf("Part %q: %q\n", contentDisposition, slurp)

				if contentDisposition == `form-data; name="operations"` {
					require.Equal(t, []byte(`{"query":"{ id }","variables":{"input":{"files":[{},{}]}}}`), slurp)
				}
				if contentDisposition == `form-data; name="map"` {
					require.Equal(t, []byte(`{"0":["variables.input.files.0"],"1":["variables.input.files.1"]}`), slurp)
				}
				if regexp.MustCompile(`form-data; name="0"; filename=.*`).MatchString(contentDisposition) {
					require.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					require.Equal(t, []byte(`The quick brown fox jumps over the lazy dog`), slurp)
				}
				if regexp.MustCompile(`form-data; name="1"; filename=.*`).MatchString(contentDisposition) {
					require.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					require.Equal(t, []byte(`hello world`), slurp)
				}
			}
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
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(mediaType, "multipart/"))

			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				require.NoError(t, err)

				slurp, err := ioutil.ReadAll(p)
				require.NoError(t, err)

				contentDisposition := p.Header.Get("Content-Disposition")
				fmt.Printf("Part %q: %q\n", contentDisposition, slurp)

				if contentDisposition == `form-data; name="operations"` {
					require.Equal(t, []byte(`{"query":"{ id }","variables":{"req":{"files":[{},{}],"foo":{"bar":{}}}}}`), slurp)
				}
				if contentDisposition == `form-data; name="map"` {
					require.Equal(t, []byte(`{"0":["variables.req.files.0"],"1":["variables.req.files.1"],"2":["variables.req.foo.bar"]}`), slurp)
				}
				if regexp.MustCompile(`form-data; name="0"; filename=.*`).MatchString(contentDisposition) {
					require.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					require.Equal(t, []byte(`The quick brown fox jumps over the lazy dog`), slurp)
				}
				if regexp.MustCompile(`form-data; name="1"; filename=.*`).MatchString(contentDisposition) {
					require.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					require.Equal(t, []byte(`hello world`), slurp)
				}
				if regexp.MustCompile(`form-data; name="2"; filename=.*`).MatchString(contentDisposition) {
					require.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					require.Equal(t, []byte(`La-Li-Lu-Le-Lo`), slurp)
				}
			}
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

	t.Run("with multiple files and file reuse", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(mediaType, "multipart/"))

			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				require.NoError(t, err)

				slurp, err := ioutil.ReadAll(p)
				require.NoError(t, err)

				contentDisposition := p.Header.Get("Content-Disposition")
				fmt.Printf("Part %q: %q\n", contentDisposition, slurp)

				if contentDisposition == `form-data; name="operations"` {
					require.Equal(t, []byte(`{"query":"{ id }","variables":{"files":[{},{},{}]}}`), slurp)
				}
				if contentDisposition == `form-data; name="map"` {
					require.Equal(t, []byte(`{"0":["variables.files.0","variables.files.2"],"1":["variables.files.1"]}`), slurp)
				}
				if regexp.MustCompile(`form-data; name="0"; filename=.*`).MatchString(contentDisposition) {
					require.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					require.Equal(t, []byte(`The quick brown fox jumps over the lazy dog`), slurp)
				}
				if regexp.MustCompile(`form-data; name="1"; filename=.*`).MatchString(contentDisposition) {
					require.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					require.Equal(t, []byte(`hello world`), slurp)
				}
			}
			w.Write([]byte(`{}`))
		})

		c := client.New(h)

		var resp struct{}
		c.MustPost("{ id }", &resp,
			client.Var("files", []*os.File{tempFile1, tempFile2, tempFile1}),
			client.WithFiles(),
		)
	})
}
