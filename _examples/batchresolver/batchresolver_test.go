package batchresolver

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

type gqlError struct {
	Message string `json:"message"`
	Path    []any  `json:"path"`
}

func newTestClient(r *Resolver) *client.Client {
	srv := handler.New(NewExecutableSchema(Config{Resolvers: r}))
	srv.AddTransport(transport.POST{})
	return client.New(srv)
}

func marshalJSON(t *testing.T, v any) string {
	t.Helper()
	blob, err := json.Marshal(v)
	require.NoError(t, err)
	return string(blob)
}

func requireErrorJSON(t *testing.T, err error, expected string) {
	t.Helper()
	require.Error(t, err)
	actual := normalizeErrorJSON(t, err.Error())
	expectedNorm := normalizeErrorJSON(t, expected)
	require.Equal(t, expectedNorm, actual)
}

func requireErrorListJSON(t *testing.T, errs gqlerror.List, expected string) {
	t.Helper()
	require.NotEmpty(t, errs)
	err := errors.New(marshalJSON(t, errs))
	requireErrorJSON(t, err, expected)
}

func normalizeErrorJSON(t *testing.T, jsonStr string) string {
	t.Helper()
	if jsonStr == "" {
		return ""
	}
	var list []gqlError
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &list))
	sort.Slice(list, func(i, j int) bool {
		return errorKey(t, list[i]) < errorKey(t, list[j])
	})
	blob, err := json.Marshal(list)
	require.NoError(t, err)
	return string(blob)
}

func errorKey(t *testing.T, err gqlError) string {
	t.Helper()
	blob, marshalErr := json.Marshal(err.Path)
	require.NoError(t, marshalErr)
	return err.Message + "|" + string(blob)
}

