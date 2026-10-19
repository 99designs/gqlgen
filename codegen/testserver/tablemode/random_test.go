package tablemode_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/functions"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/table"
)

// TestRandomRequests sends operations built at random from the schema to both
// executors and requires the same responses. The operations follow the schema, so
// that most of them reach the resolvers, and some carry wrong or missing values to
// reach the error paths. A failure names the seed, which reproduces the request.
func TestRandomRequests(t *testing.T) {
	n := 3000
	if testing.Short() {
		n = 300
	}
	for seed := range uint64(n) {
		compareRandomRequest(t, seed)
	}
}

// FuzzRandomRequests is TestRandomRequests driven by go test -fuzz.
func FuzzRandomRequests(f *testing.F) {
	for seed := range uint64(10) {
		f.Add(seed)
	}
	f.Fuzz(compareRandomRequest)
}

func compareRandomRequest(t *testing.T, seed uint64) {
	t.Helper()
	// The subtest carries the seed, so that a failure anywhere under it names the seed.
	t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) {
		functionsSchema := functions.NewSchema()
		tableSchema := table.NewSchema()

		g := &generator{
			schema: tableSchema.Schema(),
			rand:   rand.New(rand.NewPCG(seed, seed)),
			vars:   map[string]any{},
		}
		req := g.request()
		want := run(t, functionsSchema, req)
		got := run(t, tableSchema, req)
		if want != got {
			t.Fatalf("query: %s\nvariables: %v\nfunctions: %s\ntable:     %s",
				req.query, req.variables, want, got)
		}
	})
}

// generator builds a random operation from a schema.
type generator struct {
	schema  *ast.Schema
	rand    *rand.Rand
	vars    map[string]any
	varDefs []string
	// fragments are the named fragments the operation spreads.
	fragments []string
	aliases   int
	labels    int
	// deferred says whether a fragment on the value of the current field is deferred.
	deferred bool
	// noVars says that values may not come from variables: inside an argument that is a
	// variable as a whole, as the value of a variable cannot hold another one.
	noVars bool
}

func (g *generator) chance(p float64) bool { return g.rand.Float64() < p }

func pick[T any](g *generator, items []T) T { return items[g.rand.IntN(len(items))] }

func (g *generator) request() request {
	op, root, directive := "query", g.schema.Query, "@logQuery"
	switch {
	case g.chance(0.1):
		op, root, directive = "subscription", g.schema.Subscription, "@logSubscription"
	case g.chance(0.15):
		op, root, directive = "mutation", g.schema.Mutation, "@logMutation"
	}
	var sel string
	if op == "subscription" {
		// A subscription selects exactly one stream.
		sel = "{ " + g.field(g.fields(root)[0], 0) + " }"
	} else {
		sel = g.selection(root, 0)
	}
	var b strings.Builder
	b.WriteString(op)
	var name string
	if g.chance(0.2) {
		name = "Op" + strconv.Itoa(g.rand.IntN(3))
		b.WriteString(" " + name)
	}
	if len(g.varDefs) > 0 {
		b.WriteString("(" + strings.Join(g.varDefs, ", ") + ")")
	}
	if g.chance(0.2) {
		b.WriteString(" " + directive)
	}
	b.WriteString(" " + sel)
	for _, f := range g.fragments {
		b.WriteString("\n" + f)
	}
	// A document with several operations names the one to run.
	if name != "" && g.chance(0.3) {
		b.WriteString("\nquery Other { __typename }")
	}
	return request{query: b.String(), variables: g.vars, operationName: name, quiet: true}
}

// variable declares a variable of type t that holds value, and returns its use. Some
// variables are declared with the literal as their default and not given a value, and
// some nullable ones are declared but not given a value, so that the request sees an
// absent variable.
func (g *generator) variable(t *ast.Type, literal string, value any) string {
	name := "v" + strconv.Itoa(len(g.varDefs))
	def := "$" + name + ": " + t.String()
	switch {
	case literal != "null" && g.chance(0.1):
		def += " = " + literal
	case !t.NonNull && g.chance(0.05):
	default:
		g.vars[name] = value
	}
	g.varDefs = append(g.varDefs, def)
	return "$" + name
}

// boolVar declares a Boolean! variable with a random value and returns its use.
func (g *generator) boolVar() string {
	v := g.chance(0.5)
	return g.variable(ast.NonNullNamedType("Boolean", nil), strconv.FormatBool(v), v)
}

// spreadDirectives returns the directives of a fragment spread: none, mostly.
func (g *generator) spreadDirectives() string {
	switch r := g.rand.Float64(); {
	case r < 0.04:
		return " @skip(if: true)"
	case r < 0.08:
		return " @include(if: " + g.boolVar() + ")"
	}
	return ""
}

