package validation

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

type context struct {
	schema *schema.Schema
	doc    *query.Document
	errs   []*errors.QueryError
}

func (c *context) addErr(loc errors.Location, rule string, format string, a ...interface{}) {
	c.addErrMultiLoc([]errors.Location{loc}, rule, format, a...)
}

func (c *context) addErrMultiLoc(locs []errors.Location, rule string, format string, a ...interface{}) {
	c.errs = append(c.errs, &errors.QueryError{
		Message:   fmt.Sprintf(format, a...),
		Locations: locs,
		Rule:      rule,
	})
}

func Validate(s *schema.Schema, doc *query.Document) []*errors.QueryError {
	c := context{
		schema: s,
		doc:    doc,
	}

	opNames := make(nameSet)
	fragUsed := make(map[*query.FragmentDecl]struct{})
	for _, op := range doc.Operations {
		if op.Name.Name == "" && len(doc.Operations) != 1 {
			c.addErr(op.Name.Loc, "LoneAnonymousOperation", "This anonymous operation must be the only defined operation.")
		}
		if op.Name.Name != "" {
			c.validateName(opNames, op.Name, "UniqueOperationNames", "operation")
		}

		c.validateDirectives(string(op.Type), op.Directives)

		varNames := make(nameSet)
		for _, v := range op.Vars {
			c.validateName(varNames, v.Name, "UniqueVariableNames", "variable")

			t := c.resolveType(v.Type)
			if !canBeInput(t) {
				c.addErr(v.TypeLoc, "VariablesAreInputTypes", "Variable %q cannot be non-input type %q.", "$"+v.Name.Name, t)
			}

			if t != nil && v.Default != nil {
				if nn, ok := t.(*common.NonNull); ok {
					c.addErr(v.Default.Loc, "DefaultValuesOfCorrectType", "Variable %q of type %q is required and will not use the default value. Perhaps you meant to use type %q.", "$"+v.Name.Name, t, nn.OfType)
				}

				if ok, reason := validateValue(v.Default.Value, t); !ok {
					c.addErr(v.Default.Loc, "DefaultValuesOfCorrectType", "Variable %q of type %q has invalid default value %s.\n%s", "$"+v.Name.Name, t, common.Stringify(v.Default.Value), reason)
				}
			}
		}

		var entryPoint common.Type
		switch op.Type {
		case query.Query:
			entryPoint = s.EntryPoints["query"]
		case query.Mutation:
			entryPoint = s.EntryPoints["mutation"]
		case query.Subscription:
			entryPoint = s.EntryPoints["subscription"]
		default:
			panic("unreachable")
		}
		c.shallowValidateSelectionSet(op.SelSet, entryPoint)
		c.markUsedFragments(op.SelSet, fragUsed)
	}

	fragNames := make(nameSet)
	fragVisited := make(map[*query.FragmentDecl]struct{})
	for _, frag := range doc.Fragments {
		c.validateName(fragNames, frag.Name, "UniqueFragmentNames", "fragment")
		c.validateDirectives("FRAGMENT_DEFINITION", frag.Directives)

		t := c.resolveType(&frag.On)
		// continue even if t is nil
		if t != nil && !canBeFragment(t) {
			c.addErr(frag.On.Loc, "FragmentsOnCompositeTypes", "Fragment %q cannot condition on non composite type %q.", frag.Name.Name, t)
			continue
		}

		c.shallowValidateSelectionSet(frag.SelSet, t)

		if _, ok := fragVisited[frag]; !ok {
			c.detectFragmentCycle(frag.SelSet, fragVisited, nil, map[string]int{frag.Name.Name: 0})
		}
	}

	for _, frag := range doc.Fragments {
		if _, ok := fragUsed[frag]; !ok {
			c.addErr(frag.Loc, "NoUnusedFragments", "Fragment %q is never used.", frag.Name.Name)
		}
	}

	return c.errs
}

func (c *context) shallowValidateSelectionSet(selSet *query.SelectionSet, t common.Type) {
	for _, sel := range selSet.Selections {
		c.shallowValidateSelection(sel, t)
	}
}

