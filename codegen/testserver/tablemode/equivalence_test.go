package tablemode_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	aliasesfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/aliases/functions"
	aliasestable "github.com/99designs/gqlgen/codegen/testserver/tablemode/aliases/table"
	aliasesfsfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/aliasesfs/functions"
	aliasesfstable "github.com/99designs/gqlgen/codegen/testserver/tablemode/aliasesfs/table"
	aliasesfs2functions "github.com/99designs/gqlgen/codegen/testserver/tablemode/aliasesfs2/functions"
	aliasesfs2table "github.com/99designs/gqlgen/codegen/testserver/tablemode/aliasesfs2/table"
	batchfsfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/batchfs/functions"
	batchfstable "github.com/99designs/gqlgen/codegen/testserver/tablemode/batchfs/table"
	batchglobalfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/batchglobal/functions"
	batchglobaltable "github.com/99designs/gqlgen/codegen/testserver/tablemode/batchglobal/table"
	batchvaluesfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/batchvalues/functions"
	batchvaluestable "github.com/99designs/gqlgen/codegen/testserver/tablemode/batchvalues/table"
	caseargsfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/caseargs/functions"
	caseargstable "github.com/99designs/gqlgen/codegen/testserver/tablemode/caseargs/table"
	defaultsfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/defaults/functions"
	defaultstable "github.com/99designs/gqlgen/codegen/testserver/tablemode/defaults/table"
	federationfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/federation/functions"
	federationtable "github.com/99designs/gqlgen/codegen/testserver/tablemode/federation/table"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/functions"
	introspectedfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/introspected/functions"
	introspectedtable "github.com/99designs/gqlgen/codegen/testserver/tablemode/introspected/table"
	linkedfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/linked/functions"
	linkedtable "github.com/99designs/gqlgen/codegen/testserver/tablemode/linked/table"
	noqueryfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/noquery/functions"
	noquerytable "github.com/99designs/gqlgen/codegen/testserver/tablemode/noquery/table"
	querytypefunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/querytype/functions"
	querytypetable "github.com/99designs/gqlgen/codegen/testserver/tablemode/querytype/table"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/resolvers"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/table"
	withnullfunctions "github.com/99designs/gqlgen/codegen/testserver/tablemode/withnull/functions"
	withnulltable "github.com/99designs/gqlgen/codegen/testserver/tablemode/withnull/table"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/introspection"
)

type request struct {
	name      string
	query     string
	variables map[string]any
	// operationName selects the operation of a document that has several.
	operationName string
	// contains lists strings the response must contain, so that two executors that
	// fail the same way cannot pass by matching each other.
	contains []string
	// complexityLimit is the limit of the complexity extension, which otherwise has no
	// limit and only reports the complexity of every request.
	complexityLimit int
	// noIntrospection leaves out the introspection extension, as servers that disable
	// introspection do.
	noIntrospection bool
	// quiet recovers panics without printing their stack, for requests that are sent
	// in bulk.
	quiet bool
	// withoutFieldContext names fields whose resolvers the field middleware passes a
	// context without a field context, as middleware that starts a context of its own
	// does.
	withoutFieldContext string
}

