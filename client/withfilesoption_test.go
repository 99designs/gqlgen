package client_test

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
)

func TestWithFiles(t *testing.T) {
	tempFile1, err := os.CreateTemp(t.TempDir(), "tempFile1")
	require.NoError(t, err)
	tempFile2, err := os.CreateTemp(t.TempDir(), "tempFile2")
	require.NoError(t, err)
	tempFile3, err := os.CreateTemp(t.TempDir(), "tempFile3")
	require.NoError(t, err)
	defer tempFile1.Close()
	defer tempFile2.Close()
	defer tempFile3.Close()
	tempFile1.WriteString(`The quick brown fox jumps over the lazy dog`)
	tempFile2.WriteString(`hello world`)
	tempFile3.WriteString(`La-Li-Lu-Le-Lo`)

	t.Run("with one file", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if !assert.NoError(t, err) {
				return
			}
			assert.True(t, strings.HasPrefix(mediaType, "multipart/"))

			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if !assert.NoError(t, err) {
					return
				}

				slurp, err := io.ReadAll(p)
				if !assert.NoError(t, err) {
					return
				}

				contentDisposition := p.Header.Get("Content-Disposition")

				if contentDisposition == `form-data; name="operations"` {
					assert.JSONEq(t, `{"query":"{ id }","variables":{"file":{}}}`, string(slurp))
				}
				if contentDisposition == `form-data; name="map"` {
					assert.JSONEq(t, `{"0":["variables.file"]}`, string(slurp))
				}
				if regexp.MustCompile(`form-data; name="0"; filename=.*`).MatchString(contentDisposition) {
					assert.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					assert.EqualValues(t, `The quick brown fox jumps over the lazy dog`, slurp)
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
			if !assert.NoError(t, err) {
				return
			}
			assert.True(t, strings.HasPrefix(mediaType, "multipart/"))

			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if !assert.NoError(t, err) {
					return
				}

				slurp, err := io.ReadAll(p)
				if !assert.NoError(t, err) {
					return
				}

				contentDisposition := p.Header.Get("Content-Disposition")

				if contentDisposition == `form-data; name="operations"` {
					assert.JSONEq(t, `{"query":"{ id }","variables":{"input":{"files":[{},{}]}}}`, string(slurp))
				}
				if contentDisposition == `form-data; name="map"` {
					// returns `{"0":["variables.input.files.0"],"1":["variables.input.files.1"]}`
					// but the order of file inputs is unpredictable between different OS systems
					assert.Contains(t, string(slurp), `{"0":`)
					assert.Contains(t, string(slurp), `["variables.input.files.0"]`)
					assert.Contains(t, string(slurp), `,"1":`)
					assert.Contains(t, string(slurp), `["variables.input.files.1"]`)
					assert.Contains(t, string(slurp), `}`)
				}
				if regexp.MustCompile(`form-data; name="[0,1]"; filename=.*`).MatchString(contentDisposition) {
					assert.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					assert.Contains(t, []string{
						`The quick brown fox jumps over the lazy dog`,
						`hello world`,
					}, string(slurp))
				}
			}
			w.Write([]byte(`{}`))
		})

		c := client.New(h)

		var resp struct{}
		c.MustPost("{ id }", &resp,
			client.Var("input", map[string]any{
				"files": []*os.File{tempFile1, tempFile2},
			}),
			client.WithFiles(),
		)
	})

	t.Run("with multiple files across multiple variables", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if !assert.NoError(t, err) {
				return
			}
			assert.True(t, strings.HasPrefix(mediaType, "multipart/"))

			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if !assert.NoError(t, err) {
					return
				}

				slurp, err := io.ReadAll(p)
				if !assert.NoError(t, err) {
					return
				}

				contentDisposition := p.Header.Get("Content-Disposition")

				if contentDisposition == `form-data; name="operations"` {
					assert.JSONEq(t, `{"query":"{ id }","variables":{"req":{"files":[{},{}],"foo":{"bar":{}}}}}`, string(slurp))
				}
				if contentDisposition == `form-data; name="map"` {
					// returns `{"0":["variables.req.files.0"],"1":["variables.req.files.1"],"2":["variables.req.foo.bar"]}`
					// but the order of file inputs is unpredictable between different OS systems
					assert.Contains(t, string(slurp), `{"0":`)
					assert.Contains(t, string(slurp), `["variables.req.files.0"]`)
					assert.Contains(t, string(slurp), `,"1":`)
					assert.Contains(t, string(slurp), `["variables.req.files.1"]`)
					assert.Contains(t, string(slurp), `,"2":`)
					assert.Contains(t, string(slurp), `["variables.req.foo.bar"]`)
					assert.Contains(t, string(slurp), `}`)
				}
				if regexp.MustCompile(`form-data; name="[0,1,2]"; filename=.*`).MatchString(contentDisposition) {
					assert.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					assert.Contains(t, []string{
						`The quick brown fox jumps over the lazy dog`,
						`La-Li-Lu-Le-Lo`,
						`hello world`,
					}, string(slurp))
				}
			}
			w.Write([]byte(`{}`))
		})

		c := client.New(h)

		var resp struct{}
		c.MustPost("{ id }", &resp,
			client.Var("req", map[string]any{
				"files": []*os.File{tempFile1, tempFile2},
				"foo": map[string]any{
					"bar": tempFile3,
				},
			}),
			client.WithFiles(),
		)
	})

	t.Run("with multiple files and file reuse", func(t *testing.T) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if !assert.NoError(t, err) {
				return
			}
			assert.True(t, strings.HasPrefix(mediaType, "multipart/"))

			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if !assert.NoError(t, err) {
					return
				}

				slurp, err := io.ReadAll(p)
				if !assert.NoError(t, err) {
					return
				}

				contentDisposition := p.Header.Get("Content-Disposition")

				if contentDisposition == `form-data; name="operations"` {
					assert.JSONEq(t, `{"query":"{ id }","variables":{"files":[{},{},{}]}}`, string(slurp))
				}
				if contentDisposition == `form-data; name="map"` {
					assert.JSONEq(t, `{"0":["variables.files.0","variables.files.2"],"1":["variables.files.1"]}`, string(slurp))
					// returns `{"0":["variables.files.0","variables.files.2"],"1":["variables.files.1"]}`
					// but the order of file inputs is unpredictable between different OS systems
					assert.Contains(t, string(slurp), `{"0":`)
					assert.Contains(t, string(slurp), `["variables.files.0"`)
					assert.Contains(t, string(slurp), `,"1":`)
					assert.Contains(t, string(slurp), `"variables.files.1"]`)
					assert.Contains(t, string(slurp), `"variables.files.2"]`)
					assert.NotContains(t, string(slurp), `,"2":`)
					assert.Contains(t, string(slurp), `}`)
				}
				if regexp.MustCompile(`form-data; name="[0,1]"; filename=.*`).MatchString(contentDisposition) {
					assert.Equal(t, `text/plain; charset=utf-8`, p.Header.Get("Content-Type"))
					assert.Contains(t, []string{
						`The quick brown fox jumps over the lazy dog`,
						`hello world`,
					}, string(slurp))
				}
				assert.False(t, regexp.MustCompile(`form-data; name="2"; filename=.*`).MatchString(contentDisposition))
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
