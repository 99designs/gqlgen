package validation

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/lexer"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

type context struct {
	schema *schema.Schema
	doc    *query.Document
	errs   []*errors.QueryError
}

func (c *context) addErr(loc errors.Location, rule string, format string, a ...interface{}) {
	c.errs = append(c.errs, &errors.QueryError{
		Message:   fmt.Sprintf(format, a...),
		Locations: []errors.Location{loc},
		Rule:      rule,
	})
}

func Validate(s *schema.Schema, doc *query.Document) []*errors.QueryError {
	c := context{
		schema: s,
		doc:    doc,
	}

	for _, op := range doc.Operations {
		if op.Name == "" && len(doc.Operations) != 1 {
			c.addErr(op.Loc, "LoneAnonymousOperation", "This anonymous operation must be the only defined operation.")
		}

		c.validateDirectives(string(op.Type), op.Directives)

		for _, v := range op.Vars {
			t := c.resolveType(v.Type)
			if t != nil && v.Default != nil {
				if nn, ok := t.(*common.NonNull); ok {
					c.addErr(v.Default.Loc, "DefaultValuesOfCorrectType", "Variable %q of type %q is required and will not use the default value. Perhaps you meant to use type %q.", "$"+v.Name.Name, t, nn.OfType)
				}

				if ok, reason := validateValue(v.Default.Value, t); !ok {
					c.addErr(v.Default.Loc, "DefaultValuesOfCorrectType", "Variable %q of type %q has invalid default value %s.\n%s", "$"+v.Name.Name, t, stringify(v.Default.Value), reason)
				}
			}
		}

		var entryPoint common.Type
		switch op.Type {
		case query.Query:
			entryPoint = s.EntryPoints["query"]
		case query.Mutation:
			entryPoint = s.EntryPoints["mutation"]
		default:
			panic("unreachable")
		}
		c.validateSelectionSet(op.SelSet, entryPoint)
	}

	for _, frag := range doc.Fragments {
		c.validateDirectives("FRAGMENT_DEFINITION", frag.Directives)
		t := c.resolveType(&frag.On)
		// continue even if t is nil
		if !canBeFragment(t) {
			c.addErr(frag.On.Loc, "FragmentsOnCompositeTypes", "Fragment %q cannot condition on non composite type %q.", frag.Name, t)
			continue
		}
		c.validateSelectionSet(frag.SelSet, t)
	}

	sort.Slice(c.errs, func(i, j int) bool { return c.errs[i].Locations[0].Before(c.errs[j].Locations[0]) })
	return c.errs
}

func (c *context) validateSelectionSet(selSet *query.SelectionSet, t common.Type) {
	for _, sel := range selSet.Selections {
		c.validateSelection(sel, t)
	}
	return
}

func (c *context) validateSelection(sel query.Selection, t common.Type) {
	switch sel := sel.(type) {
	case *query.Field:
		c.validateDirectives("FIELD", sel.Directives)
		if sel.Name == "__schema" || sel.Name == "__type" || sel.Name == "__typename" {
			return
		}

		t = unwrapType(t)
		f := fields(t).Get(sel.Name)
		if f == nil && t != nil {
			suggestion := makeSuggestion("Did you mean", fields(t).Names(), sel.Name)
			c.addErr(sel.Loc, "FieldsOnCorrectType", "Cannot query field %q on type %q.%s", sel.Name, t, suggestion)
		}

		c.validateArgumentNames(sel.Arguments)
		if f != nil {
			for _, selArg := range sel.Arguments {
				arg := f.Args.Get(selArg.Name.Name)
				if arg == nil {
					c.addErr(selArg.Name.Loc, "KnownArgumentNames", "Unknown argument %q on field %q of type %q.", selArg.Name.Name, sel.Name, t)
					continue
				}
				value := selArg.Value
				if ok, reason := validateValue(value.Value, arg.Type); !ok {
					c.addErr(value.Loc, "ArgumentsOfCorrectType", "Argument %q has invalid value %s.\n%s", arg.Name.Name, stringify(value.Value), reason)
				}
			}
		}

		var ft common.Type
		if f != nil {
			ft = f.Type
		}
		if sel.SelSet != nil {
			c.validateSelectionSet(sel.SelSet, ft)
		}

	case *query.InlineFragment:
		c.validateDirectives("INLINE_FRAGMENT", sel.Directives)
		if sel.On.Name != "" {
			t = c.resolveType(&sel.On)
			// continue even if t is nil
		}
		if t != nil && !canBeFragment(t) {
			c.addErr(sel.On.Loc, "FragmentsOnCompositeTypes", "Fragment cannot condition on non composite type %q.", t)
			return
		}
		c.validateSelectionSet(sel.SelSet, t)

	case *query.FragmentSpread:
		c.validateDirectives("FRAGMENT_SPREAD", sel.Directives)
		if _, ok := c.doc.Fragments[sel.Name.Name]; !ok {
			c.addErr(sel.Name.Loc, "KnownFragmentNames", "Unknown fragment %q.", sel.Name.Name)
		}

	default:
		panic("unreachable")
	}
	return
}

func fields(t common.Type) schema.FieldList {
	switch t := t.(type) {
	case *schema.Object:
		return t.Fields
	case *schema.Interface:
		return t.Fields
	default:
		return nil
	}
}

func unwrapType(t common.Type) common.Type {
	switch t := t.(type) {
	case *common.List:
		return unwrapType(t.OfType)
	case *common.NonNull:
		return unwrapType(t.OfType)
	default:
		return t
	}
}

func (c *context) resolveType(t common.Type) common.Type {
	t2, err := common.ResolveType(t, c.schema.Resolve)
	if err != nil {
		c.errs = append(c.errs, err)
	}
	return t2
}

