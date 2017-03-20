package query

import (
	"fmt"
	"strings"
	"text/scanner"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/lexer"
)

type Document struct {
	Operations map[string]*Operation
	Fragments  map[string]*NamedFragment
}

type Operation struct {
	Type   OperationType
	Name   string
	Vars   common.InputValueList
	SelSet *SelectionSet
}

type OperationType int

const (
	Query OperationType = iota
	Mutation
)

type NamedFragment struct {
	Fragment
	Name string
}

type Fragment struct {
	On         string
	SelSet     *SelectionSet
	Directives map[string]common.ArgumentList
}

type SelectionSet struct {
	Selections []Selection
}

type Selection interface {
	isSelection()
}

type Field struct {
	Alias      string
	Name       string
	Arguments  common.ArgumentList
	Directives map[string]common.ArgumentList
	SelSet     *SelectionSet
	Location   *errors.Location
}

type FragmentSpread struct {
	Name       string
	Directives map[string]common.ArgumentList
}

func (Field) isSelection()          {}
func (Fragment) isSelection()       {}
func (FragmentSpread) isSelection() {}

func Parse(queryString string) (*Document, *errors.QueryError) {
	sc := &scanner.Scanner{
		Mode: scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings,
	}
	sc.Init(strings.NewReader(queryString))

	l := lexer.New(sc)
	var doc *Document
	err := l.CatchSyntaxError(func() {
		doc = parseDocument(l)
	})
	if err != nil {
		return nil, err
	}

	for _, op := range doc.Operations {
		if err := resolveSelSet(doc, op.SelSet); err != nil {
			return nil, err
		}
	}

	for _, f := range doc.Fragments {
		if err := resolveSelSet(doc, f.Fragment.SelSet); err != nil {
			return nil, err
		}
	}

	return doc, nil
}

func resolveSelSet(doc *Document, selSet *SelectionSet) *errors.QueryError {
	var err *errors.QueryError
	for i, sel := range selSet.Selections {
		selSet.Selections[i], err = resolveSelection(doc, sel)
		if err != nil {
			return err
		}
	}
	return nil
}

func resolveSelection(doc *Document, sel Selection) (Selection, *errors.QueryError) {
	switch sel := sel.(type) {
	case *Field:
		if sel.SelSet != nil {
			if err := resolveSelSet(doc, sel.SelSet); err != nil {
				return nil, err
			}
		}
		return sel, nil

	case *FragmentSpread:
		frag, ok := doc.Fragments[sel.Name]
		if !ok {
			return nil, errors.Errorf("fragment %q not found", sel.Name)
		}
		return &Fragment{
			On:         frag.On,
			SelSet:     frag.SelSet,
			Directives: sel.Directives,
		}, nil

	case *Fragment:
		return sel, nil

	default:
		panic("unreachable")
	}
}

func parseDocument(l *lexer.Lexer) *Document {
	d := &Document{
		Operations: make(map[string]*Operation),
		Fragments:  make(map[string]*NamedFragment),
	}
	for l.Peek() != scanner.EOF {
		if l.Peek() == '{' {
			d.Operations[""] = &Operation{SelSet: parseSelectionSet(l)}
			continue
		}

		switch x := l.ConsumeIdent(); x {
		case "query":
			q := parseOperation(l, Query)
			d.Operations[q.Name] = q

		case "mutation":
			q := parseOperation(l, Mutation)
			d.Operations[q.Name] = q

		case "fragment":
			f := parseFragment(l)
			d.Fragments[f.Name] = f

		default:
			l.SyntaxError(fmt.Sprintf(`unexpected %q, expecting "fragment"`, x))
		}
	}
	return d
}

func parseOperation(l *lexer.Lexer, opType OperationType) *Operation {
	op := &Operation{Type: opType}
	if l.Peek() == scanner.Ident {
		op.Name = l.ConsumeIdent()
	}
	if l.Peek() == '(' {
		l.ConsumeToken('(')
		for l.Peek() != ')' {
			l.ConsumeToken('$')
			op.Vars = append(op.Vars, common.ParseInputValue(l))
		}
		l.ConsumeToken(')')
	}
	op.SelSet = parseSelectionSet(l)
	return op
}

func parseFragment(l *lexer.Lexer) *NamedFragment {
	f := &NamedFragment{}
	f.Name = l.ConsumeIdent()
	l.ConsumeKeyword("on")
	f.On = l.ConsumeIdent()
	f.SelSet = parseSelectionSet(l)
	return f
}

func parseSelectionSet(l *lexer.Lexer) *SelectionSet {
	sel := &SelectionSet{}
	l.ConsumeToken('{')
	for l.Peek() != '}' {
		sel.Selections = append(sel.Selections, parseSelection(l))
	}
	l.ConsumeToken('}')
	return sel
}

func parseSelection(l *lexer.Lexer) Selection {
	if l.Peek() == '.' {
		return parseSpread(l)
	}
	return parseField(l)
}

func parseField(l *lexer.Lexer) *Field {
	f := &Field{
		Location: l.Location(),
	}
	f.Alias = l.ConsumeIdent()
	f.Name = f.Alias
	if l.Peek() == ':' {
		l.ConsumeToken(':')
		f.Name = l.ConsumeIdent()
	}
	if l.Peek() == '(' {
		f.Arguments = common.ParseArguments(l)
	}
	f.Directives = common.ParseDirectives(l)
	if l.Peek() == '{' {
		f.SelSet = parseSelectionSet(l)
	}
	return f
}

func parseSpread(l *lexer.Lexer) Selection {
	l.ConsumeToken('.')
	l.ConsumeToken('.')
	l.ConsumeToken('.')

	f := &Fragment{}
	if l.Peek() == scanner.Ident {
		ident := l.ConsumeIdent()
		if ident != "on" {
			fs := &FragmentSpread{
				Name: ident,
			}
			fs.Directives = common.ParseDirectives(l)
			return fs
		}
		f.On = l.ConsumeIdent()
	}
	f.Directives = common.ParseDirectives(l)
	f.SelSet = parseSelectionSet(l)
	return f
}
