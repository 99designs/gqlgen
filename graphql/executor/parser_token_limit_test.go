package executor_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/99designs/gqlgen/graphql/executor/testexecutor"
)

// SetParserTokenLimit caps how many tokens the parser will read before giving
// up, which makes it a denial-of-service guard: without it, a single small
// request can ask the server to build an enormous document. The limit is
// enforced during parsing, so it rejects a query before any validation or
// execution cost is paid, and a query cache cannot shorten the path for a
// request that never parses.
//
// These tests pin that a breach is reported as a parse failure rather than
// sailing through as a partial document, because parseQuery decides that by
// type-asserting the parser's error and falling through when the assertion
// does not hold.

// aliasedQuery returns a valid query selecting `name` n times under distinct
// aliases. Each alias costs three tokens, so n controls the token count without
// changing whether the query is valid against the schema.
func aliasedQuery(n int) string {
	var sb strings.Builder
	sb.WriteString("query limited {")
	for i := range n {
		fmt.Fprintf(&sb, " a%d: name", i)
	}
	sb.WriteString(" }")
	return sb.String()
}

func TestExecutorParserTokenLimit(t *testing.T) {
	// The fixture has roughly 32 tokens, so 20 is comfortably under it and
	// 1000 comfortably over. Both were confirmed against the parser rather
	// than derived from counting by hand.
	const fields = 10

	cases := map[string]struct {
		limit     int
		setLimit  bool
		wantError bool
	}{
		"unset leaves the parser unlimited": {
			setLimit:  false,
			wantError: false,
		},
		"an explicit zero means unlimited": {
			limit:     0,
			setLimit:  true,
			wantError: false,
		},
		"a limit above the query's token count admits it": {
			limit:     1000,
			setLimit:  true,
			wantError: false,
		},
		"a limit below the query's token count rejects it": {
			limit:     20,
			setLimit:  true,
			wantError: true,
		},
		"a limit of one rejects everything": {
			limit:     1,
			setLimit:  true,
			wantError: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			exec := testexecutor.New()
			if tc.setLimit {
				exec.SetParserTokenLimit(tc.limit)
			}

			resp := query(exec, "", aliasedQuery(fields))

			if !tc.wantError {
				require.Empty(t, resp.Errors, "the query should have been admitted")
				assert.JSONEq(t, `{"name":"test"}`, string(resp.Data))
				return
			}

			assert.Empty(t, string(resp.Data), "a rejected query must produce no data")
			require.Len(t, resp.Errors, 1)
			// The code is what a client and the errcode-based metrics key on, so
			// a breach has to classify as a parse failure and not as a
			// validation failure or an unlabelled internal error.
			assert.Equal(t, errcode.ParseFailed, resp.Errors[0].Extensions["code"])
			// Naming the limit in the message is what lets an operator tell a
			// limit that is too tight from a query that is too large.
			assert.Contains(t, resp.Errors[0].Message, strconv.Itoa(tc.limit))
		})
	}
}

// The limit has to hold for every request, not just the first. A limit stored
// somewhere consumed by parsing would let the second request through, which is
// exactly the shape of bug a denial-of-service guard cannot afford.
func TestExecutorParserTokenLimitAppliesToEveryRequest(t *testing.T) {
	exec := testexecutor.New()
	exec.SetParserTokenLimit(20)
	q := aliasedQuery(10)

	for attempt := range 3 {
		resp := query(exec, "", q)
		require.Len(t, resp.Errors, 1, "attempt %d should have been rejected", attempt)
		assert.Equal(t, errcode.ParseFailed, resp.Errors[0].Extensions["code"],
			"attempt %d", attempt)
	}
}

// Raising the limit after it has rejected a query admits the same query, which
// shows the setter changes behaviour rather than latching on first use.
func TestExecutorParserTokenLimitIsNotLatched(t *testing.T) {
	exec := testexecutor.New()
	q := aliasedQuery(10)

	exec.SetParserTokenLimit(20)
	require.NotEmpty(t, query(exec, "", q).Errors, "the tight limit should reject")

	exec.SetParserTokenLimit(1000)
	resp := query(exec, "", q)
	require.Empty(t, resp.Errors, "the raised limit should admit the same query")
	assert.JSONEq(t, `{"name":"test"}`, string(resp.Data))
}

// A query small enough to pass the limit still has to be validated. Pinning
// this keeps the limit from being mistaken for a substitute for validation:
// the two reject different things, and a breach of neither should be reported
// as a breach of the other.
func TestExecutorParserTokenLimitDoesNotBypassValidation(t *testing.T) {
	exec := testexecutor.New()
	exec.SetParserTokenLimit(1000)

	resp := query(exec, "", "{ noSuchField }")
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, errcode.ValidationFailed, resp.Errors[0].Extensions["code"])
}