func (c *context) validateDirectives(loc string, directives map[string]*common.Directive) {
	for name, d := range directives {
		c.validateArgumentNames(d.Args)

		dd, ok := c.schema.Directives[name]
		if !ok {
			c.addErr(d.Name.Loc, "KnownDirectives", "Unknown directive %q.", name)
			continue
		}

		locOK := false
		for _, allowedLoc := range dd.Locs {
			if loc == allowedLoc {
				locOK = true
				break
			}
		}
		if !locOK {
			c.addErr(d.Name.Loc, "KnownDirectives", "Directive %q may not be used on %s.", name, loc)
		}

		for _, arg := range d.Args {
			iv := dd.Args.Get(arg.Name.Name)
			if iv == nil {
				c.addErr(arg.Name.Loc, "KnownArgumentNames", "Unknown argument %q on directive %q.", arg.Name.Name, "@"+name)
				continue
			}
			if ok, reason := validateValue(arg.Value.Value, iv.Type); !ok {
				c.addErr(arg.Value.Loc, "ArgumentsOfCorrectType", "Argument %q has invalid value %s.\n%s", arg.Name.Name, stringify(arg.Value.Value), reason)
			}
		}
	}
	return
}

func (c *context) validateArgumentNames(args common.ArgumentList) {
	seen := make(map[string]errors.Location)
	for _, arg := range args {
		if loc, ok := seen[arg.Name.Name]; ok {
			c.errs = append(c.errs, &errors.QueryError{
				Message:   fmt.Sprintf("There can be only one argument named %q.", arg.Name.Name),
				Locations: []errors.Location{loc, arg.Name.Loc},
				Rule:      "UniqueArgumentNames",
			})
			continue
		}
		seen[arg.Name.Name] = arg.Name.Loc
	}
}

func validateValue(v interface{}, t common.Type) (bool, string) {
	if nn, ok := t.(*common.NonNull); ok {
		if v == nil {
			return false, fmt.Sprintf("Expected %q, found null.", t)
		}
		t = nn.OfType
	}
	if v == nil {
		return true, ""
	}

	if l, ok := t.(*common.List); ok {
		if _, ok := v.([]interface{}); !ok {
			return validateValue(v, l.OfType)
		}
	}

	if _, ok := v.(lexer.Variable); ok {
		// TODO
		return true, ""
	}

	if v, ok := v.(*lexer.Literal); ok {
		if validateLiteral(v, t) {
			return true, ""
		}
	}

	switch t := t.(type) {
	case *common.List:
		v, ok := v.([]interface{})
		if !ok {
			return false, fmt.Sprintf("Expected %q, found not a list.", t)
		}
		for i, entry := range v {
			if ok, reason := validateValue(entry, t.OfType); !ok {
				return false, fmt.Sprintf("In element #%d: %s", i, reason)
			}
		}
		return true, ""
	case *schema.InputObject:
		v, ok := v.(map[string]interface{})
		if !ok {
			return false, fmt.Sprintf("Expected %q, found not an object.", t)
		}
		for name, entry := range v {
			f := t.Values.Get(name)
			if f == nil {
				return false, fmt.Sprintf("In field %q: Unknown field.", name)
			}
			if ok, reason := validateValue(entry, f.Type); !ok {
				return false, fmt.Sprintf("In field %q: %s", name, reason)
			}
		}
		for _, f := range t.Values {
			if _, ok := v[f.Name.Name]; !ok {
				if _, ok := f.Type.(*common.NonNull); ok && f.Default == nil {
					return false, fmt.Sprintf("In field %q: Expected %q, found null.", f.Name.Name, f.Type)
				}
			}
		}
		return true, ""
	}

	return false, fmt.Sprintf("Expected type %q, found %s.", t, stringify(v))
}

func validateLiteral(v *lexer.Literal, t common.Type) bool {
	switch t := t.(type) {
	case *schema.Scalar:
		switch t.Name {
		case "Int":
			if v.Type != scanner.Int {
				return false
			}
			f, err := strconv.ParseFloat(v.Text, 64)
			if err != nil {
				panic(err)
			}
			return f >= math.MinInt32 && f <= math.MaxInt32
		case "Float":
			return v.Type == scanner.Int || v.Type == scanner.Float
		case "String":
			return v.Type == scanner.String
		case "Boolean":
			return v.Type == scanner.Ident && (v.Text == "true" || v.Text == "false")
		case "ID":
			return v.Type == scanner.Int || v.Type == scanner.String
		}

	case *schema.Enum:
		if v.Type != scanner.Ident {
			return false
		}
		for _, option := range t.Values {
			if option.Name == v.Text {
				return true
			}
		}
		return false
	}

	return false
}

func canBeFragment(t common.Type) bool {
	switch t.(type) {
	case *schema.Object, *schema.Interface, *schema.Union:
		return true
	default:
		return false
	}
}

func stringify(v interface{}) string {
	switch v := v.(type) {
	case *lexer.Literal:
		return v.Text

	case []interface{}:
		entries := make([]string, len(v))
		for i, entry := range v {
			entries[i] = stringify(entry)
		}
		return "[" + strings.Join(entries, ", ") + "]"

	case map[string]interface{}:
		names := make([]string, 0, len(v))
		for name := range v {
			names = append(names, name)
		}
		sort.Strings(names)

		entries := make([]string, 0, len(names))
		for _, name := range names {
			entries = append(entries, name+": "+stringify(v[name]))
		}
		return "{" + strings.Join(entries, ", ") + "}"

	case nil:
		return "null"

	default:
		return fmt.Sprintf("(invalid type: %T)", v)
	}
}
