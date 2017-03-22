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
	Operations OperationList
	Fragments  map[string]*NamedFragment
}

type OperationList []*Operation

func (l OperationList) Get(name string) *Operation {
	for _, f := range l {
		if f.Name == name {
			return f
		}
	}
	return nil
}

type Operation struct {
	Type       OperationType
	Name       string
	Vars       common.InputValueList
	SelSet     *SelectionSet
	Directives map[string]*common.Directive
	Loc        errors.Location
}

type OperationType string

const (
	Query    OperationType = "QUERY"
	Mutation               = "MUTATION"
)

type Fragment struct {
	On     common.TypeName
	SelSet *SelectionSet
}

type NamedFragment struct {
	Fragment
	Name       string
	Directives map[string]*common.Directive
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
	Directives map[string]*common.Directive
	SelSet     *SelectionSet
	Loc        errors.Location
}

type InlineFragment struct {
	Fragment
	Directives map[string]*common.Directive
}

type FragmentSpread struct {
	Name       lexer.Ident
	Directives map[string]*common.Directive
}

func (Field) isSelection()          {}
func (InlineFragment) isSelection() {}
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

	return doc, nil
}

func parseDocument(l *lexer.Lexer) *Document {
	d := &Document{
		Fragments: make(map[string]*NamedFragment),
	}
	for l.Peek() != scanner.EOF {
		if l.Peek() == '{' {
			op := &Operation{Type: Query}
			op.Loc = l.Location()
			op.SelSet = parseSelectionSet(l)
			d.Operations = append(d.Operations, op)
			continue
		}

		switch x := l.ConsumeIdent(); x {
		case "query":
			d.Operations = append(d.Operations, parseOperation(l, Query))

		case "mutation":
			d.Operations = append(d.Operations, parseOperation(l, Mutation))

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
	op.Loc = l.Location()
	if l.Peek() == scanner.Ident {
		op.Name = l.ConsumeIdent()
	}
	op.Directives = common.ParseDirectives(l)
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
	f.On = common.TypeName{Ident: l.ConsumeIdentWithLoc()}
	f.Directives = common.ParseDirectives(l)
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
		Loc: l.Location(),
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

	f := &InlineFragment{}
	if l.Peek() == scanner.Ident {
		ident := l.ConsumeIdentWithLoc()
		if ident.Name != "on" {
			fs := &FragmentSpread{
				Name: ident,
			}
			fs.Directives = common.ParseDirectives(l)
			return fs
		}
		f.On = common.TypeName{Ident: l.ConsumeIdentWithLoc()}
	}
	f.Directives = common.ParseDirectives(l)
	f.SelSet = parseSelectionSet(l)
	return f
}