var requests = []request{
	{
		name: "characters",
		query: `{
			hero { __typename id name appearsIn friends { __typename name ... on Human { height(unit: FOOT) } ... on Droid { primaryFunction serial } } }
			luke: human(id: "1000") { id name height mass starships { id name } secret nickname tags wallet friends { name } }
			ship: starship(id: "3000") { name length(unit: FOOT) history }
			han: human(id: "1002") { name nickname height(unit: METER) }
			vader: human(id: "1001") { name starships { name } }
			droid(id: "2000") { id name primaryFunction serial appearsIn friends { name } }
		}`,
		contains: []string{`"LUKE"`, `"Solo"`, `"12.34"`, `"Protocol"`, `starships are classified`},
	},
	{
		name:     "non-null error propagates to the nullable parent",
		query:    `{ droid(id: "2001") { name primaryFunction } other: droid(id: "2000") { name } }`,
		contains: []string{`droid has no function`, `"C-3PO"`},
	},
	{
		name:     "missing droid",
		query:    `{ droid(id: "9999") { name } }`,
		contains: []string{`no droid 9999`},
	},
	{
		name:     "union",
		query:    `{ search(text: "a") { __typename ... on Human { name } ... on Droid { name serial } ... on Starship { id name history } } }`,
		contains: []string{`"Falcon"`, `"HAN"`},
	},
	{
		name:     "non-null list element error nulls the whole response",
		query:    `{ search(text: "X") { ... on Starship { name length } } }`,
		contains: []string{`unknown length`},
	},
	{
		name:     "interfaces",
		query:    `{ node(id: "2000") { id __typename ... on Droid { name } } human: node(id: "1000") { id ... on Character { name } } missing: node(id: "x") { id } }`,
		contains: []string{`"C-3PO"`},
	},
	{
		name:      "variables and custom scalars",
		query:     `query($e: Episode!, $since: Time) { reviews(episode: $e, since: $since) { stars commentary time tags episode } }`,
		variables: map[string]any{"e": "JEDI", "since": "2020-01-01T00:00:00Z"},
		contains:  []string{`"2026-01-02T03:04:05Z"`},
	},
	{
		name: "input with defaults, nesting and directives",
		query: `{ echoReview(input: {
			stars: 4, commentary: "fine", time: "2026-05-06T07:08:09Z", note: "dropped",
			nested: { numbers: [1, 2], deeper: { value: "deep" } }, wallet: "1.05"
		}) { stars commentary time tags episode } }`,
		contains: []string{
			`"fresh"`,
			`"default"`,
			`"deep"`,
			`"wallet:105"`,
			`"validated"`,
			`"JEDI"`,
		},
	},
	{
		name:  "input from variables",
		query: `query($in: ReviewInput!) { echoReview(input: $in) { stars commentary tags episode } }`,
		variables: map[string]any{
			"in": map[string]any{"stars": 1, "tags": []any{"x"}, "episode": "EMPIRE"},
		},
		contains: []string{`"EMPIRE"`, `"validated"`},
	},
	{
		name:     "input field directive error",
		query:    `{ echoReview(input: { stars: 1, commentary: "this commentary is far too long" }) { stars } }`,
		contains: []string{`out of range`},
	},
	{
		name:     "input object directive error",
		query:    `{ echoReview(input: { stars: 6 }) { stars } }`,
		contains: []string{`at most 5 stars`},
	},
	{
		name:      "invalid custom scalar in input",
		query:     `query($in: ReviewInput!) { echoReview(input: $in) { stars } }`,
		variables: map[string]any{"in": map[string]any{"stars": 1, "wallet": "abc"}},
		contains:  []string{`invalid money`},
	},
	{
		name:     "list of inputs",
		query:    `{ a: echoReviews(inputs: [{ stars: 1 }, { stars: 2, tags: [] }]) { stars tags } b: echoReviews(inputs: null) { stars } c: echoReviews(inputs: []) { stars } }`,
		contains: []string{`"validated"`},
	},
	{
		name:     "map input",
		query:    `{ a: echoMap(input: { size: 3, role: USER }) b: echoMap(input: null) c: echoMap }`,
		contains: []string{`unnamed`, `USER`},
	},
	{
		name:     "input field set by a resolver",
		query:    `{ a: echoResolved(input: { label: "l", shout: "hey" }) b: echoResolved(input: { label: "l" }) }`,
		contains: []string{`"l/HEY"`},
	},
	{
		name:     "input field resolver error",
		query:    `{ echoResolved(input: { label: "l", shout: "" }) }`,
		contains: []string{`shout must not be empty`},
	},
	{
		name:     "omittable input fields",
		query:    `{ a: echoOmittable(input: { value: null }) b: echoOmittable(input: { count: 3 }) c: echoOmittable(input: {}) }`,
		contains: []string{`"nil,unset"`, `"unset,3"`, `"unset,unset"`},
	},
	{
		name:     "argument directive returning nil",
		query:    `{ echoNullified(value: "x") }`,
		contains: []string{`"nil"`},
	},
	{
		name:     "argument directive error",
		query:    `{ search(text: "") { __typename } }`,
		contains: []string{`out of range`},
	},
	{
		name: "type that marshals itself, bound to a converted scalar",
		query: `{ human(id: "1000") { code } a: echoReview(input: { stars: 1, code: "#x" }) { code }
			b: echoReview(input: { stars: 1 }) { code } }`,
		contains: []string{`"#luke"`, `"#x"`},
	},
	{
		name:     "invalid value of a type that marshals itself",
		query:    `{ echoReview(input: { stars: 1, code: "" }) { code } }`,
		contains: []string{`code must be a non-empty string`},
	},
	{
		name:     "scalars and enums",
		query:    `{ money(amount: "2.50") roles }`,
		contains: []string{`"5.00"`, `"GUEST"`},
	},
	{
		name:     "argument defaults",
		query:    `{ a: withDefaults b: withDefaults(a: 5, b: "y", c: [JEDI, NEWHOPE], d: { value: "w" }) c: withDefaults(a: null, c: null) }`,
		contains: []string{`"1|x|[EMPIRE]|v|[1 2]"`},
	},
	{
		name:  "null propagation",
		query: `{ nullChain { required optional child { required optional } other: child { optional requiredChild { required optional } } } }`,
		contains: []string{
			`"grandchild"`,
			`cannot return null for non-null field NullChain.required (String!)`,
		},
	},
	{
		name:     "errors and panics",
		query:    `{ failingOptional panicking summary { label count extra query { roles summary { label } } } }`,
		contains: []string{`optional failure`, `internal system error`, `"summary"`},
	},
	{
		name:     "non-null root error nulls data",
		query:    `{ failing summary { label } }`,
		contains: []string{`failing on purpose`},
	},
	{
		name:     "field and object directives",
		query:    `{ secretNumber human(id: "1000") { name secret } droid(id: "2000") { name } }`,
		contains: []string{`role ADMIN required`, `"LUKE"`, `"C-3PO"`},
	},
	{
		name:     "query directives",
		query:    `query @logQuery { human(id: "1002") { name @trace(label: "t") nickname @trace(label: "n") } summary { label @trace } }`,
		contains: []string{`"HAN [t]"`, `"Solo [n]"`},
	},
	{
		name:     "mutation",
		query:    `mutation @logMutation { createReview(episode: EMPIRE, review: { stars: 3, commentary: "ok" }) { stars commentary episode tags } setRole(role: GUEST) }`,
		contains: []string{`"EMPIRE"`, `"GUEST"`},
	},
	{
		name:     "defer",
		query:    `{ human(id: "1000") { name ... @defer(label: "ships") { starships { name } } ... @defer { friends { name } } } }`,
		contains: []string{`"ships"`, `"Falcon"`, `"hasNext":true`},
	},
	{
		name:     "type introspection",
		query:    `{ __type(name: "Human") { name kind fields { name args { name defaultValue } type { kind name ofType { kind name } } } interfaces { name } } }`,
		contains: []string{`"Character"`, `"METER"`},
	},
	{
		name:     "full introspection",
		query:    introspection.Query,
		contains: []string{`"ReviewInput"`, `"validReview"`},
	},
	{
		name:            "complexity uses the arguments",
		query:           `{ a: human(id: "1000") { height } b: human(id: "1002") { height(unit: FOOT) } }`,
		complexityLimit: 50,
		contains:        []string{`exceeds the limit of 50`},
	},
	{
		name:            "complexity under the limit",
		query:           `{ a: human(id: "1000") { height } }`,
		complexityLimit: 50,
		contains:        []string{`"height":1.72`},
	},
	{
		name: "directives of the types of fields and input fields, and Float defaults",
		query: `{ a: tag(input: { label: "x", mood: HAPPY })
			b: tag(input: { label: "y", ratio: 0.25 }, eps: 0.5) }`,
		contains: []string{`directive tagged`, `"y"`},
	},
	{
		name:            "schema introspection when introspection is disabled",
		query:           `{ __schema { queryType { name } } roles }`,
		noIntrospection: true,
		contains:        []string{`introspection disabled`, `"ADMIN"`},
	},
	{
		name:     "subscription events with field errors and nulls",
		query:    `subscription { ticks(count: 4, failAt: 2, nullAt: 3) { n label fail } }`,
		contains: []string{`tick 2 fails`, `"TICK 1"`},
	},
	{
		name:     "subscription event that is null in a non-null position",
		query:    `subscription { requiredTicks(count: 3, nullAt: 2) { n } }`,
		contains: []string{`"n":1`, `"n":3`},
	},
	{
		name:     "subscription of lists with an operation directive",
		query:    `subscription @logSubscription { tickBatches(count: 2) { n label } }`,
		contains: []string{`"TICK 2"`, `logSubscription`},
	},
	{
		name:     "subscription that fails",
		query:    `subscription { failingStream }`,
		contains: []string{`stream fails`},
	},
	{
		name:     "subscription field directive error",
		query:    `subscription { secretTicks }`,
		contains: []string{`role ADMIN required`},
	},
	{
		name:     "subscription of several fields",
		query:    `subscription { a: ticks(count: 1) { n } b: secretTicks }`,
		contains: []string{`only one top level field`},
	},
	{
		name: "marshalers and directives that fail or panic",
		query: `{ a: fragile(value: "x") b: fragile(value: "boom") c: fragiles(values: ["y", "boom"])
			d: special(kind: "nan") e: special(kind: "inf") f: special(kind: "1.5") exploding }`,
		quiet:    true,
		contains: []string{`internal system error`, `"x"`},
	},
	{
		name:     "non-null marshalers that fail or panic",
		query:    `{ a: fragileRequired(value: "boom") summary { label } }`,
		quiet:    true,
		contains: []string{`internal system error`},
	},
	{
		name:     "non-null Float that cannot be marshaled",
		query:    `{ specialRequired(kind: "-inf") summary { label } }`,
		contains: []string{`specialRequired`},
	},
	{
		name: "inputs decoded through the index of input unmarshalers",
		query: `{ a: decodeInputs(raw: { stars: 4, commentary: "fine", nested: { numbers: [1, 2] },
			label: "l", value: "v", count: 2, size: 3 })
			b: decodeInputs(raw: { stars: 9 }) c: decodeInputs(raw: {}) }`,
		contains: []string{`ReviewInput=`, `TagInput=`},
	},
	{
		name: "errors in deferred fragments",
		query: `{ droid(id: "2001") { name ... @defer(label: "f") { primaryFunction } }
			human(id: "1001") { name ... @defer { starships { name } } } }`,
		contains: []string{`droid has no function`},
	},
	{
		name:     "argument directive that changes the raw arguments",
		query:    `{ trimmed(text: "  hi  ") }`,
		contains: []string{`directive trimArg`},
	},
	{
		name:     "null for a directive argument with a default",
		query:    `{ nulledRatio }`,
		contains: []string{`null ratio`},
	},
	{
		name:  "null for non-null directive arguments with defaults",
		query: `{ nulledNonNull }`,
		contains: []string{
			`directive nulled names=[]string(nil) data=map[string]interface {}(nil)`,
		},
	},
	{
		name:  "directive arguments with the names of the objects of directives",
		query: `{ probed(x: "a", input: {v: "b"}) }`,
		contains: []string{
			`directive probe obj=map[string]interface {}{input: map[string]interface {}{v: string("b")}, x: string("a")} rawArgs=&string("r") asMap=(*string)(nil) it=(*string)(nil) nil=&string("n") true=&bool(false) on=&bool(true)`,
			`directive probe obj=map[string]interface {}{v: string("b")} rawArgs=(*string)(nil) asMap=&string("m") it=&string("i")`,
		},
	},
	{
		name:     "an object bound to a named map type that its first model can hold",
		query:    `{ bagHolder { bag { name } } }`,
		contains: []string{`"bag":{"name":"bag"}`},
	},
	{
		name:                "a field with arguments that its Go field does not take, without a field context",
		query:               `{ human(id: "1000") { massIn(unit: FOOT) } }`,
		withoutFieldContext: "massIn",
		contains:            []string{`"massIn":77.5`},
	},
	{
		name:                "a method of introspection that takes an argument, without a field context",
		query:               `{ __type(name: "Human") { fields { name args { name } } } }`,
		withoutFieldContext: "args",
		contains:            []string{`"fields":null`},
	},
	{
		name:     "arguments of a method in another order than the schema",
		query:    `{ human(id: "1000") { greet(a: "x", b: "y") } ok: human(id: "1000") { greet(a: "1", b: "2") } }`,
		contains: []string{`invalid money`},
	},
	{
		name:     "directive without an implementation",
		query:    `{ missingField missingArg(x: "v") }`,
		contains: []string{`not implemented`},
	},
	{
		name:     "list field bound to a named slice that marshals itself",
		query:    `{ luke: human(id: "1000") { words } vader: human(id: "1001") { words } }`,
		contains: []string{`use the force`},
	},
	{
		name:     "list input field bound to a named slice that unmarshals itself",
		query:    `{ tag(input: { label: "l", words: ["a b", "c"] }) none: tag(input: { label: "l", words: null }) }`,
		contains: []string{`Words`},
	},
	{
		name:     "input field and input object directives without an implementation",
		query:    `{ a: missingInput(input: { value: "v" }) b: missingInput(input: {}) }`,
		contains: []string{`directive validReview`, `directive length`},
	},
	{
		name:     "null for the arguments with a default of directives on an argument, an input field, an input object and a scalar",
		query:    `{ noted(x: "a", input: { v: "b", s: "c" }) }`,
		contains: []string{`directive note text=`},
	},
	{
		name:     "argument directives of a method that takes the arguments in another order",
		query:    `{ human(id: "1000") { greetOrder(a: "A", b: "B") } }`,
		contains: []string{`directive order seen=`},
	},
	{
		name: "directives that return another type than an argument or input field takes",
		query: `{ retyped(blob: "x") a: retypedInput(input: { extra: { k: 1 } })
			b: retypedInput(input: { blob: "y" }) retypedField }`,
		contains: []string{`unexpected type string`},
	},
	{
		name: "lists bound to a named slice of self-marshaling elements, a named map and a pointer",
		query: `{ luke: human(id: "1000") { tagList counts wordsPtr }
			vader: human(id: "1001") { counts wordsPtr } }`,
		contains: []string{`jedi+pilot`},
	},
	{
		name:     "non-null list bound to a named slice of self-marshaling elements that is nil",
		query:    `{ human(id: "1001") { tagList } }`,
		contains: []string{`errors`},
	},
	{
		name: "list inputs bound to a named slice of self-unmarshaling elements, a named map and a pointer",
		query: `{ shapes(input: { tagList: ["a", "b"], counts: [1, 2], wordsPtr: ["x y"] })
			none: shapes(input: { tagList: null, counts: null, wordsPtr: null }) }`,
		contains: []string{`unmarshaled a`},
	},
	{
		name: "nil non-null lists of lists bound to the same type that marshals itself",
		query: `{ luke: human(id: "1000") { matrixA matrixB }
			a: human(id: "1001") { matrixA } b: human(id: "1001") { matrixB } }`,
		contains: []string{`[[Int!]!]!`},
	},
	{
		name: "lists of more than two elements, and a null after the second element",
		query: `{ fragiles(values: ["a", "b", "c", "d"])
			echoReviews(inputs: [{ stars: 1 }, { stars: 2 }, { stars: 3 }, { stars: 4 }]) { stars }
			fleet { name } }`,
		contains: []string{`"d"`, `"stars":4`, `"fleet":null`},
	},
	{
		name:     "resolver error like that of a directive without an implementation, inside a directive",
		query:    `{ stubbed }`,
		contains: []string{`stub:`},
	},
	{
		name:     "field that a plugin made non-null",
		query:    `{ pluginRequired }`,
		contains: []string{`errors`},
	},
	{
		name:     "field that a plugin removed",
		query:    `{ pluginRemoved }`,
		contains: []string{`unknown field`},
	},
	{
		name:     "subscription whose resolver panics",
		query:    `subscription { panickingStream }`,
		contains: []string{`errors`},
	},
	{
		name:     "directives that a plugin put on a field and an argument",
		query:    `{ pluginGuarded pluginTrimmed(text: "  hi  ") }`,
		contains: []string{`role ADMIN required`, `directive trimArg`},
	},
	{
		name:     "typename on the root",
		query:    `{ __typename }`,
		contains: []string{`"Query"`},
	},
	{
		name: "batch resolvers",
		query: `{ fleets { id crew miscount again: crew }
			ship: starship(id: "3000") { crew rating rating10: rating(scale: 10) } }`,
		contains: []string{`"crew":3`, `returned 3 results for 2 parents`, `"rating10":3.437`},
	},
	{
		name: "batch resolver of a field that returns the query type",
		query: `{ fleets { query { __typename } }
			ship: starship(id: "3000") { query { __typename ship: starship(id: "3001") { id } } } }`,
		contains: []string{`"__typename":"Query"`, `"id":"3001"`},
	},
	{
		name: "batch resolver with an error for one parent",
		query: `{ luke: human(id: "1000") { starships { id rating(scale: 10) } }
			ship: starship(id: "3001") { rating } }`,
		contains: []string{`X-Wing has no rating`},
	},
	{
		name: "batch resolvers of the implementations of an interface and a union",
		query: `{ search(text: "-") { __typename ... on Droid { name lastSeen } ... on Starship { id crew } }
			c: search(text: "C") { ... on Droid { lastSeen } }
			luke: human(id: "1000") { friends { name ... on Droid { lastSeen } } } }`,
		contains: []string{`R2-D2 is hiding`, `"C-3PO on Tatooine"`},
	},
	{
		name:                "batch resolver without a field context",
		query:               `{ fleets { crew } }`,
		withoutFieldContext: "crew",
		contains:            []string{`"crew":3`},
	},
	{
		name:     "batch resolvers in a deferred fragment",
		query:    `{ fleets { id ... @defer { crew miscount } } }`,
		contains: []string{`returned 3 results for 2 parents`},
	},
}

