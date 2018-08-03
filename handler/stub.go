package handler

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

type executableSchemaStub struct {
}

var _ graphql.ExecutableSchema = &executableSchemaStub{}

func (e *executableSchemaStub) Schema() *ast.Schema {
	return gqlparser.MustLoadSchema(&ast.Source{Input: `
		schema { query: Query }
		type Query { me: User! }
		type User { name: String! }
	`})
}

func (e *executableSchemaStub) Query(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
	return &graphql.Response{Data: []byte(`{"name":"test"}`)}
}

func (e *executableSchemaStub) Mutation(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
	return graphql.ErrorResponse(ctx, "mutations are not supported")
}

func (e *executableSchemaStub) Subscription(ctx context.Context, op *ast.OperationDefinition) func() *graphql.Response {
	return func() *graphql.Response {
		time.Sleep(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			return nil
		default:
			return &graphql.Response{
				Data: []byte(`{"name":"test"}`),
			}
		}
	}
}