// spread returns the spread of a fragment on def with the selection body, inline or
// named, with the directives dirs.
func (g *generator) spread(def *ast.Definition, body, dirs string) string {
	if g.chance(0.2) {
		name := "f" + strconv.Itoa(len(g.fragments)+1)
		g.fragments = append(g.fragments, "fragment "+name+" on "+def.Name+" "+body)
		return "..." + name + dirs
	}
	return "... on " + def.Name + dirs + " " + body
}

// selection returns a selection set for a value of the type def.
func (g *generator) selection(def *ast.Definition, depth int) string {
	var parts []string
	switch def.Kind {
	case ast.Object:
		fields := g.fields(def)
		count := 1 + g.rand.IntN(min(4, len(fields)))
		for _, f := range fields[:count] {
			if part := g.field(f, depth); part != "" {
				parts = append(parts, part)
			}
		}
		if len(parts) == 0 || g.chance(0.1) {
			parts = append(parts, "__typename")
		}
		// Some fields are selected through a fragment on the object type itself.
		if g.chance(0.1) {
			parts[0] = g.spread(def, "{ "+parts[0]+" }", g.spreadDirectives())
		}
	case ast.Interface, ast.Union:
		parts = append(parts, "__typename")
		if def.Kind == ast.Interface && g.chance(0.5) {
			parts = append(parts, g.field(def.Fields.ForName("id"), depth))
		}
		for _, impl := range g.schema.GetPossibleTypes(def) {
			if !g.chance(0.6) {
				continue
			}
			var dirs string
			// At most one fragment on a value is deferred: the deferred fragments of one
			// object share whether one of their fields failed, so with several the
			// response depends on which finishes first. Labels are unique, as the spec
			// requires. A fragment with @defer(if: false) is not deferred.
			switch {
			case depth > 0 && g.chance(0.05):
				dirs = " @defer(if: false)"
			case depth > 0 && !g.deferred && g.chance(0.15):
				g.deferred = true
				g.labels++
				dirs = fmt.Sprintf(" @defer(label: %q)", "d"+strconv.Itoa(g.labels))
				if g.chance(0.2) {
					dirs = " @defer"
				}
			}
			dirs += g.spreadDirectives()
			parts = append(parts, g.spread(impl, g.selection(impl, depth+1), dirs))
		}
	}
	return "{ " + strings.Join(parts, " ") + " }"
}

// fields returns the fields of def that can be selected, in random order. The plugin of
// the generator removes pluginRemoved, on which both modes panic without recovering.
func (g *generator) fields(def *ast.Definition) ast.FieldList {
	var fields ast.FieldList
	for _, f := range def.Fields {
		if !strings.HasPrefix(f.Name, "__") && f.Name != "pluginRemoved" {
			fields = append(fields, f)
		}
	}
	g.rand.Shuffle(len(fields), func(i, j int) { fields[i], fields[j] = fields[j], fields[i] })
	return fields
}

// field returns the selection of f, or "" when f has a composite type and the
// selection is already too deep.
func (g *generator) field(f *ast.FieldDefinition, depth int) string {
	def := g.schema.Types[f.Type.Name()]
	composite := def.Kind == ast.Object || def.Kind == ast.Interface || def.Kind == ast.Union
	if composite && depth >= 3 {
		return ""
	}
	var b strings.Builder
	if g.chance(0.3) {
		g.aliases++
		fmt.Fprintf(&b, "a%d: ", g.aliases)
	}
	b.WriteString(f.Name)
	var args []string
	for _, a := range f.Arguments {
		if a.Type.NonNull && a.DefaultValue == nil || g.chance(0.6) {
			args = append(args, a.Name+": "+g.argument(a.Type))
		}
	}
	if len(args) > 0 {
		b.WriteString("(" + strings.Join(args, ", ") + ")")
	}
	switch r := g.rand.Float64(); {
	case r < 0.05:
		b.WriteString(" @skip(if: true)")
	case r < 0.1:
		b.WriteString(" @include(if: false)")
	case r < 0.13:
		b.WriteString(" @skip(if: " + g.boolVar() + ")")
	case r < 0.16:
		b.WriteString(" @include(if: " + g.boolVar() + ")")
	case r < 0.18:
		b.WriteString(" @include(if: true)")
	case r < 0.28:
		b.WriteString(` @trace(label: "t")`)
	case r < 0.3:
		label := g.variable(ast.NamedType("String", nil), `"v"`, "v")
		b.WriteString(" @trace(label: " + label + ")")
	}
	if composite {
		deferred := g.deferred
		g.deferred = false
		b.WriteString(" " + g.selection(def, depth+1))
		g.deferred = deferred
	}
	return b.String()
}

