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

func Validate(s *schema.Schema, q *query.Document) (errs []*errors.QueryError) {
	for _, op := range q.Operations {
		for _, v := range op.Vars {
			if v.Default != nil {
				t, err := common.ResolveType(v.Type, s.Resolve)
				if err != nil {
					errs = append(errs, errors.ErrorfWithLoc(v.Default.Loc, "TODO"))
					continue
				}

				if nn, ok := t.(*common.NonNull); ok {
					errs = append(errs, errors.ErrorfWithLoc(v.Default.Loc, "Variable %q of type %q is required and will not use the default value. Perhaps you meant to use type %q.", "$"+v.Name, t, nn.OfType))
				}

				if ok, reason := validateValue(v.Default.Value, t); !ok {
					errs = append(errs, errors.ErrorfWithLoc(v.Default.Loc, "Variable %q of type %q has invalid default value %s.\n%s", "$"+v.Name, t, stringify(v.Default.Value), reason))
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
	return
}

func validateSelectionSet(s *schema.Schema, selSet *query.SelectionSet, t common.Type) []*errors.QueryError {
	var errs []*errors.QueryError
	switch t := t.(type) {
	case *schema.Object:
		for _, sel := range selSet.Selections {
			errs = append(errs, validateSelection(s, sel, t.Fields)...)
		}
	case *schema.Interface:
		for _, sel := range selSet.Selections {
			errs = append(errs, validateSelection(s, sel, t.Fields)...)
		}
	}
	return errs
}

func validateSelection(s *schema.Schema, sel query.Selection, fields schema.FieldList) (errs []*errors.QueryError) {
	switch sel := sel.(type) {
	case *query.Field:
		errs = append(errs, validateDirectives(s, sel.Directives)...)
		f := fields.Get(sel.Name)
		if f == nil {
			// TODO
			return
		}
		if len(f.Args) != 0 { // seems like a bug in graphql-js tests
			for _, selArg := range sel.Arguments {
				arg := f.Args.Get(selArg.Name)
				value := selArg.Value
				if ok, reason := validateValue(value.Value, arg.Type); !ok {
					errs = append(errs, errors.ErrorfWithLoc(value.Loc, "Argument %q has invalid value %s.\n%s", arg.Name, stringify(value.Value), reason))
				}
			}
		}
		if sel.SelSet != nil {
			errs = append(errs, validateSelectionSet(s, sel.SelSet, f.Type)...)
		}

	case *query.Fragment:
	// errs = append(errs, validateDirectives(s, sel.Directives)...)
	// for _, sel := range sel.SelSet.Selections {
	// 	errs = append(errs, validateSelection(s, sel, fields)...)
	// }

	default:
		panic("unreachable")
	}
	return
}

func validateDirectives(s *schema.Schema, directives map[string]common.ArgumentList) (errs []*errors.QueryError) {
	for name, args := range directives {
		d, ok := s.Directives[name]
		if !ok {
			errs = append(errs, errors.Errorf("TODO"))
			continue
		}
		for _, arg := range d.Args {
			value := args.Get(arg.Name)
			if ok, reason := validateValue(value.Value, arg.Type); !ok {
				errs = append(errs, errors.ErrorfWithLoc(value.Loc, "Argument %q has invalid value %s.\n%s", arg.Name, stringify(value.Value), reason))
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
			if _, ok := v[f.Name]; !ok {
				if _, ok := f.Type.(*common.NonNull); ok && f.Default == nil {
					return false, fmt.Sprintf("In field %q: Expected %q, found null.", f.Name, f.Type)
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
