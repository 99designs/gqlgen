package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/ast"
)

func TestServer(t *testing.T) {
	es := &graphql.ExecutableSchemaMock{
		QueryFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			return &graphql.Response{Data: []byte(`"query resp"`)}
		},
		MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			return &graphql.Response{Data: []byte(`"mutation resp"`)}
		},
		SubscriptionFunc: func(ctx context.Context, op *ast.OperationDefinition) func() *graphql.Response {
			called := 0
			return func() *graphql.Response {
				fmt.Println("asdf")
				called++
				if called > 2 {
					return nil
				}
				return &graphql.Response{Data: []byte(`"subscription resp"`)}
			}
		},
		SchemaFunc: func() *ast.Schema {
			return &ast.Schema{}
		},
	}
	srv := New(es)
	srv.AddTransport(&transport.HTTPGet{})
	srv.Use(func(next graphql.Handler) graphql.Handler {
		return func(ctx context.Context, writer graphql.Writer) {
			next(ctx, writer)
		}
	})

	t.Run("returns an error if no transport matches", func(t *testing.T) {
		resp := post(srv, "/foo", "application/json")
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"transport not supported"}],"data":null}`, resp.Body.String())
	})

	t.Run("calls query on executable schema", func(t *testing.T) {
		resp := get(srv, "/foo?query={a}")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":"query resp"}`, resp.Body.String())
	})

	t.Run("calls mutation on executable schema", func(t *testing.T) {
		resp := get(srv, "/foo?query=mutation{a}")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":"mutation resp"}`, resp.Body.String())
	})

	t.Run("calls subscription repeatedly on executable schema", func(t *testing.T) {
		resp := get(srv, "/foo?query=subscription{a}")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"data":"subscription resp"}{"data":"subscription resp"}`, resp.Body.String())
	})

	t.Run("invokes middleware in order", func(t *testing.T) {
		var calls []string
		srv.Use(func(next graphql.Handler) graphql.Handler {
			return func(ctx context.Context, writer graphql.Writer) {
				calls = append(calls, "first")
				next(ctx, writer)
			}
		})
		srv.Use(func(next graphql.Handler) graphql.Handler {
			return func(ctx context.Context, writer graphql.Writer) {
				calls = append(calls, "second")
				next(ctx, writer)
			}
		})

		resp := get(srv, "/foo?query={a}")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, []string{"first", "second"}, calls)
	})
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