// argument returns an argument value of type t: mostly a literal, sometimes a
// variable.
func (g *generator) argument(t *ast.Type) string {
	if g.chance(0.25) {
		g.noVars = true
		literal, value := g.value(t, 0)
		g.noVars = false
		return g.variable(t, literal, value)
	}
	literal, _ := g.value(t, 0)
	return literal
}

// ids are the IDs the generator uses: those of the test data, and ones that match
// nothing.
var ids = []string{"1000", "1001", "1002", "1003", "2000", "2001", "3000", "3001", "9999", "x"}

// value returns a random value of type t as a GraphQL literal and as the JSON value of
// a variable. A small share of values is invalid for t.
func (g *generator) value(t *ast.Type, depth int) (string, any) {
	if !t.NonNull && g.chance(0.15) {
		return "null", nil
	}
	if g.chance(0.03) {
		i := g.rand.IntN(4)
		return []string{`"wrong"`, "17", "{}", "[]"}[i], []any{"wrong", 17, map[string]any{}, []any{}}[i]
	}
	// A value inside a list or an input may come from a variable of its type.
	if !g.noVars && depth > 0 && g.chance(0.1) {
		g.noVars = true
		literal, value := g.value(t, depth)
		g.noVars = false
		return g.variable(t, literal, value), value
	}
	if t.Elem != nil {
		// Lists hold up to four elements, past the two that a bug in the handling of the
		// elements after the first ones would need.
		n := g.rand.IntN(5)
		if g.chance(0.15) {
			// A single value where a list is expected is coerced to a list of one.
			return g.value(t.Elem, depth+1)
		}
		literals, values := make([]string, n), make([]any, n)
		for i := range n {
			if t.Elem.NonNull && g.chance(0.05) {
				// A null where the element must not be null.
				literals[i], values[i] = "null", nil
				continue
			}
			literals[i], values[i] = g.value(t.Elem, depth+1)
		}
		return "[" + strings.Join(literals, ", ") + "]", values
	}
	def := g.schema.Types[t.NamedType]
	switch def.Kind {
	case ast.Enum:
		v := pick(g, def.EnumValues).Name
		return v, v
	case ast.InputObject:
		return g.input(def, depth)
	}
	switch def.Name {
	case "Int":
		if g.chance(0.05) {
			// Values an Int does not hold: a fraction, a number as a string, and
			// integers beyond 32 and 64 bits.
			i := g.rand.IntN(4)
			return []string{"1.5", `"7"`, "2147483648", "9223372036854775808"}[i],
				[]any{1.5, "7", int64(2147483648), json.Number("9223372036854775808")}[i]
		}
		v := pick(g, []int{0, 1, 2, 4, -1, 1000, 2147483647, -2147483648})
		return strconv.Itoa(v), v
	case "Float":
		v := pick(g, []float64{0, 1.5, -2.25, 172, 1e308, -1e-308, 1e21})
		return strconv.FormatFloat(v, 'f', -1, 64), v
	case "Boolean":
		v := g.chance(0.5)
		return strconv.FormatBool(v), v
	case "ID":
		v := pick(g, ids)
		return strconv.Quote(v), v
	case "Time":
		v := pick(g, []string{
			"2020-01-01T00:00:00Z", "2026-05-06T07:08:09Z", "yesterday",
			"2026-02-30T00:00:00Z", "2026-05-06T07:08:09.123456789+09:00", "",
		})
		return strconv.Quote(v), v
	case "Money":
		v := pick(g, []string{"1.05", "0", "-3", "12.345", "lots", "1e3", "-0", "１０"})
		return strconv.Quote(v), v
	case "Map":
		return `{ a: 1, b: "two" }`, map[string]any{"a": 1, "b": "two"}
	}
	v := pick(g, []string{
		"", "a", "LUKE", "Falcon", "X", "日本語", "a rather long commentary text",
		`with "quotes" and \ backslash`, "line\nbreak\ttab", "control \x01 char", "emoji 😀",
	})
	return quote(v), v
}

// quote writes s as a GraphQL string literal: control characters are written as
// \uXXXX, as Go's \xXX is not an escape of GraphQL.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20:
			fmt.Fprintf(&b, `\u%04X`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// input returns a random value of the input object def.
func (g *generator) input(def *ast.Definition, depth int) (string, any) {
	var literals []string
	values := map[string]any{}
	for _, f := range def.Fields {
		required := f.Type.NonNull && f.DefaultValue == nil
		if !required && (depth > 1 || !g.chance(0.6)) {
			continue
		}
		literal, value := g.value(f.Type, depth+1)
		literals = append(literals, f.Name+": "+literal)
		values[f.Name] = value
	}
	slices.Sort(literals)
	return "{" + strings.Join(literals, ", ") + "}", values
}
