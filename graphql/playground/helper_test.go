package playground

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func testResourceIntegrity(t *testing.T, handler func(title, endpoint string) http.HandlerFunc) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://localhost:8080/", http.NoBody)
	handler("example.org API", "/query").ServeHTTP(recorder, request)

	res := recorder.Result()
	defer require.NoError(t, res.Body.Close())

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.True(t, strings.HasPrefix(res.Header.Get("Content-Type"), "text/html"))

	doc, err := html.Parse(res.Body)
	require.NoError(t, err)
	assert.NotNil(t, doc)

	baseUrl := findBaseRef(doc)
	assertNodesIntegrity(t, baseUrl, doc, "script", "src", "integrity")
	assertNodesIntegrity(t, baseUrl, doc, "link", "href", "integrity")
}

func assertNodesIntegrity(
	t *testing.T,
	baseUrl string,
	root *html.Node,
	tagName, urlAttrKey, integrityAttrKey string,
) {
	t.Helper()

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == tagName {
			url, _ := getAttr(n, urlAttrKey)
			integrity, found := getAttr(n, integrityAttrKey)
			if found {
				assert.NotEmpty(t, url)
				assert.NotEmpty(t, integrity)
			}

			if url != "" && integrity != "" {
				resp, err := http.Get(baseUrl + url)
				require.NoError(t, err)
				hasher := sha256.New()
				_, err = io.Copy(hasher, resp.Body)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				actual := "sha256-" + base64.StdEncoding.EncodeToString(hasher.Sum(nil))
				assert.Equal(t, integrity, actual)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
}

func getAttr(n *html.Node, key string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

func findBaseRef(root *html.Node) string {
	var base string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "base" {
			if href, ok := getAttr(n, "href"); ok {
				base = href
				return
			}
		}
		for c := n.FirstChild; c != nil && base == ""; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return base
}