func TestTableModeMatchesFunctionsMode(t *testing.T) {
	functionsSchema := functions.NewSchema()
	tableSchema := table.NewSchema()

	for _, req := range requests {
		t.Run(req.name, func(t *testing.T) {
			want := runOrPanic(t, functionsSchema, req)
			got := runOrPanic(t, tableSchema, req)
			require.Equal(t, want, got)
			for _, s := range req.contains {
				require.Contains(t, want, s)
			}
		})
	}
}

// TestDefaultsPerRequest sends a request twice whose input fields and directive arguments
// take Map defaults, which the unmarshaler of Map passes on as they are. run changes the
// values a request received once it is over, so a mode that handed every request the
// default of the schema itself would show the change in the second request.
func TestDefaultsPerRequest(t *testing.T) {
	functionsSchema := defaultsfunctions.NewSchema()
	tableSchema := defaultstable.NewSchema()
	req := request{query: `{ echo(input: {}, asMap: {}) }`}
	for i := range 2 {
		want := run(t, functionsSchema, req)
		got := run(t, tableSchema, req)
		require.Equal(t, want, got, "request %d", i)
		require.NotContains(t, want, "_mutated", "request %d", i)
	}
}

// TestOtherSchemas compares the modes on schemas and configs that the schema of this test
// server cannot have: a schema without a query type, to which gqlgen adds an empty one
// while it generates the code, a schema with a type called Query that is not its query
// type, which gqlgen replaces with an empty one, directives on the types of introspection,
// and call_argument_directives_with_null.
// entities is the query of the entities of representations, with the fields of every
// entity type of the federation schema.
const entities = `query($r: [_Any!]!) { _entities(representations: $r) { __typename
	... on Hello { name greeting }
	... on World { foo bar hello { name greeting } }
	... on Planet { name size }
	... on MultiHello { name }
	... on Product { id display tax discount }
	... on Item { id label }
} }`

