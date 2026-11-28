// Package exectest holds the helpers that the tests of execbehavior and of its
// subpackages share.
package exectest

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
)

// RawPost sends query with variables to srv as a JSON POST request, and returns the
// recorded response as it is.
func RawPost(
	t testing.TB,
	srv http.Handler,
	query string,
	variables map[string]any,
) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

// Post sends query with variables to srv and returns the response body as JSON, with the
// errors normalized by NormalizeErrors, so that tests can compare it as a whole.
func Post(t testing.TB, srv http.Handler, query string, variables map[string]any) string {
	t.Helper()

	w := RawPost(t, srv, query, variables)
	var resp graphql.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), w.Body.String())
	NormalizeErrors(resp.Errors)
	out, err := json.Marshal(resp)
	require.NoError(t, err)
	return string(out)
}

// NormalizeErrors removes the locations of errs and sorts them by message and then by
// path, so that they do not depend on the order in which concurrent fields failed.
func NormalizeErrors(errs gqlerror.List) {
	for _, err := range errs {
		err.Locations = nil
	}
	slices.SortStableFunc(errs, func(a, b *gqlerror.Error) int {
		return cmp.Or(cmp.Compare(a.Message, b.Message), ComparePaths(a.Path, b.Path))
	})
}

// ComparePaths orders paths element by element, with list indices compared as numbers
// and field names before indices.
func ComparePaths(a, b ast.Path) int {
	for i := range min(len(a), len(b)) {
		ai, aIsIndex := a[i].(ast.PathIndex)
		bi, bIsIndex := b[i].(ast.PathIndex)
		var c int
		switch {
		case aIsIndex && bIsIndex:
			c = cmp.Compare(ai, bi)
		case aIsIndex != bIsIndex:
			if aIsIndex {
				c = 1
			} else {
				c = -1
			}
		default:
			c = cmp.Compare(a[i].(ast.PathName), b[i].(ast.PathName))
		}
		if c != 0 {
			return c
		}
	}
	return cmp.Compare(len(a), len(b))
}

// Subscribe runs the subscription query on exec and returns its responses as a JSON
// array, without the locations of errors, or, when running it panics, the value of the
// panic as a string.
func Subscribe(t testing.TB, exec *executor.Executor, query string) (res string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			b, err := json.Marshal(fmt.Sprint("panic: ", r))
			require.NoError(t, err)
			res = string(b)
		}
	}()

	ctx := graphql.StartOperationTrace(context.Background())
	opCtx, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{Query: query})
	require.Empty(t, errs)
	handler, ctx := exec.DispatchOperation(ctx, opCtx)

	// The handler reuses the buffer of the data, so each response is encoded before the
	// next one is read.
	var responses []json.RawMessage
	for range 100 {
		resp := handler(ctx)
		if resp == nil {
			break
		}
		for _, err := range resp.Errors {
			err.Locations = nil
		}
		b, err := json.Marshal(resp)
		require.NoError(t, err)
		responses = append(responses, b)
	}
	b, err := json.Marshal(responses)
	require.NoError(t, err)
	return string(b)
}
