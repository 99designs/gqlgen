package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/testserver"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	srv := testserver.New()
	srv.AddTransport(&transport.GET{})

	t.Run("returns an error if no transport matches", func(t *testing.T) {
		resp := post(srv, "/foo", "application/json")
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"transport not supported"}],"data":null}`, resp.Body.String())
	})

	t.Run("calls query on executable schema", func(t *testing.T) {
		resp := get(srv, "/foo?query={name}")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":{"name":"test"}}`, resp.Body.String())
	})

	t.Run("mutations are forbidden", func(t *testing.T) {
		resp := get(srv, "/foo?query=mutation{name}")
		assert.Equal(t, http.StatusNotAcceptable, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"GET requests only allow query operations"}],"data":null}`, resp.Body.String())
	})

	t.Run("subscriptions are forbidden", func(t *testing.T) {
		resp := get(srv, "/foo?query=subscription{name}")
		assert.Equal(t, http.StatusNotAcceptable, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"GET requests only allow query operations"}],"data":null}`, resp.Body.String())
	})

	t.Run("invokes operation middleware in order", func(t *testing.T) {
		var calls []string
		srv.Use(opFunc(func(ctx context.Context, next graphql.OperationHandler, writer graphql.Writer) {
			calls = append(calls, "first")
			next(ctx, writer)
		}))
		srv.Use(opFunc(func(ctx context.Context, next graphql.OperationHandler, writer graphql.Writer) {
			calls = append(calls, "second")
			next(ctx, writer)
		}))

		resp := get(srv, "/foo?query={name}")
		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.Equal(t, []string{"first", "second"}, calls)
	})

	t.Run("invokes field middleware in order", func(t *testing.T) {
		var calls []string
		srv.Use(fieldFunc(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
			fmt.Println("first")
			calls = append(calls, "first")
			return next(ctx)
		}))
		srv.Use(fieldFunc(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
			fmt.Println("second")
			calls = append(calls, "second")
			return next(ctx)
		}))

		resp := get(srv, "/foo?query={name}")
		assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		assert.Equal(t, []string{"first", "second"}, calls)
	})
}

type opFunc func(ctx context.Context, next graphql.OperationHandler, writer graphql.Writer)

func (r opFunc) InterceptOperation(ctx context.Context, next graphql.OperationHandler, writer graphql.Writer) {
	r(ctx, next, writer)
}

type fieldFunc func(ctx context.Context, next graphql.Resolver) (res interface{}, err error)

func (f fieldFunc) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	return f(ctx, next)
}

func get(handler http.Handler, target string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", target, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}

func post(handler http.Handler, target, contentType string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", target, nil)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}