// federationRequests are the requests of a router to the federation schema.
var federationRequests = []request{
	{name: "service", query: `{ _service { sdl } }`, contains: []string{`type Hello @key`}},
	{
		name:  "an entity with one key",
		query: entities,
		variables: map[string]any{"r": []any{
			map[string]any{"__typename": "Hello", "name": "a"},
			map[string]any{"__typename": "Hello", "name": "fail"},
			map[string]any{"__typename": "Hello", "name": "nil"},
			map[string]any{"__typename": "Hello", "name": 1},
		}},
		contains: []string{`"HELLO A"`, `no hello called fail`},
	},
	{
		name:  "an entity with a nested key and a second key",
		query: entities,
		variables: map[string]any{"r": []any{
			map[string]any{"__typename": "World", "hello": map[string]any{"name": "h"}, "foo": "f"},
			map[string]any{"__typename": "World", "bar": 3},
			map[string]any{"__typename": "World", "foo": "x"},
			map[string]any{"__typename": "World", "hello": "h", "foo": "y"},
		}},
		contains: []string{`"foo3"`, `missing Key Field`},
	},
	{
		name:  "a field that an entity requires",
		query: entities,
		variables: map[string]any{"r": []any{
			map[string]any{"__typename": "Planet", "name": "p", "diameter": 12},
			map[string]any{"__typename": "Planet", "name": "q", "diameter": "big"},
			map[string]any{"__typename": "Planet", "name": "r"},
		}},
		contains: []string{`"size":120`},
	},
	{
		name:  "entities resolved together, with errors of their own",
		query: entities,
		variables: map[string]any{"r": []any{
			map[string]any{"__typename": "MultiHello", "name": "a"},
			map[string]any{"__typename": "MultiHello", "name": "fail"},
			map[string]any{"__typename": "MultiHello", "name": "b"},
		}},
		contains: []string{`no multi hello 1`},
	},
	{
		name:  "fields required before the call, and computed by field and batch resolvers",
		query: entities,
		variables: map[string]any{"r": []any{
			map[string]any{"__typename": "Product", "id": "1", "category": "c", "price": 100},
			map[string]any{"__typename": "Product", "id": "2", "category": "d", "price": 55},
			map[string]any{"__typename": "Product", "id": "3", "category": "e"},
			map[string]any{"__typename": "Product", "id": "4", "category": "f", "price": "cheap"},
		}},
		contains: []string{`"display":"1 in c"`, `"tax":10`, `price cheap is not a number`},
	},
	{
		name:  "fields computed by a batch resolver",
		query: entities,
		variables: map[string]any{"r": []any{
			map[string]any{"__typename": "Product", "id": "1", "category": "c", "price": 100},
			map[string]any{"__typename": "Hello", "name": "a"},
			map[string]any{"__typename": "Product", "id": "2", "category": "d"},
		}},
		contains: []string{`"discount":20`},
	},
	{
		name:  "an entity that is an interface, and its implementation",
		query: entities,
		variables: map[string]any{"r": []any{
			map[string]any{"__typename": "Node", "id": "n1"},
			map[string]any{"__typename": "Item", "id": "i1"},
			map[string]any{"__typename": "Item", "id": "none"},
		}},
		contains: []string{`"node n1"`, `no item none`},
	},
	{
		name:  "entities of several types, of an unknown type and without a type",
		query: entities,
		variables: map[string]any{"r": []any{
			map[string]any{"__typename": "Hello", "name": "a"},
			map[string]any{"__typename": "Planet", "name": "p", "diameter": 1},
			map[string]any{"__typename": "Unknown", "id": "u"},
			map[string]any{"name": "x"},
			map[string]any{"__typename": "MultiHello", "name": "m"},
		}},
		contains: []string{`unknown type`, `__typename must be an existing string`},
	},
}

