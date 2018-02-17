package handler

import (
	"context"
	"time"

	"github.com/vektah/gqlgen/graphql"
	"github.com/vektah/gqlgen/neelance/errors"
	"github.com/vektah/gqlgen/neelance/query"
	"github.com/vektah/gqlgen/neelance/schema"
)

type executableSchemaStub struct {
}

var _ graphql.ExecutableSchema = &executableSchemaStub{}

func (e *executableSchemaStub) Schema() *schema.Schema {
	return schema.MustParse(`
		schema { query: Query }
		type Query { me: User! }
		type User { name: String! }
	`)
}

func (e *executableSchemaStub) Query(ctx context.Context, document *query.Document, variables map[string]interface{}, op *query.Operation) *graphql.Response {
	data := graphql.OrderedMap{}
	data.Add("name", graphql.MarshalString("test"))

	return &graphql.Response{Data: &data}
}

func (e *executableSchemaStub) Mutation(ctx context.Context, document *query.Document, variables map[string]interface{}, op *query.Operation) *graphql.Response {
	return &graphql.Response{
		Errors: []*errors.QueryError{{Message: "mutations are not supported"}},
	}
}

func (e *executableSchemaStub) Subscription(ctx context.Context, document *query.Document, variables map[string]interface{}, op *query.Operation) <-chan *graphql.Response {
	events := make(chan *graphql.Response, 0)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(events)
				return
			default:
				data := graphql.OrderedMap{}
				data.Add("name", graphql.MarshalString("test"))

				events <- &graphql.Response{
					Data: &data,
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()
	return events
}
