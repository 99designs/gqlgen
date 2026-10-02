package proptests

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"github.com/99designs/gqlgen/graphql"
)

// Every scalar in this package has a Marshal/Unmarshal pair, and the pair has
// one job between them: a value a resolver returns has to survive being written
// into a response and read back out of a request. The existing tests check each
// direction against fixed examples, which leaves the composition unchecked —
// and the composition is where a scalar actually fails, usually at a boundary
// no hand-written example happens to name.
//
// These are round-trip properties over generated values: for all v,
// Unmarshal(wire(Marshal(v))) equals v. `wire` is not a shortcut; it is the
// transport's own decoding, UseNumber included, so the property describes the
// path a real request takes rather than an idealised one.

// overTheWire marshals the way a resolver writes a field into a response, then
// decodes the way graphql/handler/transport decodes an incoming variable.
//
// The UseNumber call matters and is copied from transport.jsonDecode: numbers
// reach an Unmarshal function as json.Number rather than float64, which is what
// lets a 64-bit integer survive a round trip at all.
func overTheWire(t *rapid.T, m graphql.Marshaler) any {
	t.Helper()

	var buf bytes.Buffer
	m.MarshalGQL(&buf)

	dec := json.NewDecoder(&buf)
	dec.UseNumber()
	var decoded any
	require.NoError(t, dec.Decode(&decoded),
		"a marshaled scalar must be valid JSON, got %q", buf.String())
	return decoded
}

// roundTrip builds the property that marshalling then unmarshalling a generated
// value returns that value. equal is supplied because not every scalar type is
// usefully comparable with require.Equal: a time.Time carries a location and a
// monotonic reading that the wire format does not.
func roundTrip[T any](
	gen *rapid.Generator[T],
	marshal func(T) graphql.Marshaler,
	unmarshal func(any) (T, error),
	equal func(t *rapid.T, want, got T),
) func(*rapid.T) {
	return func(t *rapid.T) {
		want := gen.Draw(t, "value")
		got, err := unmarshal(overTheWire(t, marshal(want)))
		require.NoError(t, err, "unmarshalling the marshalled form of %#v", want)
		equal(t, want, got)
	}
}

// equalValue is the comparison for every scalar whose Go type means what it
// says on the wire.
func equalValue[T any](t *rapid.T, want, got T) {
	t.Helper()
	require.Equal(t, want, got)
}

// equalInstant compares two times by the instant they name. RFC3339 records an
// offset rather than a zone name and carries no monotonic reading, so a
// round-tripped time is a different time.Time value denoting the same moment.
func equalInstant(t *rapid.T, want, got time.Time) {
	t.Helper()
	require.True(t, want.Equal(got), "want %s, got %s", want, got)
}

// finiteFloat generates float64 values that JSON can represent. Infinities and
// NaN are deliberately excluded here and covered by their own test below,
// because they are not a round-trip question: they do not survive the encoding
// at all.
func finiteFloat() *rapid.Generator[float64] {
	return rapid.Float64().Filter(func(f float64) bool {
		return !math.IsInf(f, 0) && !math.IsNaN(f)
	})
}

// utcTime generates instants with nanosecond precision inside the range
// RFC3339 can express. The year bound keeps generated values inside four-digit
// years, which the format requires.
func utcTime() *rapid.Generator[time.Time] {
	return rapid.Custom(func(t *rapid.T) time.Time {
		sec := rapid.Int64Range(0, 4102444800).Draw(t, "sec") // 1970 to 2100
		nsec := rapid.Int64Range(0, int64(time.Second)-1).Draw(t, "nsec")
		return time.Unix(sec, nsec).UTC()
	})
}

// utcDate generates midnight-UTC instants, which is the only shape graphql.MarshalDate
// round-trips: it formats date components only, so any time of day is
// discarded on the way out.
func utcDate() *rapid.Generator[time.Time] {
	return rapid.Custom(func(t *rapid.T) time.Time {
		days := rapid.Int64Range(0, 47482).Draw(t, "days") // 1970 to 2100
		return time.Unix(days*86400, 0).UTC()
	})
}

func uuidValue() *rapid.Generator[uuid.UUID] {
	return rapid.Custom(func(t *rapid.T) uuid.UUID {
		b := rapid.SliceOfN(rapid.Byte(), 16, 16).Draw(t, "bytes")
		id, err := uuid.FromBytes(b)
		require.NoError(t, err)
		return id
	})
}

