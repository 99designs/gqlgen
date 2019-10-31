package testserver

import (
	"context"
	"fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

// New provides a server for use in tests that isn't relying on generated code. It isnt a perfect reproduction of
// a generated server, but it aims to be good enough to test the handler package without relying on codegen.
func New() *TestServer {
	next := make(chan struct{})
	now := time.Unix(0, 0)

	graphql.Now = func() time.Time {
		defer func() {
			now = now.Add(100 * time.Nanosecond)
		}()
		return now
	}

	schema := gqlparser.MustLoadSchema(&ast.Source{Input: `
		schema { query: Query }
		type Query {
			name: String!
			find(id: Int!): String!
		}
		type Mutation {
			name: String!
		}
		type Subscription {
			name: String!
		}
	`})

	es := &graphql.ExecutableSchemaMock{
		QueryFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			// Field execution happens inside the generated code, lets simulate some of it.
			ctx = graphql.WithResolverContext(ctx, &graphql.ResolverContext{
				Object: "Query",
				Field: graphql.CollectedField{
					Field: &ast.Field{
						Name:       "name",
						Alias:      "name",
						Definition: schema.Types["Query"].Fields.ForName("name"),
					},
				},
			})
			res, err := graphql.GetRequestContext(ctx).ResolverMiddleware(ctx, func(ctx context.Context) (interface{}, error) {
				return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
			})
			if err != nil {
				panic(err)
			}
			return res.(*graphql.Response)
		},
		MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			return graphql.ErrorResponse(ctx, "mutations are not supported")
		},
		SubscriptionFunc: func(ctx context.Context, op *ast.OperationDefinition) func() *graphql.Response {
			return func() *graphql.Response {
				select {
				case <-ctx.Done():
					return nil
				case <-next:
					return &graphql.Response{
						Data: []byte(`{"name":"test"}`),
					}
				}
			}
		},
		SchemaFunc: func() *ast.Schema {
			return schema
		},
	}
	return &TestServer{
		Server: handler.New(es),
		next:   next,
	}
}

type TestServer struct {
	*handler.Server
	next chan struct{}
}

func (s *TestServer) SendNextSubscriptionMessage() {
	select {
	case s.next <- struct{}{}:
	case <-time.After(1 * time.Second):
		fmt.Println("WARNING: no active subscription")
	}

}
