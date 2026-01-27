package batchresolver

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
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

func TestBatchResolver_Parity_WithArgs(t *testing.T) {
	resolver := &Resolver{
		users:         []*User{{}, {}},
		profiles:      []*Profile{{ID: "p1"}, {ID: "p2"}, {ID: "p3"}},
		profileErrIdx: -1,
	}

	c := newTestClient(resolver)
	var resp struct {
		Users []struct {
			NullableBatchWithArg *struct {
				ID string `json:"id"`
			} `json:"nullableBatchWithArg"`
			NullableNonBatchWithArg *struct {
				ID string `json:"id"`
			} `json:"nullableNonBatchWithArg"`
		} `json:"users"`
	}

	err := c.Post(`query { users { nullableBatchWithArg(offset: 1) { id } nullableNonBatchWithArg(offset: 1) { id } } }`, &resp)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{"users":[{"nullableBatchWithArg":{"id":"p2"},"nullableNonBatchWithArg":{"id":"p2"}},{"nullableBatchWithArg":{"id":"p3"},"nullableNonBatchWithArg":{"id":"p3"}}]}`,
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