// durationNanos generates every time.Duration except the minimum.
//
// math.MinInt64 is excluded because it does not round-trip, and the reason is
// upstream: github.com/sosodev/duration formats it as "-PT-9223372036.854776S",
// having negated a value whose negation overflows, and then refuses to parse
// its own output. TestMarshalDurationFailsAtMinInt64 records that, so this
// exclusion is documented by a test rather than by a comment alone.
func durationNanos() *rapid.Generator[int64] {
	return rapid.Int64Range(math.MinInt64+1, math.MaxInt64)
}

// jsonValue generates values already in the shape a JSON decoder produces:
// strings, bools, nil, json.Number, and maps and slices of those. graphql.MarshalMap and
// graphql.MarshalAny encode whatever they are given and their Unmarshal counterparts
// hand the decoded form straight back, so a round trip is only an identity over
// values that are already decoded JSON. A Go int put in would come back as a
// json.Number, which is a lossy conversion rather than a bug.
func jsonValue(depth int) *rapid.Generator[any] {
	gens := []*rapid.Generator[any]{
		rapid.Map(rapid.String(), func(s string) any { return s }),
		rapid.Map(rapid.Bool(), func(b bool) any { return b }),
		rapid.Just[any](nil),
		rapid.Map(rapid.Int64(), func(i int64) any {
			return json.Number(strconv.FormatInt(i, 10))
		}),
	}
	if depth > 0 {
		gens = append(gens,
			rapid.Map(
				rapid.MapOfN(rapid.String(), jsonValue(depth-1), 0, 3),
				func(m map[string]any) any { return m },
			),
			rapid.Map(
				rapid.SliceOfN(jsonValue(depth-1), 0, 3),
				func(s []any) any { return s },
			),
		)
	}
	return rapid.OneOf(gens...)
}

// TestScalarRoundTripProperties is the whole pair surface, one subtest each.
func TestScalarRoundTripProperties(t *testing.T) {
	t.Parallel()

	t.Run("Boolean", rapid.MakeCheck(
		roundTrip(rapid.Bool(), graphql.MarshalBoolean, graphql.UnmarshalBoolean, equalValue)))
	t.Run("String", rapid.MakeCheck(
		roundTrip(rapid.String(), graphql.MarshalString, graphql.UnmarshalString, equalValue)))
	t.Run("ID", rapid.MakeCheck(
		roundTrip(rapid.String(), graphql.MarshalID, graphql.UnmarshalID, equalValue)))

	t.Run("Int", rapid.MakeCheck(
		roundTrip(rapid.Int(), graphql.MarshalInt, graphql.UnmarshalInt, equalValue)))
	t.Run("Int8", rapid.MakeCheck(
		roundTrip(rapid.Int8(), graphql.MarshalInt8, graphql.UnmarshalInt8, equalValue)))
	t.Run("Int16", rapid.MakeCheck(
		roundTrip(rapid.Int16(), graphql.MarshalInt16, graphql.UnmarshalInt16, equalValue)))
	t.Run("Int32", rapid.MakeCheck(
		roundTrip(rapid.Int32(), graphql.MarshalInt32, graphql.UnmarshalInt32, equalValue)))
	t.Run("Int64", rapid.MakeCheck(
		roundTrip(rapid.Int64(), graphql.MarshalInt64, graphql.UnmarshalInt64, equalValue)))
	t.Run("IntID", rapid.MakeCheck(
		roundTrip(rapid.Int(), graphql.MarshalIntID, graphql.UnmarshalIntID, equalValue)))

	t.Run("Uint", rapid.MakeCheck(
		roundTrip(rapid.Uint(), graphql.MarshalUint, graphql.UnmarshalUint, equalValue)))
	t.Run("Uint8", rapid.MakeCheck(
		roundTrip(rapid.Uint8(), graphql.MarshalUint8, graphql.UnmarshalUint8, equalValue)))
	t.Run("Uint16", rapid.MakeCheck(
		roundTrip(rapid.Uint16(), graphql.MarshalUint16, graphql.UnmarshalUint16, equalValue)))
	t.Run("Uint32", rapid.MakeCheck(
		roundTrip(rapid.Uint32(), graphql.MarshalUint32, graphql.UnmarshalUint32, equalValue)))
	t.Run("Uint64", rapid.MakeCheck(
		roundTrip(rapid.Uint64(), graphql.MarshalUint64, graphql.UnmarshalUint64, equalValue)))
	t.Run("UintID", rapid.MakeCheck(
		roundTrip(rapid.Uint(), graphql.MarshalUintID, graphql.UnmarshalUintID, equalValue)))

	t.Run("Float", rapid.MakeCheck(
		roundTrip(finiteFloat(), graphql.MarshalFloat, graphql.UnmarshalFloat, equalValue)))
	t.Run("FloatContext", rapid.MakeCheck(
		roundTrip(finiteFloat(),
			func(f float64) graphql.Marshaler {
				return graphql.WrapContextMarshaler(
					context.Background(),
					graphql.MarshalFloatContext(f),
				)
			},
			func(v any) (float64, error) {
				return graphql.UnmarshalFloatContext(context.Background(), v)
			},
			equalValue)))

	t.Run("Time", rapid.MakeCheck(
		roundTrip(utcTime(), graphql.MarshalTime, graphql.UnmarshalTime, equalInstant)))
	t.Run("Date", rapid.MakeCheck(
		roundTrip(utcDate(), graphql.MarshalDate, graphql.UnmarshalDate, equalInstant)))
	t.Run("Duration", rapid.MakeCheck(
		roundTrip(durationNanos(),
			func(i int64) graphql.Marshaler { return graphql.MarshalDuration(time.Duration(i)) },
			func(v any) (int64, error) {
				d, err := graphql.UnmarshalDuration(v)
				return int64(d), err
			},
			equalValue)))

	t.Run("UUID", rapid.MakeCheck(
		roundTrip(uuidValue(), graphql.MarshalUUID, graphql.UnmarshalUUID, equalValue)))

	t.Run("Any", rapid.MakeCheck(
		roundTrip(jsonValue(2), graphql.MarshalAny, graphql.UnmarshalAny, equalValue)))
	t.Run("Map", rapid.MakeCheck(
		roundTrip(
			rapid.MapOfN(rapid.String(), jsonValue(2), 0, 4),
			graphql.MarshalMap, graphql.UnmarshalMap, equalValue)))
}

