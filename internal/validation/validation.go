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

func addErr(errs *[]*errors.QueryError, loc errors.Location, rule string, format string, a ...interface{}) {
	*errs = append(*errs, &errors.QueryError{
		Message:   fmt.Sprintf(format, a...),
		Locations: []errors.Location{loc},
		Rule:      rule,
	})
}

func Validate(s *schema.Schema, q *query.Document) (errs []*errors.QueryError) {
	for _, op := range q.Operations {
		for _, v := range op.Vars {
			if v.Default != nil {
				t, err := common.ResolveType(v.Type, s.Resolve)
				if err != nil {
					continue
				}

				if nn, ok := t.(*common.NonNull); ok {
					addErr(&errs, v.Default.Loc, "DefaultValuesOfCorrectType", "Variable %q of type %q is required and will not use the default value. Perhaps you meant to use type %q.", "$"+v.Name.Name, t, nn.OfType)
				}

				if ok, reason := validateValue(v.Default.Value, t); !ok {
					addErr(&errs, v.Default.Loc, "DefaultValuesOfCorrectType", "Variable %q of type %q has invalid default value %s.\n%s", "$"+v.Name.Name, t, stringify(v.Default.Value), reason)
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
		errs = append(errs, validateSelectionSet(s, op.SelSet, entryPoint)...)
	}

	for _, frag := range q.Fragments {
		errs = append(errs, validateDirectives(s, frag.Directives)...)
		t, ok := s.Types[frag.On.Name]
		if !ok {
			continue
		}
		if !canBeFragment(t) {
			addErr(&errs, frag.On.Loc, "FragmentsOnCompositeTypes", "Fragment %q cannot condition on non composite type %q.", frag.Name, t)
			continue
		}
		errs = append(errs, validateSelectionSet(s, frag.SelSet, t)...)
	}

	return
}

func validateSelectionSet(s *schema.Schema, selSet *query.SelectionSet, t common.Type) (errs []*errors.QueryError) {
	for _, sel := range selSet.Selections {
		errs = append(errs, validateSelection(s, sel, t)...)
	}
	return
}

func validateSelection(s *schema.Schema, sel query.Selection, t common.Type) (errs []*errors.QueryError) {
	switch sel := sel.(type) {
	case *query.Field:
		errs = append(errs, validateDirectives(s, sel.Directives)...)
		if sel.Name == "__schema" || sel.Name == "__type" || sel.Name == "__typename" {
			return
		}

		t = unwrapType(t)
		f := fields(t).Get(sel.Name)
		if f == nil && t != nil {
			suggestion := makeSuggestion("Did you mean", fields(t).Names(), sel.Name)
			addErr(&errs, sel.Loc, "FieldsOnCorrectType", "Cannot query field %q on type %q.%s", sel.Name, t, suggestion)
		}

		if f != nil {
			for _, selArg := range sel.Arguments {
				arg := f.Args.Get(selArg.Name.Name)
				if arg == nil {
					addErr(&errs, selArg.Name.Loc, "KnownArgumentNames", "Unknown argument %q on field %q of type %q.", selArg.Name.Name, sel.Name, t)
					continue
				}
				value := selArg.Value
				if ok, reason := validateValue(value.Value, arg.Type); !ok {
					addErr(&errs, value.Loc, "ArgumentsOfCorrectType", "Argument %q has invalid value %s.\n%s", arg.Name.Name, stringify(value.Value), reason)
				}
			}
		}

		var ft common.Type
		if f != nil {
			ft = f.Type
		}
		if sel.SelSet != nil {
			errs = append(errs, validateSelectionSet(s, sel.SelSet, ft)...)
		}

	case *query.InlineFragment:
		errs = append(errs, validateDirectives(s, sel.Directives)...)
		if sel.On.Name != "" {
			t = s.Types[sel.On.Name]
		}
		if !canBeFragment(t) {
			addErr(&errs, sel.On.Loc, "FragmentsOnCompositeTypes", "Fragment cannot condition on non composite type %q.", t)
			return
		}
		errs = append(errs, validateSelectionSet(s, sel.SelSet, t)...)

	case *query.FragmentSpread:
		// TODO

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

func validateDirectives(s *schema.Schema, directives map[string]common.ArgumentList) (errs []*errors.QueryError) {
	for name, args := range directives {
		d, ok := s.Directives[name]
		if !ok {
			continue
		}
		for _, arg := range args {
			iv := d.Args.Get(arg.Name.Name)
			if iv == nil {
				addErr(&errs, arg.Name.Loc, "KnownArgumentNames", "Unknown argument %q on directive %q.", arg.Name.Name, "@"+name)
				continue
			}
			if ok, reason := validateValue(arg.Value.Value, iv.Type); !ok {
				addErr(&errs, arg.Value.Loc, "ArgumentsOfCorrectType", "Argument %q has invalid value %s.\n%s", arg.Name.Name, stringify(arg.Value.Value), reason)
			}
		}
	}
	return
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
