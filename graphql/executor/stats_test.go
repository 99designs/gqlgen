package executor_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor/testexecutor"
	"github.com/99designs/gqlgen/graphql/handler/lru"
)

// Stats carries the four timings a request produces: when it was read, when
// parsing ran, when validation ran, and when the operation started. Nothing in
// the executor reads them back, so they are easy to drop without any test
// noticing — and apollotracing computes durations from them as
// Parsing.End.Sub(Parsing.Start). A missing Start is therefore not a missing
// number but a wrong one, roughly two thousand years wide, reported to whatever
// consumes the trace.
//
// These tests assert the timings exist and are ordered. They deliberately do
// not stub graphql.Now: real time is enough to catch an unset field, because an
// unset time.Time is the zero value rather than a plausible instant, and
// avoiding the stub keeps these tests free of the package-level global that
// would make them unsafe to run in parallel.

// requireOrderedTimings asserts the four timings are present and in the order
// the request pipeline produces them.
func requireOrderedTimings(t *testing.T, stats *graphql.Stats) {
	t.Helper()

	require.False(t, stats.Parsing.Start.IsZero(), "Parsing.Start should be recorded")
	require.False(t, stats.Parsing.End.IsZero(), "Parsing.End should be recorded")
	require.False(t, stats.Validation.Start.IsZero(), "Validation.Start should be recorded")
	require.False(t, stats.Validation.End.IsZero(), "Validation.End should be recorded")

	assert.False(t, stats.Parsing.End.Before(stats.Parsing.Start),
		"parsing cannot end before it starts")
	assert.False(t, stats.Validation.Start.Before(stats.Parsing.End),
		"validation cannot start before parsing ends")
	assert.False(t, stats.Validation.End.Before(stats.Validation.Start),
		"validation cannot end before it starts")
}

// requireSaneDurations asserts the durations a tracing extension derives are
// plausible. This is the assertion that fails loudly on an unrecorded timing:
// subtracting a recorded instant from the zero time yields a duration measured
// in centuries, which is exactly what would otherwise reach a trace.
func requireSaneDurations(t *testing.T, stats *graphql.Stats) {
	t.Helper()

	// The same arithmetic apollotracing performs.
	parsing := stats.Parsing.End.Sub(stats.Parsing.Start)
	validation := stats.Validation.End.Sub(stats.Validation.Start)

	for name, d := range map[string]time.Duration{
		"parsing":    parsing,
		"validation": validation,
	} {
		assert.GreaterOrEqual(t, d, time.Duration(0), "%s duration must not be negative", name)
		assert.Less(t, d, time.Minute, "%s duration of %s is not a plausible measurement", name, d)
	}
}

func TestCreateOperationContextRecordsStats(t *testing.T) {
	t.Parallel()

	exec := testexecutor.New()
	ctx := graphql.StartOperationTrace(context.Background())
	readStart := time.Now()
	readEnd := readStart.Add(3 * time.Millisecond)

	opCtx, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{
		Query:    "{ name }",
		ReadTime: graphql.TraceTiming{Start: readStart, End: readEnd},
	})
	require.Empty(t, errs)

	requireOrderedTimings(t, &opCtx.Stats)
	requireSaneDurations(t, &opCtx.Stats)

	// Read timing is supplied by the transport rather than measured here, so it
	// has to survive onto the operation context unchanged.
	assert.True(t, readStart.Equal(opCtx.Stats.Read.Start))
	assert.True(t, readEnd.Equal(opCtx.Stats.Read.End))

	// OperationStart comes from the trace the transport began, so a request
	// that was traced must carry it.
	assert.False(t, opCtx.Stats.OperationStart.IsZero(),
		"OperationStart should come from StartOperationTrace")
	assert.True(t, graphql.GetStartTime(ctx).Equal(opCtx.Stats.OperationStart))
}

// A cache hit returns the parsed document without parsing or validating it, and
// still has to record the timings. The hit path sets Parsing.End and
// Validation.Start to one shared instant, which is what tells a trace that
// neither phase did work on this request.
func TestCreateOperationContextRecordsStatsOnACacheHit(t *testing.T) {
	t.Parallel()

	exec := testexecutor.New()
	exec.SetQueryCache(lru.New[*ast.QueryDocument](16))
	params := &graphql.RawParams{Query: "{ name }"}

	// The first request populates the cache; the second is the one under test.
	first, errs := exec.CreateOperationContext(
		graphql.StartOperationTrace(context.Background()), params)
	require.Empty(t, errs)
	requireOrderedTimings(t, &first.Stats)

	opCtx, errs := exec.CreateOperationContext(
		graphql.StartOperationTrace(context.Background()), params)
	require.Empty(t, errs)

	requireOrderedTimings(t, &opCtx.Stats)
	requireSaneDurations(t, &opCtx.Stats)

	// On the hit path both instants are read from one graphql.Now() call, so
	// parsing reports as having taken no time at all.
	assert.True(t, opCtx.Stats.Parsing.End.Equal(opCtx.Stats.Validation.Start),
		"a cache hit should close parsing and open validation at the same instant")
	assert.Zero(t, opCtx.Stats.Parsing.End.Sub(opCtx.Stats.Parsing.Start).Nanoseconds()/
		int64(time.Second),
		"a cache hit should not report seconds of parsing")
}

// A request that fails validation is still traced. Dropping the timings on the
// failure path would leave a tracing extension computing durations from a zero
// instant precisely when something has gone wrong and the trace matters most.
func TestCreateOperationContextRecordsStatsWhenValidationFails(t *testing.T) {
	t.Parallel()

	exec := testexecutor.New()
	opCtx, errs := exec.CreateOperationContext(
		graphql.StartOperationTrace(context.Background()),
		&graphql.RawParams{Query: "{ noSuchField }"},
	)
	require.NotEmpty(t, errs, "the fixture query should fail validation")
	require.NotNil(t, opCtx, "a failed request still returns its operation context")

	// Parsing completed, so its timings are whole. Validation started and then
	// failed, so its start is recorded even though its end is not.
	require.False(t, opCtx.Stats.Parsing.Start.IsZero(), "Parsing.Start should be recorded")
	require.False(t, opCtx.Stats.Parsing.End.IsZero(), "Parsing.End should be recorded")
	assert.False(t, opCtx.Stats.Validation.Start.IsZero(), "Validation.Start should be recorded")
	assert.False(t, opCtx.Stats.Parsing.End.Before(opCtx.Stats.Parsing.Start))
}

// A request that fails to parse never reaches validation, so only the parsing
// start is meaningful. Asserting it separately keeps the earlier tests honest
// about which timings each path is actually expected to fill in.
func TestCreateOperationContextRecordsStatsWhenParsingFails(t *testing.T) {
	t.Parallel()

	exec := testexecutor.New()
	opCtx, errs := exec.CreateOperationContext(
		graphql.StartOperationTrace(context.Background()),
		&graphql.RawParams{Query: "{ this is not valid graphql"},
	)
	require.NotEmpty(t, errs, "the fixture query should fail to parse")
	require.NotNil(t, opCtx)

	assert.False(t, opCtx.Stats.Parsing.Start.IsZero(),
		"Parsing.Start is recorded before the parse is attempted")
}