// Upload has no round-trip property and is excluded above deliberately.
// graphql.MarshalUpload copies the file's bytes to the writer rather than encoding
// JSON, and graphql.UnmarshalUpload only type-asserts an Upload that the multipart
// transport built. The two are not inverses of each other and were never meant
// to be, so a property asserting they are would be asserting a false thing.
func TestUploadHasNoRoundTripProperty(t *testing.T) {
	t.Parallel()

	_, err := graphql.UnmarshalUpload("a string that came off the wire")
	require.Error(t, err,
		"graphql.UnmarshalUpload takes an Upload the transport built, not a marshalled form")
}

// The minimum duration does not survive a round trip, and the failure is in
// github.com/sosodev/duration rather than in gqlgen: negating math.MinInt64
// overflows, so the formatter emits both a leading sign and a negative seconds
// component, producing an ISO-8601 duration its own parser rejects.
//
// This test will start failing if that is ever fixed upstream, which is the
// intent: durationNanos excludes the value, and the exclusion should not
// outlive the bug.
func TestMarshalDurationFailsAtMinInt64(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	graphql.MarshalDuration(math.MinInt64).MarshalGQL(&buf)

	var decoded any
	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	dec.UseNumber()
	require.NoError(t, dec.Decode(&decoded), "the marshalled form is still valid JSON")

	_, err := graphql.UnmarshalDuration(decoded)
	require.Error(t, err,
		"the minimum duration marshals to %s, which does not parse back", buf.String())
}

// graphql.MarshalFloat formats with %g, which renders a non-finite float as +Inf, -Inf
// or NaN. None of those is valid JSON, so the value a resolver returns cannot
// be parsed by any client. This records the behaviour rather than asserting it
// is correct: the property test above excludes non-finite floats because of it.
func TestMarshalFloatEmitsInvalidJSONForNonFiniteValues(t *testing.T) {
	t.Parallel()

	for name, f := range map[string]float64{
		"positive infinity": math.Inf(1),
		"negative infinity": math.Inf(-1),
		"not a number":      math.NaN(),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			graphql.MarshalFloat(f).MarshalGQL(&buf)

			var decoded any
			dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
			dec.UseNumber()
			require.Error(t, dec.Decode(&decoded),
				"%s marshals to %q, which a client cannot parse", name, buf.String())
		})
	}
}
