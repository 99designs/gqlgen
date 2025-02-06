package playground

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testResourceIntegrity(t *testing.T, handler func(title, endpoint string) http.HandlerFunc) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://localhost:8080/", http.NoBody)
	handler("example.org API", "/query").ServeHTTP(recorder, request)

	res := recorder.Result()
	defer require.NoError(t, res.Body.Close())

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.True(t, strings.HasPrefix(res.Header.Get("Content-Type"), "text/html"))

	doc, err := goquery.NewDocumentFromReader(res.Body)
	require.NoError(t, err)
	assert.NotNil(t, doc)

	var baseUrl string
	if base := doc.Find("base"); len(base.Nodes) != 0 {
		if value, exists := base.Attr("href"); exists {
			baseUrl = value
		}
	}

	assertNodesIntegrity(t, baseUrl, doc, "script", "src", "integrity")
	assertNodesIntegrity(t, baseUrl, doc, "link", "href", "integrity")
}

func assertNodesIntegrity(t *testing.T, baseUrl string, doc *goquery.Document, selector, urlAttrKey, integrityAttrKey string) {
	selection := doc.Find(selector)
	for _, node := range selection.Nodes {
		var url string
		var integrity string
		for _, attribute := range node.Attr {
			if attribute.Key == urlAttrKey {
				url = attribute.Val
			} else if attribute.Key == integrityAttrKey {
				integrity = attribute.Val
			}
		}

		if integrity != "" {
			assert.NotEmpty(t, url)
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
}
