package batchresolver

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/codegen/config"
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

	err := c.Post(`
query {
  users {
    nullableBatchWithArg(offset: 1) { id }
    nullableNonBatchWithArg(offset: 1) { id }
  }
}`, &resp)
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

func TestBatchResolver_BatchErrors_ErrLenMismatch_AddsErrorPerParent(t *testing.T) {
	cases := []struct {
		name   string
		errLen int
	}{
		{name: "len1", errLen: 1},
		{name: "len0", errLen: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolver := &Resolver{
				users:             []*User{{}, {}},
				profiles:          []*Profile{{ID: "p1"}, {ID: "p2"}},
				profileErrIdx:     -1,
				batchErrsWrongLen: true,
				batchErrsLen:      tc.errLen,
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
			requireErrorJSON(t, err, fmt.Sprintf(`[
				{"message":"index 0: batch resolver User.nullableBatch returned %d errors for 2 parents","path":["users",0,"nullableBatch"]},
				{"message":"index 1: batch resolver User.nullableBatch returned %d errors for 2 parents","path":["users",1,"nullableBatch"]}
			]`, tc.errLen, tc.errLen))
			require.JSONEq(
				t,
				`{"users":[{"nullableBatch":null},{"nullableBatch":null}]}`,
				marshalJSON(t, resp),
			)
		})
	}
}