func TestOtherSchemas(t *testing.T) {
	for _, tc := range []struct {
		name             string
		functions, table func() graphql.ExecutableSchema
		requests         []request
	}{
		{
			name:      "without a query type",
			functions: noqueryfunctions.NewSchema,
			table:     noquerytable.NewSchema,
			requests: []request{
				{name: "mutation", query: `mutation { ping }`, contains: []string{`ping`}},
				{name: "query", query: `{ __typename }`, contains: []string{`does not support operation type`}},
			},
		},
		{
			name:      "a type called Query that is not the query type",
			functions: querytypefunctions.NewSchema,
			table:     querytypetable.NewSchema,
			requests: []request{
				{name: "mutation", query: `mutation { ping }`, contains: []string{`ping`}},
				{name: "query", query: `{ __typename }`, contains: []string{`does not support operation type`}},
			},
		},
		{
			name: "without a query type, with a schema in the Config that has one",
			functions: func() graphql.ExecutableSchema {
				return noqueryfunctions.NewSchemaFor(withQuery(t))
			},
			table: func() graphql.ExecutableSchema { return noquerytable.NewSchemaFor(withQuery(t)) },
			requests: []request{
				{name: "introspection", query: `{ __schema { queryType { name } } }`, contains: []string{`Query`}},
			},
		},
		{
			name:      "a method whose parameters take the same argument",
			functions: caseargsfunctions.NewSchema,
			table:     caseargstable.NewSchema,
			requests: []request{
				{name: "both parameters", query: `{ caseArgs { f(id: "x") } }`, contains: []string{`x,x`}},
			},
		},
		{
			name:      "inputs bound to the types of two packages with the same name",
			functions: aliasesfunctions.NewSchema,
			table:     aliasestable.NewSchema,
			requests: []request{
				{name: "INPUT_OBJECT directives that return another type", query: `{ a(in: { x: 1 }) b(in: { y: 2 }) }`, contains: []string{`should be`}},
			},
		},
		{
			name:      "a schema linked to the changed schema, with a non-root type called Subscription and directives on String",
			functions: linkedfunctions.NewSchema,
			table:     linkedtable.NewSchema,
			requests: []request{
				{name: "fields of the ordinary type", query: `{ plan(id: "1") { name fail } }`, contains: []string{`plan fails`}},
				{name: "a field of the type that fails", query: `{ plan(id: "x") { name } }`, contains: []string{`no plan`}},
				{name: "String fields", query: `{ name pluginRequired }`, contains: []string{`name`}},
				{name: "mutation", query: `mutation { ping }`, contains: []string{`ping`}},
			},
		},
		{
			name:      "directives on types of introspection",
			functions: introspectedfunctions.NewSchema,
			table:     introspectedtable.NewSchema,
			requests: []request{
				{name: "schema", query: `{ __schema { queryType { name } } }`, contains: []string{`directive seen`}},
				{name: "type", query: `{ __type(name: "Query") { name fields { type { name } } } }`, contains: []string{`directive seen`}},
			},
		},
		{
			name:      "federation",
			functions: federationfunctions.NewSchema,
			table:     federationtable.NewSchema,
			requests:  federationRequests,
		},
		{
			name:      "call_argument_directives_with_null",
			functions: withnullfunctions.NewSchema,
			table:     withnulltable.NewSchema,
			requests: []request{
				{name: "absent argument that a directive sets", query: `{ posts }`, contains: []string{`u1`}},
				{name: "argument", query: `{ posts(authorID: "x") }`, contains: []string{`x`}},
			},
		},
		// The batch resolvers put the size of their batch into the values they return, and
		// record the parents of each call, so that a mode that does not collect the parents
		// of a list into one call differs from the functions mode.
		{
			name:      "batch resolvers from resolver.batch",
			functions: batchglobalfunctions.NewSchema,
			table:     batchglobaltable.NewSchema,
			requests: []request{
				{
					name:  "a batch resolver configured as a resolver and one forced by the model",
					query: `{ users { id name posts { id title author { id } } } }`,
					contains: []string{
						`batch User.posts [u1 u2 u3]`,
						`"post by Luke:p1/3"`,
						`batch Post.author [p1 p2]`,
						`"author:p3/1"`,
					},
				},
				{
					name:     "introspection, which resolver.batch leaves unbatched",
					query:    `{ __type(name: "User") { fields { name } } users { id } }`,
					contains: []string{`"posts"`},
				},
			},
		},
		{
			name:      "batch resolvers of the values of a slice, with omit_slice_element_pointers",
			functions: batchvaluesfunctions.NewSchema,
			table:     batchvaluestable.NewSchema,
			requests: []request{
				{
					name:     "parents of a value slice",
					query:    `{ users { id posts { id author { id } } } }`,
					contains: []string{`batch Post.author [p1 p2 p3]`, `"author:p1/3"`, `"author:p4/1"`},
				},
			},
		},
		{
			name:      "batch resolvers of implementations declared in another file, in the follow-schema layout",
			functions: batchfsfunctions.NewSchema,
			table:     batchfstable.NewSchema,
			requests: []request{
				{
					name:  "both implementations",
					query: `{ animals { id ... on Dog { batchProp } ... on Cat { batchProp } } }`,
					contains: []string{
						`batch Dog.batchProp [d1 d2 d3]`,
						`batch Cat.batchProp [c1 c2]`,
						`"dog:d2/3"`,
						`"cat:c2/2"`,
					},
				},
				{
					name:     "the implementation of the other file only",
					query:    `{ animals { __typename ... on Cat { batchProp } } }`,
					contains: []string{`"cat:c1/2"`},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var functionsSchema, tableSchema graphql.ExecutableSchema
			require.NotPanics(t, func() { functionsSchema = tc.functions() })
			require.NotPanics(t, func() { tableSchema = tc.table() })
			for _, req := range tc.requests {
				t.Run(req.name, func(t *testing.T) {
					want := run(t, functionsSchema, req)
					got := run(t, tableSchema, req)
					require.Equal(t, want, got)
					for _, s := range req.contains {
						require.Contains(t, want, s)
					}
				})
			}
		})
	}
}

// TestPluginCalls compares the marshalers and unmarshalers of the functions mode that the
// code of a plugin calls by name: table mode generates functions of those names, which
// must return what the functions mode returns. See the callers plugin of the gen command.
func TestPluginCalls(t *testing.T) {
	ctx := graphql.WithOperationContext(context.Background(), &graphql.OperationContext{})
	ctx = graphql.WithResponseContext(ctx, graphql.DefaultErrorPresenter, graphql.DefaultRecover)
	want := functions.PluginCalls(ctx, functions.NewSchema())
	got := table.PluginCalls(ctx, table.NewSchema())
	require.NotEmpty(t, want)
	require.Equal(t, want, got)
	// The raw values are valid, so that both modes cannot pass by failing alike.
	for _, res := range want {
		require.True(t, strings.HasSuffix(res, " err=<nil>"), res)
		require.NotContains(t, res, " null err=", res)
	}
}

// TestInputTypeNamesInErrors pins where the modes differ: the error of an INPUT_OBJECT
// directive that returns another type names the Go type of the input with the alias that
// the file of the executor gives its package, and the modes look the packages up in
// another order. Two packages with the same name may thus get each other's alias, as here
// in the follow-schema layout, where a file refers to both packages.
func TestInputTypeNamesInErrors(t *testing.T) {
	for _, tc := range []struct {
		name             string
		functions, table func() graphql.ExecutableSchema
		query            string
		// The errors of each mode, as the message ends and the path follows.
		functionsErrors, tableErrors []string
	}{
		{
			name:      "inputs bound to the types of two packages with the same name, in the follow-schema layout",
			functions: aliasesfsfunctions.NewSchema,
			table:     aliasesfstable.NewSchema,
			query:     `{ a(in: { x: 1 }) holder { b(in: { y: 2 }) } }`,
			functionsErrors: []string{
				`should be model1.Input","path":["a","in"]`,
				`should be model.Input","path":["holder","b","in"]`,
			},
			tableErrors: []string{
				`should be model.Input","path":["a","in"]`,
				`should be model1.Input","path":["holder","b","in"]`,
			},
		},
		{
			name:      "an input of a file that refers to the package of another only through a directive",
			functions: aliasesfs2functions.NewSchema,
			table:     aliasesfs2table.NewSchema,
			query:     `{ a(in: { x: 1 }) b(in: { y: 2 }) }`,
			functionsErrors: []string{
				`should be model1.Input","path":["a","in"]`,
				`should be model1.Input","path":["b","in"]`,
			},
			tableErrors: []string{
				`should be model.Input","path":["a","in"]`,
				`should be model1.Input","path":["b","in"]`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := request{name: tc.name, query: tc.query}
			functions := run(t, tc.functions(), req)
			table := run(t, tc.table(), req)
			for _, e := range tc.functionsErrors {
				require.Contains(t, functions, e)
			}
			for _, e := range tc.tableErrors {
				require.Contains(t, table, e)
			}
		})
	}
}

// TestSchemaChangedLater changes the schema that Schema returns, which the executable
// schemas of a generated package share, as code that hides an argument from introspection
// might, and compares the modes on an executable schema created afterwards: the functions
// mode runs what it was generated for whatever the schema says then.
func TestSchemaChangedLater(t *testing.T) {
	respond := func(newSchema func() graphql.ExecutableSchema) string {
		arg := newSchema().Schema().Query.Fields.ForName("posts").Arguments.ForName("authorID")
		dirs := arg.Directives
		arg.Directives = nil
		defer func() { arg.Directives = dirs }()
		return runOrPanic(t, newSchema(), request{query: `{ posts }`})
	}
	want := respond(withnullfunctions.NewSchema)
	require.Contains(t, want, "u1")
	require.Equal(t, want, respond(withnulltable.NewSchema))
}

// withQuery returns the schema of the noquery test server with a query type added.
func withQuery(t *testing.T) *ast.Schema {
	t.Helper()
	src, err := os.ReadFile("noquery/schema.graphql")
	require.NoError(t, err)
	s, err := gqlparser.LoadSchema(
		&ast.Source{Name: "schema.graphql", Input: string(src) + "type Query { x: Int }\n"},
	)
	require.NoError(t, err)
	return s
}

// TestCustomSchema checks that an executable schema given a schema of its own in the
// Config answers as the functions mode does, which validates requests against that schema
// and answers introspection from it, but resolves the fields as the code was generated:
// such a schema may leave out a field to hide it, or declare the arguments of a field in
// another order.
func TestCustomSchema(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(s *ast.Schema)
		query  string
	}{
		{
			name: "a field left out",
			change: func(s *ast.Schema) {
				q := s.Types["Query"]
				q.Fields = slices.DeleteFunc(q.Fields, func(f *ast.FieldDefinition) bool {
					return f.Name == "secretNumber"
				})
			},
			query: `{ roles summary { label } }`,
		},
		{
			name: "arguments in another order",
			change: func(s *ast.Schema) {
				slices.Reverse(s.Types["Query"].Fields.ForName("withDefaults").Arguments)
			},
			query: `{ a: withDefaults b: withDefaults(a: 5, b: "y", d: { value: "w" }) }`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := request{query: tc.query}
			want := run(t, schemaFor(t, functions.NewSchemaFor, tc.change), req)
			got := run(t, schemaFor(t, table.NewSchemaFor, tc.change), req)
			require.Equal(t, want, got)
		})
	}
}

