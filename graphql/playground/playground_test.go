package playground

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestHandler_createsAbsoluteURLs(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://example.org/query", nil)
	h := Handler("example.org API", "https://example.org/query")
	h.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode = %d; want %d", res.StatusCode, http.StatusOK)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		panic(fmt.Errorf("reading res.Body: %w", err))
	}

	want := regexp.MustCompile(`(?m)^.*url\s*=\s*['"]https:\/\/example\.org\/query["'].*$`)
	if !want.Match(b) {
		t.Errorf("no match for %s in response body", want.String())
	}
}

func TestHandler_createsRelativeURLs(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/query", nil)
	h := Handler("example.org API", "/query")
	h.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode = %d; want %d", res.StatusCode, http.StatusOK)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		panic(fmt.Errorf("reading res.Body: %w", err))
	}

	want := regexp.MustCompile(`(?m)^.*url\s*=\s*location.protocol.*$`)
	if !want.Match(b) {
		t.Errorf("no match for %s in response body", want.String())
	}
}
