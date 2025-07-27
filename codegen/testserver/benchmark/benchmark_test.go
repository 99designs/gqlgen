//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package benchmark

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/codegen/testserver/benchmark/generated"
	"github.com/99designs/gqlgen/codegen/testserver/benchmark/generated/models"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
)

func BenchmarkResolvers(b *testing.B) {
	b.StopTimer()

	resolvers := &Stub{}
	resolvers.QueryResolver.Users = func(ctx context.Context, query *string, first *int, last *int, before *string, after *string, orderBy models.UserOrderBy) (*models.UserConnection, error) {
		return &models.UserConnection{
			Edges: []*models.UserEdge{
				{
					Cursor: "abc",
					Node: &models.User{
						FirstName: "John",
						LastName:  "Doe",
						Email:     "johndoe@acme.inc",
					},
				},
			},
			PageInfo: &models.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     "abc",
				EndCursor:       "abc",
			},
			TotalCount: 1,
		}, nil
	}

	exec := executor.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolvers,
	}))

	fragment := `fragment userConnection on UserConnection {
edges {
			cursor
			node {
				firstName
				lastName
				email
			}
		}
		pageInfo {
			hasNextPage
			hasPreviousPage
			startCursor
			endCursor
		}
		totalCount}`

	benchmarks := []struct {
		name   string
		params *graphql.RawParams
	}{
		{
			name: "query_no_params",
			params: &graphql.RawParams{
				Query: fragment + `
query GetUsersConnection {
	users {
		... userConnection
	}
}
`,
				OperationName: "GetUsersConnection",
				Variables:     nil,
			},
		},
		{
			name: "query_1_param",
			params: &graphql.RawParams{
				Query: fragment + `
query GetUsersConnection($first: Int) {
	users(first: $first) {
		... userConnection
	}
}
`,
				OperationName: "GetUsersConnection",
				Variables: map[string]any{
					"first": 1,
				},
			},
		},
		{
			name: "query_2_params",
			params: &graphql.RawParams{
				Query: fragment + `
query GetUsersConnection($query: String, $first: Int) {
	users(query: $query, first: $first) {
		... userConnection
	}
}
`,
				OperationName: "GetUsersConnection",
				Variables: map[string]any{
					"first": 1,
					"query": "john",
				},
			},
		},
		{
			name: "query_3_params",
			params: &graphql.RawParams{
				Query: fragment + `
query GetUsersConnection($query: String, $first: Int, $orderBy: UserOrderBy!) {
	users(query: $query, first: $first, orderBy: $orderBy) {
		... userConnection
	}
}
`,
				OperationName: "GetUsersConnection",
				Variables: map[string]any{
					"first": 1,
					"query": "john",
					"orderBy": map[string]any{
						"orderByField":     "FIRST_NAME",
						"orderByDirection": "DESCENDING",
					},
				},
			},
		},
		{
			name: "query_4_params",
			params: &graphql.RawParams{
				Query: fragment + `
query GetUsersConnection($query: String, $first: Int, $before: String, $orderBy: UserOrderBy!) {
	users(query: $query, first: $first, before: $before, orderBy: $orderBy) {
		... userConnection
	}
}
`,
				OperationName: "GetUsersConnection",
				Variables: map[string]any{
					"first":  1,
					"query":  "john",
					"before": "abc",
					"orderBy": map[string]any{
						"orderByField":     "FIRST_NAME",
						"orderByDirection": "DESCENDING",
					},
				},
			},
		},
	}

	b.StartTimer()

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ctx := graphql.StartOperationTrace(context.Background())
				opCtx, errs := exec.CreateOperationContext(ctx, benchmark.params)
				if errs != nil {
					b.Fatal(errs)
				}

				_, _ = exec.DispatchOperation(ctx, opCtx)
			}
		})
	}
}
