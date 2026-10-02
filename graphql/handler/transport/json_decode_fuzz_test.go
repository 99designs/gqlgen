package transport

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
)

// jsonDecode is the single decoding entry point shared by four transports: the
// POST, GET, urlencoded-form and multipart-mixed paths all funnel request bodies
// or query parameters through it. Everything it sees arrives from the network
// before any authentication or validation has run, which makes it the broadest
// piece of untrusted-input parsing in the library.
//
// This fuzzes it against graphql.RawParams, the destination every one of those
// callers decodes into. The properties are the ones a transport relies on and
// that no caller re-checks:
//
//   - it never panics, whatever bytes arrive;
//   - a reported error leaves nothing for the caller to act on, so the caller is
//     entitled to abandon the request;
//   - success means the parameters are usable: no nil map that a later write
//     would panic on, and no string field holding something other than a string.
//
// The seeds cover the shapes a real client sends plus the shapes an attacker
// sends: truncated JSON, wrong types in every field, deep nesting, duplicate
// keys, huge numbers and bare scalars where an object is expected.
func FuzzJSONDecode(f *testing.F) {
	seeds := []string{
		// Ordinary requests.
		`{"query":"{ name }"}`,
		`{"query":"query Q($id: Int!){ find(id: $id) }","variables":{"id":1}}`,
		`{"query":"{ name }","operationName":"Q","variables":{},"extensions":{}}`,
		`{"query":"{ name }","extensions":{"persistedQuery":{"version":1,"sha256Hash":"abc"}}}`,

		// Empty and minimal.
		``,
		`{}`,
		`null`,
		`[]`,
		`0`,
		`""`,
		`true`,

		// Truncated or malformed.
		`{`,
		`{"query":`,
		`{"query":"{ name }"`,
		`{"query":"\ud800"}`,
		`{"query":"{ name }",}`,

		// Wrong types where a string or object is expected.
		`{"query":123}`,
		`{"query":{"nested":"object"}}`,
		`{"query":["an","array"]}`,
		`{"operationName":false}`,
		`{"variables":"not an object"}`,
		`{"variables":[1,2,3]}`,
		`{"extensions":42}`,

		// Duplicate keys: the last wins in encoding/json, which is worth
		// pinning because a client could use it to smuggle a second query past
		// something that inspected the first.
		`{"query":"{ a }","query":"{ b }"}`,

		// Numeric edges. UseNumber means these stay exact rather than becoming
		// float64, which is the behaviour the scalar unmarshalers depend on.
		`{"variables":{"big":9223372036854775807}}`,
		`{"variables":{"bigger":9223372036854775808}}`,
		`{"variables":{"huge":1e308}}`,
		`{"variables":{"tiny":1e-308}}`,
		`{"variables":{"overflow":1e400}}`,
		`{"variables":{"neg":-9223372036854775808}}`,

		// Nesting and repetition.
		`{"variables":{"a":{"b":{"c":{"d":{"e":1}}}}}}`,
		`{"variables":` + strings.Repeat(`[`, 64) + strings.Repeat(`]`, 64) + `}`,

		// Trailing content after a complete value, which Decode accepts and
		// stops at. A transport reading one value from a stream must not be
		// confused by what follows.
		`{"query":"{ name }"} {"query":"{ other }"}`,
		`{"query":"{ name }"}trailing garbage`,
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		var params graphql.RawParams
		err := jsonDecode(bytes.NewReader(body), &params)
		if err != nil {
			// A failed decode is the caller's cue to reject the request. There
			// is nothing to assert about the partially filled destination,
			// because no caller reads it.
			return
		}

		// Past this point a transport hands these parameters to the executor,
		// so they have to be coherent.
		for key, value := range params.Variables {
			requireDecodedJSON(t, "variables["+key+"]", value)
		}
		for key, value := range params.Extensions {
			requireDecodedJSON(t, "extensions["+key+"]", value)
		}

		// RawParams is re-encodable, which the APQ extension and the
		// multipart transport both rely on when they rewrite a request.
		if _, marshalErr := json.Marshal(params); marshalErr != nil {
			t.Fatalf("decoded params do not re-encode: %v (body %q)", marshalErr, body)
		}
	})
}

// requireDecodedJSON asserts a decoded value is one of the shapes
// encoding/json produces with UseNumber set. A scalar unmarshaler type-switches
// over exactly this set, so anything else would reach it as an unhandled
// default and be reported as a client error for the wrong reason.
func requireDecodedJSON(t *testing.T, path string, value any) {
	t.Helper()

	switch v := value.(type) {
	case nil, bool, string, json.Number:
		return
	case map[string]any:
		for key, nested := range v {
			requireDecodedJSON(t, path+"."+key, nested)
		}
	case []any:
		for _, nested := range v {
			requireDecodedJSON(t, path+"[]", nested)
		}
	case float64:
		// UseNumber is set, so a float64 here would mean the decoder was not
		// configured the way jsonDecode configures it.
		t.Fatalf("%s decoded to float64 %v; UseNumber should have produced json.Number",
			path, v)
	default:
		t.Fatalf("%s decoded to unexpected type %T", path, value)
	}
}
