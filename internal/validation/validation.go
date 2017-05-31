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

type varSet map[*common.InputValue]struct{}

type context struct {
	schema   *schema.Schema
	doc      *query.Document
	errs     []*errors.QueryError
	opErrs   map[*query.Operation][]*errors.QueryError
	usedVars map[*query.Operation]varSet
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

type opContext struct {
	*context
	ops []*query.Operation
}

func Validate(s *schema.Schema, doc *query.Document) []*errors.QueryError {
	c := &context{
		schema:   s,
		doc:      doc,
		opErrs:   make(map[*query.Operation][]*errors.QueryError),
		usedVars: make(map[*query.Operation]varSet),
	}

	opNames := make(nameSet)
	fragUsedBy := make(map[*query.FragmentDecl][]*query.Operation)
	for _, op := range doc.Operations {
		c.usedVars[op] = make(varSet)
		opc := &opContext{c, []*query.Operation{op}}

		if op.Name.Name == "" && len(doc.Operations) != 1 {
			c.addErr(op.Loc, "LoneAnonymousOperation", "This anonymous operation must be the only defined operation.")
		}
		if op.Name.Name != "" {
			validateName(c, opNames, op.Name, "UniqueOperationNames", "operation")
		}

		validateDirectives(opc, string(op.Type), op.Directives)

		varNames := make(nameSet)
		for _, v := range op.Vars {
			validateName(c, varNames, v.Name, "UniqueVariableNames", "variable")

			t := resolveType(c, v.Type)
			if !canBeInput(t) {
				c.addErr(v.TypeLoc, "VariablesAreInputTypes", "Variable %q cannot be non-input type %q.", "$"+v.Name.Name, t)
			}

			if v.Default != nil {
				validateLiteral(opc, v.Default)

				if t != nil {
					if nn, ok := t.(*common.NonNull); ok {
						c.addErr(v.Default.Location(), "DefaultValuesOfCorrectType", "Variable %q of type %q is required and will not use the default value. Perhaps you meant to use type %q.", "$"+v.Name.Name, t, nn.OfType)
					}

					if ok, reason := validateValueType(opc, v.Default, t); !ok {
						c.addErr(v.Default.Location(), "DefaultValuesOfCorrectType", "Variable %q of type %q has invalid default value %s.\n%s", "$"+v.Name.Name, t, v.Default, reason)
					}
				}
			}
		}

		var entryPoint schema.NamedType
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

		validateSelectionSet(opc, op.SelSet, entryPoint)

		fragUsed := make(map[*query.FragmentDecl]struct{})
		markUsedFragments(c, op.SelSet, fragUsed)
		for frag := range fragUsed {
			fragUsedBy[frag] = append(fragUsedBy[frag], op)
		}
	}

	fragNames := make(nameSet)
	fragVisited := make(map[*query.FragmentDecl]struct{})
	for _, frag := range doc.Fragments {
		opc := &opContext{c, fragUsedBy[frag]}

		validateName(c, fragNames, frag.Name, "UniqueFragmentNames", "fragment")
		validateDirectives(opc, "FRAGMENT_DEFINITION", frag.Directives)

		t := unwrapType(resolveType(c, &frag.On))
		// continue even if t is nil
		if t != nil && !canBeFragment(t) {
			c.addErr(frag.On.Loc, "FragmentsOnCompositeTypes", "Fragment %q cannot condition on non composite type %q.", frag.Name.Name, t)
			continue
		}

		validateSelectionSet(opc, frag.SelSet, t)

		if _, ok := fragVisited[frag]; !ok {
			detectFragmentCycle(c, frag.SelSet, fragVisited, nil, map[string]int{frag.Name.Name: 0})
		}
	}

	for _, frag := range doc.Fragments {
		if len(fragUsedBy[frag]) == 0 {
			c.addErr(frag.Loc, "NoUnusedFragments", "Fragment %q is never used.", frag.Name.Name)
		}
	}

	for _, op := range doc.Operations {
		c.errs = append(c.errs, c.opErrs[op]...)

		opUsedVars := c.usedVars[op]
		for _, v := range op.Vars {
			if _, ok := opUsedVars[v]; !ok {
				opSuffix := ""
				if op.Name.Name != "" {
					opSuffix = fmt.Sprintf(" in operation %q", op.Name.Name)
				}
				c.addErr(v.Loc, "NoUnusedVariables", "Variable %q is never used%s.", "$"+v.Name.Name, opSuffix)
			}
		}
	}

	return c.errs
}

func validateSelectionSet(c *opContext, selSet *query.SelectionSet, t schema.NamedType) {
	for _, sel := range selSet.Selections {
		validateSelection(c, sel, t)
	}
}

func validateSelection(c *opContext, sel query.Selection, t schema.NamedType) {
	switch sel := sel.(type) {
	case *query.Field:
		validateDirectives(c, "FIELD", sel.Directives)

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

		validateArgumentLiterals(c, sel.Arguments)
		if f != nil {
			validateArgumentTypes(c, sel.Arguments, f.Args, sel.Alias.Loc,
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
			validateSelectionSet(c, sel.SelSet, unwrapType(ft))
		}

	case *query.InlineFragment:
		validateDirectives(c, "INLINE_FRAGMENT", sel.Directives)
		if sel.On.Name != "" {
			fragTyp := unwrapType(resolveType(c.context, &sel.On))
			if fragTyp != nil && !compatible(t, fragTyp) {
				c.addErr(sel.Loc, "PossibleFragmentSpreads", "Fragment cannot be spread here as objects of type %q can never be of type %q.", t, fragTyp)
			}
			t = fragTyp
			// continue even if t is nil
		}
		if t != nil && !canBeFragment(t) {
			c.addErr(sel.On.Loc, "FragmentsOnCompositeTypes", "Fragment cannot condition on non composite type %q.", t)
			return
		}
		validateSelectionSet(c, sel.SelSet, unwrapType(t))

	case *query.FragmentSpread:
		validateDirectives(c, "FRAGMENT_SPREAD", sel.Directives)
		frag := c.doc.Fragments.Get(sel.Name.Name)
		if frag == nil {
			c.addErr(sel.Name.Loc, "KnownFragmentNames", "Unknown fragment %q.", sel.Name.Name)
			return
		}
		fragTyp := c.schema.Types[frag.On.Name]
		if !compatible(t, fragTyp) {
			c.addErr(sel.Loc, "PossibleFragmentSpreads", "Fragment %q cannot be spread here as objects of type %q can never be of type %q.", frag.Name.Name, t, fragTyp)
		}

	default:
		panic("unreachable")
	}
}

func compatible(a, b common.Type) bool {
	for _, pta := range possibleTypes(a) {
		for _, ptb := range possibleTypes(b) {
			if pta == ptb {
				return true
			}
		}
	}
	return false
}

func possibleTypes(t common.Type) []*schema.Object {
	switch t := t.(type) {
	case *schema.Object:
		return []*schema.Object{t}
	case *schema.Interface:
		return t.PossibleTypes
	case *schema.Union:
		return t.PossibleTypes
	default:
		return nil
	}
}

func markUsedFragments(c *context, selSet *query.SelectionSet, fragUsed map[*query.FragmentDecl]struct{}) {
	for _, sel := range selSet.Selections {
		switch sel := sel.(type) {
		case *query.Field:
			if sel.SelSet != nil {
				markUsedFragments(c, sel.SelSet, fragUsed)
			}

		case *query.InlineFragment:
			markUsedFragments(c, sel.SelSet, fragUsed)

		case *query.FragmentSpread:
			frag := c.doc.Fragments.Get(sel.Name.Name)
			if frag == nil {
				return
			}

			if _, ok := fragUsed[frag]; ok {
				return
			}
			fragUsed[frag] = struct{}{}
			markUsedFragments(c, frag.SelSet, fragUsed)

		default:
			panic("unreachable")
		}
	}
}

func detectFragmentCycle(c *context, selSet *query.SelectionSet, fragVisited map[*query.FragmentDecl]struct{}, spreadPath []*query.FragmentSpread, spreadPathIndex map[string]int) {
	for _, sel := range selSet.Selections {
		detectFragmentCycleSel(c, sel, fragVisited, spreadPath, spreadPathIndex)
	}
}

func detectFragmentCycleSel(c *context, sel query.Selection, fragVisited map[*query.FragmentDecl]struct{}, spreadPath []*query.FragmentSpread, spreadPathIndex map[string]int) {
	switch sel := sel.(type) {
	case *query.Field:
		if sel.SelSet != nil {
			detectFragmentCycle(c, sel.SelSet, fragVisited, spreadPath, spreadPathIndex)
		}

	case *query.InlineFragment:
		detectFragmentCycle(c, sel.SelSet, fragVisited, spreadPath, spreadPathIndex)

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
		detectFragmentCycle(c, frag.SelSet, fragVisited, spreadPath, spreadPathIndex)
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

func unwrapType(t common.Type) schema.NamedType {
	if t == nil {
		return nil
	}
	for {
		switch t2 := t.(type) {
		case schema.NamedType:
			return t2
		case *common.List:
			t = t2.OfType
		case *common.NonNull:
			t = t2.OfType
		default:
			panic("unreachable")
		}
	}
}

func resolveType(c *context, t common.Type) common.Type {
	t2, err := common.ResolveType(t, c.schema.Resolve)
	if err != nil {
		c.errs = append(c.errs, err)
	}
	return t2
}

func validateDirectives(c *opContext, loc string, directives common.DirectiveList) {
	directiveNames := make(nameSet)
	for _, d := range directives {
		dirName := d.Name.Name
		validateNameCustomMsg(c.context, directiveNames, d.Name, "UniqueDirectivesPerLocation", func() string {
			return fmt.Sprintf("The directive %q can only be used once at this location.", dirName)
		})

		validateArgumentLiterals(c, d.Args)

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

		validateArgumentTypes(c, d.Args, dd.Args, d.Name.Loc,
			func() string { return fmt.Sprintf("directive %q", "@"+dirName) },
			func() string { return fmt.Sprintf("Directive %q", "@"+dirName) },
		)
	}
	return
}

type nameSet map[string]errors.Location

func validateName(c *context, set nameSet, name common.Ident, rule string, kind string) {
	validateNameCustomMsg(c, set, name, rule, func() string {
		return fmt.Sprintf("There can be only one %s named %q.", kind, name.Name)
	})
}

func validateNameCustomMsg(c *context, set nameSet, name common.Ident, rule string, msg func() string) {
	if loc, ok := set[name.Name]; ok {
		c.addErrMultiLoc([]errors.Location{loc, name.Loc}, rule, msg())
		return
	}
	set[name.Name] = name.Loc
	return
}

func validateArgumentTypes(c *opContext, args common.ArgumentList, argDecls common.InputValueList, loc errors.Location, owner1, owner2 func() string) {
	for _, selArg := range args {
		arg := argDecls.Get(selArg.Name.Name)
		if arg == nil {
			c.addErr(selArg.Name.Loc, "KnownArgumentNames", "Unknown argument %q on %s.", selArg.Name.Name, owner1())
			continue
		}
		value := selArg.Value
		if ok, reason := validateValueType(c, value, arg.Type); !ok {
			c.addErr(value.Location(), "ArgumentsOfCorrectType", "Argument %q has invalid value %s.\n%s", arg.Name.Name, value, reason)
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

func validateArgumentLiterals(c *opContext, args common.ArgumentList) {
	argNames := make(nameSet)
	for _, arg := range args {
		validateName(c.context, argNames, arg.Name, "UniqueArgumentNames", "argument")
		validateLiteral(c, arg.Value)
	}
}

func validateLiteral(c *opContext, l common.Literal) {
	switch l := l.(type) {
	case *common.ObjectLit:
		fieldNames := make(nameSet)
		for _, f := range l.Fields {
			validateName(c.context, fieldNames, f.Name, "UniqueInputFieldNames", "input field")
			validateLiteral(c, f.Value)
		}
	case *common.ListLit:
		for _, entry := range l.Entries {
			validateLiteral(c, entry)
		}
	case *common.Variable:
		for _, op := range c.ops {
			v := op.Vars.Get(l.Name)
			if v == nil {
				byOp := ""
				if op.Name.Name != "" {
					byOp = fmt.Sprintf(" by operation %q", op.Name.Name)
				}
				c.opErrs[op] = append(c.opErrs[op], &errors.QueryError{
					Message:   fmt.Sprintf("Variable %q is not defined%s.", "$"+l.Name, byOp),
					Locations: []errors.Location{l.Loc, op.Loc},
					Rule:      "NoUndefinedVariables",
				})
				continue
			}
			c.usedVars[op][v] = struct{}{}
		}
	}
}

func validateValueType(c *opContext, v common.Literal, t common.Type) (bool, string) {
	if v, ok := v.(*common.Variable); ok {
		for _, op := range c.ops {
			if v2 := op.Vars.Get(v.Name); v2 != nil {
				t2, err := common.ResolveType(v2.Type, c.schema.Resolve)
				if _, ok := t2.(*common.NonNull); !ok && v2.Default != nil {
					t2 = &common.NonNull{OfType: t2}
				}
				if err == nil && !typeCanBeUsedAs(t2, t) {
					c.addErrMultiLoc([]errors.Location{v2.Loc, v.Loc}, "VariablesInAllowedPosition", "Variable %q of type %q used in position expecting type %q.", "$"+v.Name, t2, t)
				}
			}
		}
		return true, ""
	}

	if nn, ok := t.(*common.NonNull); ok {
		if isNull(v) {
			return false, fmt.Sprintf("Expected %q, found null.", t)
		}
		t = nn.OfType
	}
	if isNull(v) {
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
		list, ok := v.(*common.ListLit)
		if !ok {
			return validateValueType(c, v, t.OfType) // single value instead of list
		}
		for i, entry := range list.Entries {
			if ok, reason := validateValueType(c, entry, t.OfType); !ok {
				return false, fmt.Sprintf("In element #%d: %s", i, reason)
			}
		}
		return true, ""

	case *schema.InputObject:
		v, ok := v.(*common.ObjectLit)
		if !ok {
			return false, fmt.Sprintf("Expected %q, found not an object.", t)
		}
		for _, f := range v.Fields {
			name := f.Name.Name
			iv := t.Values.Get(name)
			if iv == nil {
				return false, fmt.Sprintf("In field %q: Unknown field.", name)
			}
			if ok, reason := validateValueType(c, f.Value, iv.Type); !ok {
				return false, fmt.Sprintf("In field %q: %s", name, reason)
			}
		}
		for _, iv := range t.Values {
			found := false
			for _, f := range v.Fields {
				if f.Name.Name == iv.Name.Name {
					found = true
					break
				}
			}
			if !found {
				if _, ok := iv.Type.(*common.NonNull); ok && iv.Default == nil {
					return false, fmt.Sprintf("In field %q: Expected %q, found null.", iv.Name.Name, iv.Type)
				}
			}
		}
		return true, ""
	}

	return false, fmt.Sprintf("Expected type %q, found %s.", t, v)
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
		default:
			//TODO: Type-check against expected type by Unmarshalling
			return true
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

func isNull(lit interface{}) bool {
	_, ok := lit.(*common.NullLit)
	return ok
}

func typeCanBeUsedAs(t, as common.Type) bool {
	nnT, okT := t.(*common.NonNull)
	if okT {
		t = nnT.OfType
	}

	nnAs, okAs := as.(*common.NonNull)
	if okAs {
		as = nnAs.OfType
		if !okT {
			return false // nullable can not be used as non-null
		}
	}

	if t == as {
		return true
	}

	if lT, ok := t.(*common.List); ok {
		if lAs, ok := as.(*common.List); ok {
			return typeCanBeUsedAs(lT.OfType, lAs.OfType)
		}
	}
	return false
}
