package playground

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_createsAbsoluteURLs(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://example.org/query", http.NoBody)
	h := Handler("example.org API", "https://example.org/query")
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode = %d; want %d", res.StatusCode, http.StatusOK)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("reading res.Body: %v", err)
	}

	want := regexp.MustCompile(`(?m)^.*url\s*=\s*['"]https:\/\/example\.org\/query["'].*$`)
	if !want.Match(b) {
		t.Errorf("no match for %s in response body", want.String())
	}

	wantSubURL := regexp.MustCompile(`(?m)^.*subscriptionUrl\s*=\s*['"]wss:\/\/example\.org\/query["'].*$`)
	if !wantSubURL.Match(b) {
		t.Errorf("no match for %s in response body", wantSubURL.String())
	}

	wantMetaCharsetElement := regexp.MustCompile(`<head>\n\s{0,}<meta charset="utf-8">\n\s{0,}.*<title>`) // <meta> element must be in <head> and before <title>
	if !wantMetaCharsetElement.Match(b) {
		t.Errorf("no match for %s in response body", wantMetaCharsetElement.String())
	}
}

func TestHandler_createsRelativeURLs(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/query", http.NoBody)
	h := Handler("example.org API", "/customquery")
	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode = %d; want %d", res.StatusCode, http.StatusOK)
	}
	if res.Header.Get("Content-Type") != "text/html; charset=UTF-8" {
		t.Errorf("res.Header.Get(\"Content-Type\") = %q; want %q", res.Header.Get("Content-Type"), "text/html; charset=UTF-8")
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("reading res.Body: %v", err)
	}

	wantURL := regexp.MustCompile(`(?m)^.*url\s*=\s*location\.protocol.*$`)
	if !wantURL.Match(b) {
		t.Errorf("no match for %s in response body", wantURL.String())
	}
	wantSubURL := regexp.MustCompile(`(?m)^.*subscriptionUrl\s*=\s*wsProto.*['"]\/customquery['"].*$`)
	if !wantSubURL.Match(b) {
		t.Errorf("no match for %s in response body", wantSubURL.String())
	}
}

func TestHandler_Integrity(t *testing.T) {
	testResourceIntegrity(t, func(title, endpoint string) http.HandlerFunc {
		return Handler(title, endpoint, WithGraphiqlEnablePluginExplorer(true))
	})
}

func TestWithGraphiqlFetcherHeaders(t *testing.T) {
	t.Run("should set fetcher headers", func(t *testing.T) {
		config := &GraphiqlConfig{}
		headers := map[string]string{"Authorization": "Bearer token"}

		WithGraphiqlFetcherHeaders(headers)(config)

		assert.Equal(t, headers, config.FetcherHeaders)
	})
}

func TestWithGraphiqlUiHeaders(t *testing.T) {
	t.Run("should set ui headers", func(t *testing.T) {
		config := &GraphiqlConfig{}
		headers := map[string]string{"X-Custom-Header": "value"}

		WithGraphiqlUiHeaders(headers)(config)

		assert.Equal(t, headers, config.UiHeaders)
	})
}

func TestWithGraphiqlVersion(t *testing.T) {
	t.Run("should set graphiql version", func(t *testing.T) {
		config := &GraphiqlConfig{}
		jsURL := "https://example.com/graphiql.js"
		cssURL := "https://example.com/graphiql.css"
		jsSRI := "sha256-js"
		cssSRI := "sha256-css"

		WithGraphiqlVersion(jsURL, cssURL, jsSRI, cssSRI)(config)

		assert.Equal(t, template.URL(jsURL), config.JsUrl)
		assert.Equal(t, template.URL(cssURL), config.CssUrl)
		assert.Equal(t, cssSRI, config.CssSRI)
		assert.Equal(t, jsSRI, config.JsSRI)
	})
}

func TestWithGraphiqlReactVersion(t *testing.T) {
	t.Run("should set react version", func(t *testing.T) {
		config := &GraphiqlConfig{}
		reactJSURL := "https://example.com/react.js"
		reactDomJSURL := "https://example.com/react-dom.js"
		reactJSSRI := "sha256-react"
		reactDomJSSRI := "sha256-react-dom"

		WithGraphiqlReactVersion(reactJSURL, reactDomJSURL, reactJSSRI, reactDomJSSRI)(config)

		assert.Equal(t, template.URL(reactJSURL), config.ReactUrl)
		assert.Equal(t, template.URL(reactDomJSURL), config.ReactDOMUrl)
		assert.Equal(t, reactJSSRI, config.ReactSRI)
		assert.Equal(t, reactDomJSSRI, config.ReactDOMSRI)
	})
}

func TestWithGraphiqlPluginExplorerVersion(t *testing.T) {
	t.Run("should set plugin explorer version", func(t *testing.T) {
		config := &GraphiqlConfig{}
		jsURL := "https://example.com/plugin-explorer.js"
		cssURL := "https://example.com/plugin-explorer.css"
		jsSRI := "sha256-plugin-js"
		cssSRI := "sha256-plugin-css"

		WithGraphiqlPluginExplorerVersion(jsURL, cssURL, jsSRI, cssSRI)(config)

		assert.Equal(t, template.URL(jsURL), config.PluginExplorerJsUrl)
		assert.Equal(t, template.URL(cssURL), config.PluginExplorerCssUrl)
		assert.Equal(t, cssSRI, config.PluginExplorerCssSRI)
		assert.Equal(t, jsSRI, config.PluginExplorerJsSRI)
	})
}