// schemaFor returns the executable schema that newSchema makes with the schema of this
// test server, parsed afresh and changed by change, in its Config.
func schemaFor(
	t *testing.T,
	newSchema func(*ast.Schema) graphql.ExecutableSchema,
	change func(*ast.Schema),
) (es graphql.ExecutableSchema) {
	t.Helper()
	src, err := os.ReadFile("schema.graphql")
	require.NoError(t, err)
	s, err := gqlparser.LoadSchema(&ast.Source{Name: "schema.graphql", Input: string(src)})
	require.NoError(t, err)
	change(s)
	require.NotPanics(t, func() { es = newSchema(s) })
	return es
}

// TestRecordingTellsValuesApart checks that the recording run compares distinguishes
// the Go values that the responses do not, so that TestTableModeMatchesFunctionsMode
// and TestRandomRequests compare them too.
func TestRecordingTellsValuesApart(t *testing.T) {
	schema := functions.NewSchema()
	for _, tc := range []struct{ query, want string }{
		// An Omittable that is set to nil, one set to a value, and unset ones.
		{
			query: `{ echoOmittable(input: { value: null, count: 3 }) }`,
			want:  `Value: Omittable((*string)(nil)), Count: Omittable(&int(3))`,
		},
		{
			query: `{ echoOmittable(input: {}) }`,
			want:  `Value: Omittable(unset), Count: Omittable(unset)`,
		},
		// The nil of a directive as the resolver receives it, and what the directive got.
		{
			query: `{ echoNullified(value: "x") }`,
			want:  `args=map[string]interface {}{value: (*string)(nil)}`,
		},
		{
			query: `{ echoNullified(value: "x") }`,
			want:  `directive nullify obj=map[string]interface {}{value: string("x")} -> &string("x")`,
		},
		// A nil list against an empty one.
		{
			query: `{ a: echoReviews(inputs: null) { stars } }`,
			want:  `inputs: []*models.ReviewInput(nil)`,
		},
		{
			query: `{ a: echoReviews(inputs: []) { stars } }`,
			want:  `inputs: []*models.ReviewInput{}`,
		},
		// The object a field directive receives, and the context of a field.
		{
			query: `{ human(id: "1000") { secret } }`,
			want:  `directive auth obj=&models.Human{`,
		},
		{
			query: `{ human(id: "1000") { name } }`,
			want:  `field name path=human.name method=false resolver=false object=Human index=nil parent=true`,
		},
	} {
		got := run(t, schema, request{query: tc.query, quiet: true})
		require.Contains(t, got, tc.want, tc.query)
	}
}

