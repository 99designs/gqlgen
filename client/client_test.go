package client_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"reflect"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
)

func TestClient(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if assert.NoError(t, err) {
			assert.Equal(t, `{"query":"user(id:$id){name}","variables":{"id":1}}`, string(b))

			err = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"name": "bob",
				},
			})
			assert.NoError(t, err)
		}
	})

	c := client.New(h)

	var resp struct {
		Name string
	}

	c.MustPost("user(id:$id){name}", &resp, client.Var("id", 1))

	require.Equal(t, "bob", resp.Name)
}

func TestClientMultipartFormData(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			return
		}

		assert.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="operations"`)
		assert.Contains(t, string(bodyBytes), `{"query":"mutation ($input: Input!) {}","variables":{"file":{}}`)
		assert.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="map"`)
		assert.Contains(t, string(bodyBytes), `{"0":["variables.file"]}`)
		assert.Contains(t, string(bodyBytes), `Content-Disposition: form-data; name="0"; filename="example.txt"`)
		assert.Contains(t, string(bodyBytes), `Content-Type: text/plain`)
		assert.Contains(t, string(bodyBytes), `Hello World`)

		w.Write([]byte(`{}`))
	})

	c := client.New(h)

	var resp struct{}
	c.MustPost("{ id }", &resp,
		func(bd *client.Request) {
			bodyBuf := &bytes.Buffer{}
			bodyWriter := multipart.NewWriter(bodyBuf)
			bodyWriter.WriteField("operations", `{"query":"mutation ($input: Input!) {}","variables":{"file":{}}`)
			bodyWriter.WriteField("map", `{"0":["variables.file"]}`)

			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", `form-data; name="0"; filename="example.txt"`)
			h.Set("Content-Type", "text/plain")
			ff, _ := bodyWriter.CreatePart(h)
			ff.Write([]byte("Hello World"))
			bodyWriter.Close()

			bd.HTTP.Body = io.NopCloser(bodyBuf)
			bd.HTTP.Header.Set("Content-Type", bodyWriter.FormDataContentType())
		},
	)
}

func TestAddHeader(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "ASDF", r.Header.Get("Test-Key"))

		w.Write([]byte(`{}`))
	})

	c := client.New(h)

	var resp struct{}
	c.MustPost("{ id }", &resp,
		client.AddHeader("Test-Key", "ASDF"),
	)
}

func TestAddClientHeader(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "ASDF", r.Header.Get("Test-Key"))

		w.Write([]byte(`{}`))
	})

	c := client.New(h, client.AddHeader("Test-Key", "ASDF"))

	var resp struct{}
	c.MustPost("{ id }", &resp)
}

func TestBasicAuth(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user", user)
		assert.Equal(t, "pass", pass)

		w.Write([]byte(`{}`))
	})

	c := client.New(h)

	var resp struct{}
	c.MustPost("{ id }", &resp,
		client.BasicAuth("user", "pass"),
	)
}

func TestAddCookie(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("foo")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, "value", c.Value)

		w.Write([]byte(`{}`))
	})

	c := client.New(h)

	var resp struct{}
	c.MustPost("{ id }", &resp,
		client.AddCookie(&http.Cookie{Name: "foo", Value: "value"}),
	)
}

func TestAddExtensions(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, `{"query":"user(id:1){name}","extensions":{"persistedQuery":{"sha256Hash":"ceec2897e2da519612279e63f24658c3e91194cbb2974744fa9007a7e1e9f9e7","version":1}}}`, string(b))
		err = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"Name": "Bob",
			},
		})
		assert.NoError(t, err)
	})

	c := client.New(h)

	var resp struct {
		Name string
	}
	c.MustPost("user(id:1){name}", &resp,
		client.Extensions(map[string]any{"persistedQuery": map[string]any{"version": 1, "sha256Hash": "ceec2897e2da519612279e63f24658c3e91194cbb2974744fa9007a7e1e9f9e7"}}),
	)
}

func TestSetCustomDecodeConfig(t *testing.T) {
	now := time.Now()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"data": {"created_at":"%s"}}`, now.Format(time.RFC3339))
	})

	dc := &mapstructure.DecoderConfig{
		TagName:     "json",
		ErrorUnused: true,
		ZeroFields:  true,
		DecodeHook: func(f reflect.Type, t reflect.Type, data any) (any, error) {
			if t != reflect.TypeOf(time.Time{}) {
				return data, nil
			}

			switch f.Kind() {
			case reflect.String:
				return time.Parse(time.RFC3339, data.(string))
			default:
				return data, nil
			}
		},
	}

	c := client.New(h)

	var resp struct {
		CreatedAt time.Time `json:"created_at"`
	}

	err := c.Post("user(id: 1) {created_at}", &resp)
	require.Error(t, err)

	c.SetCustomDecodeConfig(dc)

	c.MustPost("user(id: 1) {created_at}", &resp)
	require.WithinDuration(t, now, resp.CreatedAt, time.Second)
}