func TestWithGraphiqlEnablePluginExplorer(t *testing.T) {
	tests := []struct {
		name     string
		enable   bool
		expected bool
	}{
		{
			name:     "should enable plugin explorer",
			enable:   true,
			expected: true,
		},
		{
			name:     "should disable plugin explorer",
			enable:   false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &GraphiqlConfig{}

			WithGraphiqlEnablePluginExplorer(tt.enable)(config)

			assert.Equal(t, tt.expected, config.EnablePluginExplorer)
		})
	}
}

func TestWithStoragePrefix(t *testing.T) {
	t.Run("should set storage prefix", func(t *testing.T) {
		config := &GraphiqlConfig{}
		prefix := "my-prefix"

		WithStoragePrefix(prefix)(config)

		assert.Equal(t, prefix, config.StoragePrefix)
	})
}

func TestHandler_WithOptions(t *testing.T) {
	tests := []struct {
		name    string
		options []GraphiqlConfigOption
		assert  func(t *testing.T, body string)
	}{
		{
			name: "WithGraphiqlFetcherHeaders",
			options: []GraphiqlConfigOption{
				WithGraphiqlFetcherHeaders(map[string]string{"Authorization": "Bearer token"}),
			},
			assert: func(t *testing.T, body string) {
				assert.True(t, strings.Contains(body, `const fetcherHeaders = {"Authorization":"Bearer token"};`))
			},
		},
		{
			name: "WithGraphiqlUiHeaders",
			options: []GraphiqlConfigOption{
				WithGraphiqlUiHeaders(map[string]string{"X-Custom-Header": "value"}),
			},
			assert: func(t *testing.T, body string) {
				assert.True(t, strings.Contains(body, `const uiHeaders = {"X-Custom-Header":"value"};`))
			},
		},
		{
			name: "WithGraphiqlVersion",
			options: []GraphiqlConfigOption{
				WithGraphiqlVersion("https://example.com/graphiql.js", "https://example.com/graphiql.css", "sha256-js", "sha256-css"),
			},
			assert: func(t *testing.T, body string) {
				assert.True(t, strings.Contains(body, `src="https://example.com/graphiql.js"`))
				assert.True(t, strings.Contains(body, `href="https://example.com/graphiql.css"`))
				assert.True(t, strings.Contains(body, `integrity="sha256-js"`))
				assert.True(t, strings.Contains(body, `integrity="sha256-css"`))
			},
		},
		{
			name: "WithGraphiqlReactVersion",
			options: []GraphiqlConfigOption{
				WithGraphiqlReactVersion("https://example.com/react.js", "https://example.com/react-dom.js", "sha256-react", "sha256-react-dom"),
			},
			assert: func(t *testing.T, body string) {
				assert.True(t, strings.Contains(body, `src="https://example.com/react.js"`))
				assert.True(t, strings.Contains(body, `src="https://example.com/react-dom.js"`))
				assert.True(t, strings.Contains(body, `integrity="sha256-react"`))
				assert.True(t, strings.Contains(body, `integrity="sha256-react-dom"`))
			},
		},
		{
			name: "WithGraphiqlPluginExplorerVersion",
			options: []GraphiqlConfigOption{
				WithGraphiqlEnablePluginExplorer(true),
				WithGraphiqlPluginExplorerVersion("https://example.com/plugin-explorer.js", "https://example.com/plugin-explorer.css", "sha256-plugin-js", "sha256-plugin-css"),
			},
			assert: func(t *testing.T, body string) {
				assert.True(t, strings.Contains(body, `src="https://example.com/plugin-explorer.js"`))
				assert.True(t, strings.Contains(body, `href="https://example.com/plugin-explorer.css"`))
				assert.True(t, strings.Contains(body, `integrity="sha256-plugin-js"`))
				assert.True(t, strings.Contains(body, `integrity="sha256-plugin-css"`))
			},
		},
		{
			name: "WithGraphiqlEnablePluginExplorer",
			options: []GraphiqlConfigOption{
				WithGraphiqlEnablePluginExplorer(true),
			},
			assert: func(t *testing.T, body string) {
				assert.True(t, strings.Contains(body, `GraphiQLPluginExplorer.explorerPlugin()`))
			},
		},
		{
			name: "WithStoragePrefix",
			options: []GraphiqlConfigOption{
				WithStoragePrefix("my-prefix"),
			},
			assert: func(t *testing.T, body string) {
				assert.True(t, strings.Contains(body, `new PrefixedStorage('my-prefix')`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

			h := Handler("test", "/query", tt.options...)
			h.ServeHTTP(recorder, request)

			result := recorder.Result()
			defer result.Body.Close()

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			tt.assert(t, string(body))
		})
	}
}