// middlewareKey holds the name of the field in the context that the field middleware
// passes on, which the recover function records.
type middlewareKey struct{}

// runOrPanic is run, or the value of the panic that the executor does not recover, as
// for a field of a query that the code was not generated for.
func runOrPanic(t *testing.T, es graphql.ExecutableSchema, req request) (res string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			res = fmt.Sprintf("panic: %v", r)
		}
	}()
	return run(t, es, req)
}

// run executes req and returns every response, including deferred ones, as JSON,
// followed by the trace of what the resolvers, directives and middleware received.
func run(t *testing.T, es graphql.ExecutableSchema, req request) string {
	t.Helper()

	exec := executor.New(es)
	if !req.noIntrospection {
		exec.Use(extension.Introspection{})
	}
	// Every request reports its complexity, so that the modes are compared on it.
	limit := req.complexityLimit
	if limit == 0 {
		limit = math.MaxInt32
	}
	exec.Use(extension.FixedComplexityLimit(limit))
	tr := &resolvers.Recording{}
	exec.SetRecoverFunc(func(ctx context.Context, err any) error {
		// The context is the one the field middleware passed on, as the resolver had it.
		tr.Add("panic %s middleware=%v", resolvers.Describe(err), ctx.Value(middlewareKey{}))
		if req.quiet {
			return gqlerror.Errorf("internal system error")
		}
		return graphql.DefaultRecover(ctx, err)
	})
	exec.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
		// The error presenter gets the context of the field too.
		// The Go types of the error and of those it wraps tell the ways of wrapping it
		// apart, which errors.As and errors.Is see.
		var chain []string
		for e := err; e != nil; e = errors.Unwrap(e) {
			chain = append(chain, fmt.Sprintf("%T", e))
		}
		tr.Add("present %v middleware=%v types=%v", err, ctx.Value(middlewareKey{}), chain)
		return graphql.DefaultErrorPresenter(ctx, err)
	})
	exec.AroundOperations(func(
		ctx context.Context,
		next graphql.OperationHandler,
	) graphql.ResponseHandler {
		tr.Add("operation %s", graphql.GetOperationContext(ctx).Operation.Operation)
		if stats := extension.GetComplexityStats(ctx); stats != nil {
			tr.Add("complexity %d", stats.Complexity)
		}
		return next(ctx)
	})
	exec.AroundRootFields(func(ctx context.Context, next graphql.RootResolver) graphql.Marshaler {
		tr.Add("root %s", graphql.GetRootFieldContext(ctx).Field.Name)
		return next(ctx)
	})
	exec.AroundFields(func(ctx context.Context, next graphql.Resolver) (any, error) {
		fc := graphql.GetFieldContext(ctx)
		tr.Keep(fc.Args)
		rctx := ctx
		if fc.Field.Name == req.withoutFieldContext {
			rctx = graphql.WithResponseContext(
				graphql.WithOperationContext(
					context.Background(),
					graphql.GetOperationContext(ctx),
				),
				graphql.DefaultErrorPresenter,
				graphql.DefaultRecover,
			)
		}
		res, err := next(context.WithValue(rctx, middlewareKey{}, fc.Field.Name))
		index := "nil"
		if fc.Index != nil {
			index = strconv.Itoa(*fc.Index)
		}
		format := "field %s path=%s method=%t resolver=%t object=%s index=%s parent=%t " +
			"args=%s -> %s err=%v"
		tr.Add(
			format,
			fc.Field.Name,
			fc.Path(),
			fc.IsMethod,
			fc.IsResolver,
			fc.Object,
			index,
			fc.Parent != nil,
			resolvers.Describe(fc.Args),
			resolvers.Describe(res),
			err,
		)
		return res, err
	})

	// Decode the variables from JSON as the HTTP transports do, with numbers as
	// json.Number.
	var variables map[string]any
	if req.variables != nil {
		b, err := json.Marshal(req.variables)
		require.NoError(t, err)
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		require.NoError(t, d.Decode(&variables))
	}

	ctx := resolvers.WithRecording(graphql.StartOperationTrace(context.Background()), tr)
	params := &graphql.RawParams{
		Query:         req.query,
		Variables:     variables,
		OperationName: req.operationName,
	}
	opCtx, errs := exec.CreateOperationContext(ctx, params)
	if errs != nil {
		resp := exec.DispatchError(graphql.WithOperationContext(ctx, opCtx), errs)
		return marshal(t, []*graphql.Response{resp}, false) + "\n" + strings.Join(tr.Lines(), "\n")
	}

	handler, ctx := exec.DispatchOperation(ctx, opCtx)
	responses := collect(t, func() *graphql.Response { return handler(ctx) }, req.query)
	// The values this request received are changed once it is over, so that a later
	// request that is given the same values, such as a default shared between requests,
	// shows the change.
	tr.Mark()
	subscription := opCtx.Operation.Operation == ast.Subscription
	return marshal(t, responses, subscription) + "\n" + strings.Join(tr.Lines(), "\n")
}