func TestBatchResolver_Parity_NoError(t *testing.T) {
	resolver := &Resolver{
		users:         []*User{{}, {}},
		profiles:      []*Profile{{ID: "p1"}, {ID: "p2"}},
		profileErrIdx: -1,
	}

	c := newTestClient(resolver)
	var resp struct {
		Users []struct {
			NullableBatch *struct {
				ID string `json:"id"`
			} `json:"nullableBatch"`
			NullableNonBatch *struct {
				ID string `json:"id"`
			} `json:"nullableNonBatch"`
		} `json:"users"`
	}

	err := c.Post(`query { users { nullableBatch { id } nullableNonBatch { id } } }`, &resp)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{"users":[{"nullableBatch":{"id":"p1"},"nullableNonBatch":{"id":"p1"}},{"nullableBatch":{"id":"p2"},"nullableNonBatch":{"id":"p2"}}]}`,
		marshalJSON(t, resp),
	)
}

func TestBatchResolver_Parity_Error(t *testing.T) {
	resolver := &Resolver{
		users:         []*User{{}, {}},
		profiles:      []*Profile{{ID: "p1"}, {ID: "p2"}},
		profileErrIdx: 1,
	}

	c := newTestClient(resolver)
	var resp struct {
		Users []struct {
			NullableBatch *struct {
				ID string `json:"id"`
			} `json:"nullableBatch"`
			NullableNonBatch *struct {
				ID string `json:"id"`
			} `json:"nullableNonBatch"`
		} `json:"users"`
	}

	err := c.Post(`query { users { nullableBatch { id } nullableNonBatch { id } } }`, &resp)
	requireErrorJSON(t, err, `[
		{"message":"profile error at index 1","path":["users",1,"nullableBatch"]},
		{"message":"profile error at index 1","path":["users",1,"nullableNonBatch"]}
	]`)
	require.JSONEq(
		t,
		`{"users":[{"nullableBatch":{"id":"p1"},"nullableNonBatch":{"id":"p1"}},{"nullableBatch":null,"nullableNonBatch":null}]}`,
		marshalJSON(t, resp),
	)
}

func TestBatchResolver_Parity_GqlErrorList(t *testing.T) {
	resolver := &Resolver{
		users:              []*User{{}, {}},
		profiles:           []*Profile{{ID: "p1"}, {ID: "p2"}},
		profileErrListIdxs: map[int]struct{}{0: {}},
		profileErrIdx:      -1,
	}

	c := newTestClient(resolver)
	var resp struct {
		Users []struct {
			NullableBatch *struct {
				ID string `json:"id"`
			} `json:"nullableBatch"`
			NullableNonBatch *struct {
				ID string `json:"id"`
			} `json:"nullableNonBatch"`
		} `json:"users"`
	}

	err := c.Post(`query { users { nullableBatch { id } nullableNonBatch { id } } }`, &resp)
	requireErrorJSON(t, err, `[
		{"message":"profile list error 1 at index 0","path":["users",0,"nullableBatch"]},
		{"message":"profile list error 2 at index 0","path":["users",0,"nullableBatch"]},
		{"message":"profile list error 1 at index 0","path":["users",0,"nullableNonBatch"]},
		{"message":"profile list error 2 at index 0","path":["users",0,"nullableNonBatch"]}
	]`)
	require.JSONEq(
		t,
		`{"users":[{"nullableBatch":null,"nullableNonBatch":null},{"nullableBatch":{"id":"p2"},"nullableNonBatch":{"id":"p2"}}]}`,
		marshalJSON(t, resp),
	)
}

func TestBatchResolver_Parity_GqlErrorPathNil(t *testing.T) {
	resolver := &Resolver{
		users:                   []*User{{}, {}},
		profiles:                []*Profile{{ID: "p1"}, {ID: "p2"}},
		profileGqlErrNoPathIdxs: map[int]struct{}{0: {}},
		profileErrIdx:           -1,
	}

	c := newTestClient(resolver)
	var resp struct {
		Users []struct {
			NullableBatch *struct {
				ID string `json:"id"`
			} `json:"nullableBatch"`
			NullableNonBatch *struct {
				ID string `json:"id"`
			} `json:"nullableNonBatch"`
		} `json:"users"`
	}

	err := c.Post(`query { users { nullableBatch { id } nullableNonBatch { id } } }`, &resp)
	requireErrorJSON(t, err, `[
		{"message":"profile gqlerror path nil at index 0","path":["users",0,"nullableBatch"]},
		{"message":"profile gqlerror path nil at index 0","path":["users",0,"nullableNonBatch"]}
	]`)
	require.JSONEq(
		t,
		`{"users":[{"nullableBatch":null,"nullableNonBatch":null},{"nullableBatch":{"id":"p2"},"nullableNonBatch":{"id":"p2"}}]}`,
		marshalJSON(t, resp),
	)
}

func TestBatchResolver_Parity_GqlErrorPathSet(t *testing.T) {
	resolver := &Resolver{
		users:                 []*User{{}, {}},
		profiles:              []*Profile{{ID: "p1"}, {ID: "p2"}},
		profileGqlErrPathIdxs: map[int]struct{}{0: {}},
		profileErrIdx:         -1,
	}

	c := newTestClient(resolver)
	var resp struct {
		Users []struct {
			NullableBatch *struct {
				ID string `json:"id"`
			} `json:"nullableBatch"`
			NullableNonBatch *struct {
				ID string `json:"id"`
			} `json:"nullableNonBatch"`
		} `json:"users"`
	}

	err := c.Post(`query { users { nullableBatch { id } nullableNonBatch { id } } }`, &resp)
	requireErrorJSON(t, err, `[
		{"message":"profile gqlerror path set at index 0","path":["custom",0]},
		{"message":"profile gqlerror path set at index 0","path":["custom",0]}
	]`)
	require.JSONEq(
		t,
		`{"users":[{"nullableBatch":null,"nullableNonBatch":null},{"nullableBatch":{"id":"p2"},"nullableNonBatch":{"id":"p2"}}]}`,
		marshalJSON(t, resp),
	)
}

func TestBatchResolver_Parity_PartialResponseWithErrValue(t *testing.T) {
	resolver := &Resolver{
		users:                   []*User{{}, {}},
		profiles:                []*Profile{{ID: "p1"}, {ID: "p2"}},
		profileErrWithValueIdxs: map[int]struct{}{0: {}},
		profileErrIdx:           -1,
	}

	c := newTestClient(resolver)
	var resp struct {
		Users []struct {
			NullableBatch *struct {
				ID string `json:"id"`
			} `json:"nullableBatch"`
			NullableNonBatch *struct {
				ID string `json:"id"`
			} `json:"nullableNonBatch"`
		} `json:"users"`
	}

	err := c.Post(`query { users { nullableBatch { id } nullableNonBatch { id } } }`, &resp)
	requireErrorJSON(t, err, `[
		{"message":"profile error with value at index 0","path":["users",0,"nullableBatch"]},
		{"message":"profile error with value at index 0","path":["users",0,"nullableNonBatch"]}
	]`)
	require.JSONEq(
		t,
		`{"users":[{"nullableBatch":null,"nullableNonBatch":null},{"nullableBatch":{"id":"p2"},"nullableNonBatch":{"id":"p2"}}]}`,
		marshalJSON(t, resp),
	)
}

func TestBatchResolver_Parity_NonNullPropagation(t *testing.T) {
	resolver := &Resolver{
		users:         []*User{{}, {}},
		profiles:      []*Profile{{ID: "p1"}, {ID: "p2"}},
		profileErrIdx: 0,
	}

	c := newTestClient(resolver)
	var resp struct {
		Users []struct {
			NonNullableBatch *struct {
				ID string `json:"id"`
			} `json:"nonNullableBatch"`
			NonNullableNonBatch *struct {
				ID string `json:"id"`
			} `json:"nonNullableNonBatch"`
		} `json:"users"`
	}

	err := c.Post(`query { users { nonNullableBatch { id } nonNullableNonBatch { id } } }`, &resp)
	requireErrorJSON(t, err, `[
		{"message":"profile error at index 0","path":["users",0,"nonNullableBatch"]},
		{"message":"profile error at index 0","path":["users",0,"nonNullableNonBatch"]}
	]`)
	require.JSONEq(
		t,
		`{"users":null}`,
		marshalJSON(t, resp),
	)
}

func TestBatchResolver_InvalidLen_AddsErrorPerParent(t *testing.T) {
	resolver := &Resolver{
		users:           []*User{{}, {}},
		profiles:        []*Profile{{ID: "p1"}, {ID: "p2"}},
		profileErrIdx:   -1,
		profileWrongLen: true,
	}

	c := newTestClient(resolver)
	var resp struct {
		Users []struct {
			NullableBatch *struct {
				ID string `json:"id"`
			} `json:"nullableBatch"`
		} `json:"users"`
	}

	err := c.Post(`query { users { nullableBatch { id } } }`, &resp)
	requireErrorJSON(t, err, `[
		{"message":"index 0: batch resolver User.nullableBatch returned 1 results for 2 parents","path":["users",0,"nullableBatch"]},
		{"message":"index 1: batch resolver User.nullableBatch returned 1 results for 2 parents","path":["users",1,"nullableBatch"]}
	]`)
	require.JSONEq(
		t,
		`{"users":[{"nullableBatch":null},{"nullableBatch":null}]}`,
		marshalJSON(t, resp),
	)
}

func TestBatchResolver_InvalidIndex_AddsError(t *testing.T) {
	// NOTE: This error path is only reachable by internal execution context misuse,
	// not by normal GraphQL query execution.
	resolver := &Resolver{
		profiles:      []*Profile{{ID: "p1"}},
		profileErrIdx: -1,
	}
	schema := NewExecutableSchema(Config{Resolvers: resolver}).(*executableSchema)
	ec := executionContext{OperationContext: nil, executableSchema: schema}

	parents := []*User{{}}
	ctx := graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, nil)
	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("users"))
	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithIndex(2))
	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("nullableBatch"))
	ctx = ec.withBatchParents(ctx, "User", parents)

	_, _ = ec.resolveBatch_User_nullableBatch(
		ctx,
		graphql.CollectedField{Field: &ast.Field{Name: "nullableBatch"}},
		parents[0],
	)

	requireErrorListJSON(t, graphql.GetErrors(ctx), `[
		{"message":"batch resolver User.nullableBatch could not resolve parent index 2","path":["users",2,"nullableBatch"]}
	]`)
}

func TestBatchResolver_InvalidType_AddsError(t *testing.T) {
	// NOTE: This error path is only reachable by internal execution context misuse,
	// not by normal GraphQL query execution.
	resolver := &Resolver{
		profiles:      []*Profile{{ID: "p1"}},
		profileErrIdx: -1,
	}
	schema := NewExecutableSchema(Config{Resolvers: resolver}).(*executableSchema)
	ec := executionContext{OperationContext: nil, executableSchema: schema}

	parents := []*User{{}}
	ctx := graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, nil)
	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("users"))
	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithIndex(0))
	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("nullableBatch"))
	ctx = ec.withBatchParents(ctx, "User", parents)

	group := ec.getBatchParentGroup(ctx, "User")
	badResult := &batchFieldResult{done: make(chan struct{})}
	badResult.results = []BatchResult[string]{{Value: "oops"}}
	badResult.once.Do(func() {})
	close(badResult.done)
	group.fields.Store("nullableBatch", badResult)

	_, _ = ec.resolveBatch_User_nullableBatch(
		ctx,
		graphql.CollectedField{Field: &ast.Field{Name: "nullableBatch"}},
		parents[0],
	)

	requireErrorListJSON(t, graphql.GetErrors(ctx), `[
		{"message":"batch resolver User.nullableBatch returned unexpected result type (index 0)","path":["users",0,"nullableBatch"]}
	]`)
}
