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
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/ast"
)

func TestServer(t *testing.T) {
	es := &graphql.ExecutableSchemaMock{
		QueryFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			// Field execution happens inside the generated code, we want just enough to test against right now.
			res, err := graphql.GetRequestContext(ctx).ResolverMiddleware(ctx, func(ctx context.Context) (interface{}, error) {
				return &graphql.Response{Data: []byte(`"query resp"`)}, nil
			})
			require.NoError(t, err)

			return res.(*graphql.Response)
		},
		MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			return &graphql.Response{Data: []byte(`"mutation resp"`)}
		},
		SubscriptionFunc: func(ctx context.Context, op *ast.OperationDefinition) func() *graphql.Response {
			called := 0
			return func() *graphql.Response {
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
	srv.AddTransport(&transport.GET{})

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

	t.Run("mutations are forbidden", func(t *testing.T) {
		resp := get(srv, "/foo?query=mutation{a}")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, `{"errors":[{"message":"GET requests only allow query operations"}],"data":null}`, resp.Body.String())
	})

	t.Run("subscriptions are forbidden", func(t *testing.T) {
		resp := get(srv, "/foo?query=subscription{a}")
		assert.Equal(t, http.StatusOK, resp.Code)
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

		resp := get(srv, "/foo?query={a}")
		assert.Equal(t, http.StatusOK, resp.Code)
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

		resp := get(srv, "/foo?query={a}")
		assert.Equal(t, http.StatusOK, resp.Code)
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