func (c *context) shallowValidateSelection(sel query.Selection, t common.Type) {
	switch sel := sel.(type) {
	case *query.Field:
		c.validateDirectives("FIELD", sel.Directives)

		t = unwrapType(t)
		fieldName := sel.Name.Name
		var f *schema.Field
		switch fieldName {
		case "__typename":
			f = &schema.Field{
				Name: "__typename",
				Type: c.schema.Types["String"],
			}
		case "__schema":
			f = &schema.Field{
				Name: "__schema",
				Type: c.schema.Types["__Schema"],
			}
		case "__type":
			f = &schema.Field{
				Name: "__type",
				Args: common.InputValueList{
					&common.InputValue{
						Name: common.Ident{Name: "name"},
						Type: &common.NonNull{OfType: c.schema.Types["String"]},
					},
				},
				Type: c.schema.Types["__Type"],
			}
		default:
			f = fields(t).Get(fieldName)
			if f == nil && t != nil {
				suggestion := makeSuggestion("Did you mean", fields(t).Names(), fieldName)
				c.addErr(sel.Alias.Loc, "FieldsOnCorrectType", "Cannot query field %q on type %q.%s", fieldName, t, suggestion)
			}
		}

		names := make(nameSet)
		for _, arg := range sel.Arguments {
			c.validateName(names, arg.Name, "UniqueArgumentNames", "argument")
		}

		if f != nil {
			c.validateArguments(sel.Arguments, f.Args, sel.Alias.Loc,
				func() string { return fmt.Sprintf("field %q of type %q", fieldName, t) },
				func() string { return fmt.Sprintf("Field %q", fieldName) },
			)
		}

		var ft common.Type
		if f != nil {
			ft = f.Type
			sf := hasSubfields(ft)
			if sf && sel.SelSet == nil {
				c.addErr(sel.Alias.Loc, "ScalarLeafs", "Field %q of type %q must have a selection of subfields. Did you mean \"%s { ... }\"?", fieldName, ft, fieldName)
			}
			if !sf && sel.SelSet != nil {
				c.addErr(sel.SelSet.Loc, "ScalarLeafs", "Field %q must not have a selection since type %q has no subfields.", fieldName, ft)
			}
		}
		if sel.SelSet != nil {
			c.shallowValidateSelectionSet(sel.SelSet, ft)
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
		c.shallowValidateSelectionSet(sel.SelSet, t)

	case *query.FragmentSpread:
		c.validateDirectives("FRAGMENT_SPREAD", sel.Directives)
		if frag := c.doc.Fragments.Get(sel.Name.Name); frag == nil {
			c.addErr(sel.Name.Loc, "KnownFragmentNames", "Unknown fragment %q.", sel.Name.Name)
			return
		}

	default:
		panic("unreachable")
	}
}

func (c *context) markUsedFragments(selSet *query.SelectionSet, fragUsed map[*query.FragmentDecl]struct{}) {
	for _, sel := range selSet.Selections {
		switch sel := sel.(type) {
		case *query.Field:
			if sel.SelSet != nil {
				c.markUsedFragments(sel.SelSet, fragUsed)
			}

		case *query.InlineFragment:
			c.markUsedFragments(sel.SelSet, fragUsed)

		case *query.FragmentSpread:
			frag := c.doc.Fragments.Get(sel.Name.Name)
			if frag == nil {
				return
			}

			if _, ok := fragUsed[frag]; ok {
				return
			}
			fragUsed[frag] = struct{}{}
			c.markUsedFragments(frag.SelSet, fragUsed)

		default:
			panic("unreachable")
		}
	}
}

func (c *context) detectFragmentCycle(selSet *query.SelectionSet, fragVisited map[*query.FragmentDecl]struct{}, spreadPath []*query.FragmentSpread, spreadPathIndex map[string]int) {
	for _, sel := range selSet.Selections {
		c.detectFragmentCycleSel(sel, fragVisited, spreadPath, spreadPathIndex)
	}
}

func (c *context) detectFragmentCycleSel(sel query.Selection, fragVisited map[*query.FragmentDecl]struct{}, spreadPath []*query.FragmentSpread, spreadPathIndex map[string]int) {
	switch sel := sel.(type) {
	case *query.Field:
		if sel.SelSet != nil {
			c.detectFragmentCycle(sel.SelSet, fragVisited, spreadPath, spreadPathIndex)
		}

	case *query.InlineFragment:
		c.detectFragmentCycle(sel.SelSet, fragVisited, spreadPath, spreadPathIndex)

	case *query.FragmentSpread:
		frag := c.doc.Fragments.Get(sel.Name.Name)
		if frag == nil {
			return
		}

		spreadPath = append(spreadPath, sel)
		if i, ok := spreadPathIndex[frag.Name.Name]; ok {
			cyclePath := spreadPath[i:]
			via := ""
			if len(cyclePath) > 1 {
				names := make([]string, len(cyclePath)-1)
				for i, frag := range cyclePath[:len(cyclePath)-1] {
					names[i] = frag.Name.Name
				}
				via = " via " + strings.Join(names, ", ")
			}

			locs := make([]errors.Location, len(cyclePath))
			for i, frag := range cyclePath {
				locs[i] = frag.Loc
			}
			c.addErrMultiLoc(locs, "NoFragmentCycles", "Cannot spread fragment %q within itself%s.", frag.Name.Name, via)
			return
		}

		if _, ok := fragVisited[frag]; ok {
			return
		}
		fragVisited[frag] = struct{}{}

		spreadPathIndex[frag.Name.Name] = len(spreadPath)
		c.detectFragmentCycle(frag.SelSet, fragVisited, spreadPath, spreadPathIndex)
		delete(spreadPathIndex, frag.Name.Name)

	default:
		panic("unreachable")
	}
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

func (c *context) validateDirectives(loc string, directives common.DirectiveList) {
	directiveNames := make(nameSet)
	for _, d := range directives {
		dirName := d.Name.Name
		c.validateNameCustomMsg(directiveNames, d.Name, "UniqueDirectivesPerLocation", func() string {
			return fmt.Sprintf("The directive %q can only be used once at this location.", dirName)
		})

		argNames := make(nameSet)
		for _, arg := range d.Args {
			c.validateName(argNames, arg.Name, "UniqueArgumentNames", "argument")
		}

		dd, ok := c.schema.Directives[dirName]
		if !ok {
			c.addErr(d.Name.Loc, "KnownDirectives", "Unknown directive %q.", dirName)
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
			c.addErr(d.Name.Loc, "KnownDirectives", "Directive %q may not be used on %s.", dirName, loc)
		}

		c.validateArguments(d.Args, dd.Args, d.Name.Loc,
			func() string { return fmt.Sprintf("directive %q", "@"+dirName) },
			func() string { return fmt.Sprintf("Directive %q", "@"+dirName) },
		)
	}
	return
}

type nameSet map[string]errors.Location

func (c *context) validateName(set nameSet, name common.Ident, rule string, kind string) {
	c.validateNameCustomMsg(set, name, rule, func() string {
		return fmt.Sprintf("There can be only one %s named %q.", kind, name.Name)
	})
}

func (c *context) validateNameCustomMsg(set nameSet, name common.Ident, rule string, msg func() string) {
	if loc, ok := set[name.Name]; ok {
		c.addErrMultiLoc([]errors.Location{loc, name.Loc}, rule, msg())
		return
	}
	set[name.Name] = name.Loc
	return
}

func (c *context) validateArguments(args common.ArgumentList, argDecls common.InputValueList, loc errors.Location, owner1, owner2 func() string) {
	for _, selArg := range args {
		arg := argDecls.Get(selArg.Name.Name)
		if arg == nil {
			c.addErr(selArg.Name.Loc, "KnownArgumentNames", "Unknown argument %q on %s.", selArg.Name.Name, owner1())
			continue
		}
		value := selArg.Value
		if ok, reason := validateValue(value.Value, arg.Type); !ok {
			c.addErr(value.Loc, "ArgumentsOfCorrectType", "Argument %q has invalid value %s.\n%s", arg.Name.Name, common.Stringify(value.Value), reason)
		}
	}
	for _, decl := range argDecls {
		if _, ok := decl.Type.(*common.NonNull); ok {
			if _, ok := args.Get(decl.Name.Name); !ok {
				c.addErr(loc, "ProvidedNonNullArguments", "%s argument %q of type %q is required but not provided.", owner2(), decl.Name.Name, decl.Type)
			}
		}
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

	if _, ok := v.(common.Variable); ok {
		// TODO
		return true, ""
	}

	switch t := t.(type) {
	case *schema.Scalar, *schema.Enum:
		if lit, ok := v.(*common.BasicLit); ok {
			if validateBasicLit(lit, t) {
				return true, ""
			}
		}

	case *common.List:
		list, ok := v.([]interface{})
		if !ok {
			return validateValue(v, t.OfType) // single value instead of list
		}
		for i, entry := range list {
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

	return false, fmt.Sprintf("Expected type %q, found %s.", t, common.Stringify(v))
}

func validateBasicLit(v *common.BasicLit, t common.Type) bool {
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

func canBeInput(t common.Type) bool {
	switch t := t.(type) {
	case *schema.InputObject, *schema.Scalar, *schema.Enum:
		return true
	case *common.List:
		return canBeInput(t.OfType)
	case *common.NonNull:
		return canBeInput(t.OfType)
	default:
		return false
	}
}

func hasSubfields(t common.Type) bool {
	switch t := t.(type) {
	case *schema.Object, *schema.Interface, *schema.Union:
		return true
	case *common.List:
		return hasSubfields(t.OfType)
	case *common.NonNull:
		return hasSubfields(t.OfType)
	default:
		return false
	}
}
