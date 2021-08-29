package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"net/http"
	"strings"
)

// Middleware returns an instance of cachedServer.
func Middleware(h http.Handler) http.Handler {
	return &cachedServer{h: h}
}

// cachedServer works as a middleware of Server.
// The main responsible is to write cache control header's on HTTP responses
// based of value defined on CacheControlExtension context.
type cachedServer struct {
	h http.Handler
}

func (c *cachedServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodPost {
		r = r.WithContext(WithCacheControlExtension(r.Context()))
		w = responseWriter{w, r}
	}
	c.h.ServeHTTP(w, r)
}

type responseWriter struct {
	w http.ResponseWriter
	r *http.Request
}

func (c responseWriter) Header() http.Header {
	return c.w.Header()
}

func (c responseWriter) Write(bytes []byte) (int, error) {
	if c.w.Header().Get("Cache-Control") == "" {
		resp := graphql.Response{}
		err := json.Unmarshal(bytes, &resp)
		if err == nil {
			writeCacheControl(c.r.Context(), c.w, &resp)
		}
	}

	return c.w.Write(bytes)
}

func (c responseWriter) WriteHeader(statusCode int) {
	c.w.WriteHeader(statusCode)
}

func writeCacheControl(ctx context.Context, w http.ResponseWriter, response *graphql.Response) {
	if len(response.Errors) > 0 {
		return
	}

	if cachePolicy, ok := GetOverallCachePolicy(CacheControl(ctx)); ok {
		cacheControl := fmt.Sprintf("max-age: %v %s", cachePolicy.MaxAge, strings.ToLower(string(cachePolicy.Scope)))
		w.Header().Add("Cache-Control", cacheControl)
	}
}