// collectTimeout is how long the handler of a request of these tests may take to return
// all of its responses. One that takes longer does not finish, such as one that waits for
// a deferred response that never comes.
const collectTimeout = 30 * time.Second

// collect returns the responses that next returns until it returns nil. It fails the test,
// which the subtest names, such as by the seed of a random request, when next keeps
// returning responses or does not return within collectTimeout, rather than leaving the
// test binary to wait until go test times out. A panic of next is raised again in the
// goroutine of the test, where runOrPanic recovers it.
func collect(t *testing.T, next func() *graphql.Response, query string) []*graphql.Response {
	t.Helper()
	type result struct {
		responses []*graphql.Response
		tooMany   bool
		panicked  any
	}
	done := make(chan result, 1)
	go func() {
		var r result
		defer func() {
			r.panicked = recover()
			done <- r
		}()
		for resp := next(); resp != nil; resp = next() {
			// The executor writes the events of a subscription into one buffer, which a
			// transport sends before it asks for the next event, so the data is copied.
			resp.Data = bytes.Clone(resp.Data)
			r.responses = append(r.responses, resp)
			// A request of these tests defers a few fragments at most; many more
			// responses mean the executor does not finish.
			if len(r.responses) >= 200 {
				r.tooMany = true
				return
			}
		}
	}()
	timer := time.NewTimer(collectTimeout)
	defer timer.Stop()
	select {
	case r := <-done:
		if r.panicked != nil {
			panic(r.panicked)
		}
		require.False(t, r.tooMany, "the executor keeps returning responses\nquery: %s", query)
		return r.responses
	case <-timer.C:
		t.Fatalf("the executor returns no response within %v\nquery: %s", collectTimeout, query)
		return nil
	}
}

// marshal returns the responses as JSON in an order that does not depend on the order in
// which concurrent resolvers finished: errors are sorted, and deferred responses after the
// first one are sorted by label and path. The responses of a subscription are its events,
// which keep their order.
func marshal(t *testing.T, responses []*graphql.Response, subscription bool) string {
	t.Helper()

	for _, resp := range responses {
		// Sort by the whole error, as two errors may share a path and a message and
		// differ in their extensions.
		key := make(map[*gqlerror.Error]string, len(resp.Errors))
		for _, err := range resp.Errors {
			b, jsonErr := json.Marshal(err)
			require.NoError(t, jsonErr)
			key[err] = string(b)
		}
		sort.SliceStable(resp.Errors, func(i, j int) bool {
			return key[resp.Errors[i]] < key[resp.Errors[j]]
		})
	}
	if len(responses) > 1 && !subscription {
		// Every response but the last says that more follow.
		for i, resp := range responses {
			require.NotNil(t, resp.HasNext, "hasNext of response %d", i)
			require.Equal(t, i < len(responses)-1, *resp.HasNext, "hasNext of response %d", i)
		}
		rest := responses[1:]
		sort.SliceStable(rest, func(i, j int) bool {
			if rest[i].Label != rest[j].Label {
				return rest[i].Label < rest[j].Label
			}
			return rest[i].Path.String() < rest[j].Path.String()
		})
		// Which deferred response comes last depends on timing.
		for _, resp := range rest {
			resp.HasNext = nil
		}
	}

	b, err := json.Marshal(responses)
	require.NoError(t, err)
	return string(b)
}

func BenchmarkModes(b *testing.B) {
	query := `{
		hero { __typename id name appearsIn friends { __typename name ... on Human { height(unit: FOOT) } ... on Droid { primaryFunction } } }
		human(id: "1000") { id name height mass starships { id name } nickname tags wallet friends { name } }
		reviews(episode: JEDI) { stars commentary time tags episode }
		echoReview(input: { stars: 4, commentary: "fine", nested: { numbers: [1, 2] } }) { stars tags }
	}`
	for _, mode := range []struct {
		name   string
		schema graphql.ExecutableSchema
	}{
		{"functions", functions.NewSchema()},
		{"table", table.NewSchema()},
	} {
		b.Run(mode.name, func(b *testing.B) {
			exec := executor.New(mode.schema)
			b.ReportAllocs()
			for b.Loop() {
				ctx := graphql.StartOperationTrace(context.Background())
				opCtx, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{Query: query})
				if errs != nil {
					b.Fatal(errs)
				}
				handler, ctx := exec.DispatchOperation(ctx, opCtx)
				if resp := handler(ctx); len(resp.Errors) > 0 {
					b.Fatal(resp.Errors)
				}
			}
		})
	}
}