func TestBatchResolver_BatchErrors_ResultLenMismatch_AddsErrorPerParent(t *testing.T) {
	resolver := &Resolver{
		users:                []*User{{}, {}},
		profiles:             []*Profile{{ID: "p1"}, {ID: "p2"}},
		profileErrIdx:        -1,
		batchResultsWrongLen: true,
		batchResultsLen:      1,
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

func TestBatchDirectiveConfig(t *testing.T) {
	cfg, err := config.LoadConfig("gqlgen.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.Init())

	userFields := cfg.Models["User"].Fields

	// YAML-configured fields
	require.True(t, userFields["nullableBatch"].Batch)
	require.True(t, userFields["nullableBatchWithArg"].Batch)
	require.True(t, userFields["nonNullableBatch"].Batch)

	require.False(t, userFields["nullableNonBatch"].Batch)
	require.False(t, userFields["nullableNonBatchWithArg"].Batch)
	require.False(t, userFields["nonNullableNonBatch"].Batch)

	// Directive-configured fields
	require.True(t, userFields["directiveNullableBatch"].Batch)
	require.True(t, userFields["directiveNullableBatchWithArg"].Batch)
	require.True(t, userFields["directiveNonNullableBatch"].Batch)

	require.False(t, userFields["directiveNullableNonBatch"].Batch)
	require.False(t, userFields["directiveNullableNonBatchWithArg"].Batch)
	require.False(t, userFields["directiveNonNullableNonBatch"].Batch)
}

func TestBatchResolver_BatchErrors_ListPerIndex_AddsMultipleErrors(t *testing.T) {
	resolver := &Resolver{
		users:            []*User{{}, {}},
		profiles:         []*Profile{{ID: "p1"}, {ID: "p2"}},
		profileErrIdx:    -1,
		batchErrListIdxs: map[int]struct{}{0: {}},
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
		{"message":"batch list error 1 at index 0","path":["users",0,"nullableBatch"]},
		{"message":"batch list error 2 at index 0","path":["users",0,"nullableBatch"]}
	]`)
	require.JSONEq(
		t,
		`{"users":[{"nullableBatch":null},{"nullableBatch":{"id":"p2"}}]}`,
		marshalJSON(t, resp),
	)
}

func TestBatchResolver_Nested_CallCount(t *testing.T) {
	const n = 10
	users := make([]*User, n)
	profiles := make([]*Profile, n)
	images := make([]*Image, n)
	for i := 0; i < n; i++ {
		users[i] = &User{}
		profiles[i] = &Profile{ID: fmt.Sprintf("p%d", i)}
		images[i] = &Image{URL: fmt.Sprintf("https://img/%d", i)}
	}
	resolver := &Resolver{
		users:         users,
		profiles:      profiles,
		images:        images,
		profileErrIdx: -1,
	}
	client := newTestClient(resolver)

	type graphqlResp struct {
		Users []struct {
			Profile *struct {
				ID    string `json:"id"`
				Cover *struct {
					URL string `json:"url"`
				} `json:"cover"`
			} `json:"profile"`
		} `json:"users"`
	}

	assertData := func(t *testing.T, resp graphqlResp, label string) {
		t.Helper()
		require.Len(t, resp.Users, n)
		for i, u := range resp.Users {
			require.NotNil(t, u.Profile, "%s user %d profile nil", label, i)
			require.Equal(t, fmt.Sprintf("p%d", i), u.Profile.ID)
			require.NotNil(t, u.Profile.Cover, "%s user %d cover nil", label, i)
			require.Equal(t, fmt.Sprintf("https://img/%d", i), u.Profile.Cover.URL)
		}
	}

	// --- Batch path ---

	var batchResp graphqlResp
	err := client.Post(`query {
		users {
			profile: profileBatch {
				id
				cover: coverBatch {
					url
				}
			}
		}
	}`, &batchResp)
	require.NoError(t, err)
	assertData(t, batchResp, "batch")
	require.Equal(
		t,
		int32(1),
		resolver.profileBatchCalls.Load(),
		"profileBatch should be called once for all users",
	)
	// TODO: coverBatch is called once per profile (not batched) because profiles
	// are resolved as individual values, not as a list. The batch parent context
	// for "Profile" is only set when marshalling a [Profile] list field.
	// Nested batching should propagate the batch parent context from batch
	// resolver results so coverBatchCalls == 1 here.
	require.Equal(
		t,
		int32(n),
		resolver.coverBatchCalls.Load(),
		"coverBatch called once per profile (no list parent context)",
	)

	// --- Non-batch path ---
	var nonBatchResp graphqlResp
	err = client.Post(`query {
		users {
			profile: profileNonBatch {
				id
				cover: coverNonBatch {
					url
				}
			}
		}
	}`, &nonBatchResp)
	require.NoError(t, err)
	assertData(t, nonBatchResp, "non-batch")
	require.Equal(
		t,
		int32(n),
		resolver.profileNonBatchCalls.Load(),
		"profileNonBatch should be called once per user",
	)
	require.Equal(
		t,
		int32(n),
		resolver.coverNonBatchCalls.Load(),
		"coverNonBatch should be called once per profile",
	)

	// --- Verify both paths produce identical data ---
	require.Equal(
		t,
		marshalJSON(t, batchResp),
		marshalJSON(t, nonBatchResp),
		"batch and non-batch should return identical data",
	)
}

func TestBatchResolver_Nested_Connection_CallCount(t *testing.T) {
	const n = 10
	users := make([]*User, n)
	profiles := make([]*Profile, n)
	images := make([]*Image, n)
	for i := 0; i < n; i++ {
		users[i] = &User{}
		profiles[i] = &Profile{ID: fmt.Sprintf("p%d", i)}
		images[i] = &Image{URL: fmt.Sprintf("https://img/%d", i)}
	}
	resolver := &Resolver{
		users:         users,
		profiles:      profiles,
		images:        images,
		profileErrIdx: -1,
	}
	client := newTestClient(resolver)

	type graphqlResp struct {
		Users []struct {
			Conn *struct {
				Edges []struct {
					Node *struct {
						ID    string `json:"id"`
						Cover *struct {
							URL string `json:"url"`
						} `json:"cover"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"conn"`
		} `json:"users"`
	}

	assertData := func(t *testing.T, resp graphqlResp, label string) {
		t.Helper()
		require.Len(t, resp.Users, n)
		for i, u := range resp.Users {
			require.NotNil(t, u.Conn, "%s user %d connection nil", label, i)
			require.Len(t, u.Conn.Edges, 1, "%s user %d edges", label, i)
			node := u.Conn.Edges[0].Node
			require.NotNil(t, node, "%s user %d node nil", label, i)
			require.Equal(t, fmt.Sprintf("p%d", i), node.ID)
			require.NotNil(t, node.Cover, "%s user %d cover nil", label, i)
			require.Equal(t, fmt.Sprintf("https://img/%d", i), node.Cover.URL)
		}
	}

	// --- Batch path ---

	var batchResp graphqlResp
	err := client.Post(`query {
		users {
			conn: profileConnectionBatch {
				edges {
					node {
						id
						cover: coverBatch { url }
					}
				}
			}
		}
	}`, &batchResp)
	require.NoError(t, err)
	assertData(t, batchResp, "batch")
	require.Equal(
		t,
		int32(1),
		resolver.profileConnectionBatchCalls.Load(),
		"profileConnectionBatch should be called once for all users",
	)
	// TODO: coverBatch is not batched because the immediate parent (Profile)
	// and its edge are not batched — only the connection is. This should be 1
	// once nested batching propagates through non-batched intermediate types.
	require.Equal(
		t,
		int32(n),
		resolver.coverBatchCalls.Load(),
		"coverBatch called once per profile (immediate parent not batched)",
	)

	// --- Non-batch path ---

	var nonBatchResp graphqlResp
	err = client.Post(`query {
		users {
			conn: profileConnectionNonBatch {
				edges {
					node {
						id
						cover: coverNonBatch { url }
					}
				}
			}
		}
	}`, &nonBatchResp)
	require.NoError(t, err)
	assertData(t, nonBatchResp, "non-batch")
	require.Equal(
		t,
		int32(n),
		resolver.profileConnectionNonBatchCalls.Load(),
		"profileConnectionNonBatch should be called once per user",
	)
	require.Equal(
		t,
		int32(n),
		resolver.coverNonBatchCalls.Load(),
		"coverNonBatch should be called once per profile",
	)

	// --- Verify both paths produce identical data ---
	require.Equal(
		t,
		marshalJSON(t, batchResp),
		marshalJSON(t, nonBatchResp),
		"batch and non-batch should return identical data",
	)
}

func BenchmarkBatchResolver_SingleLevel(b *testing.B) {
	const n = 100
	users := make([]*User, n)
	profiles := make([]*Profile, n)
	for i := 0; i < n; i++ {
		users[i] = &User{}
		profiles[i] = &Profile{ID: fmt.Sprintf("p%d", i)}
	}

	b.Run("batch", func(b *testing.B) {
		resolver := &Resolver{
			users:         users,
			profiles:      profiles,
			profileErrIdx: -1,
		}
		c := newTestClient(resolver)
		var resp json.RawMessage
		for b.Loop() {
			_ = c.Post(`query { users { nullableBatch { id } } }`, &resp)
		}
	})

	b.Run("non-batch", func(b *testing.B) {
		resolver := &Resolver{
			users:         users,
			profiles:      profiles,
			profileErrIdx: -1,
		}
		c := newTestClient(resolver)
		var resp json.RawMessage
		for b.Loop() {
			_ = c.Post(`query { users { nullableNonBatch { id } } }`, &resp)
		}
	})
}

func BenchmarkBatchResolver_Nested(b *testing.B) {
	const n = 100
	users := make([]*User, n)
	profiles := make([]*Profile, n)
	images := make([]*Image, n)
	for i := 0; i < n; i++ {
		users[i] = &User{}
		profiles[i] = &Profile{ID: fmt.Sprintf("p%d", i)}
		images[i] = &Image{URL: fmt.Sprintf("https://img/%d", i)}
	}

	b.Run("batch", func(b *testing.B) {
		resolver := &Resolver{
			users:         users,
			profiles:      profiles,
			images:        images,
			profileErrIdx: -1,
		}
		c := newTestClient(resolver)
		var resp json.RawMessage
		for b.Loop() {
			_ = c.Post(`query {
				users {
					profile: profileBatch {
						id
						cover: coverBatch { url }
					}
				}
			}`, &resp)
		}
	})

	b.Run("non-batch", func(b *testing.B) {
		resolver := &Resolver{
			users:         users,
			profiles:      profiles,
			images:        images,
			profileErrIdx: -1,
		}
		c := newTestClient(resolver)
		var resp json.RawMessage
		for b.Loop() {
			_ = c.Post(`query {
				users {
					profile: profileNonBatch {
						id
						cover: coverNonBatch { url }
					}
				}
			}`, &resp)
		}
	})
}
