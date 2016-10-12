package graphql

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"text/scanner"
)

type Schema struct {
	types map[string]*object
}

type typ interface {
	exec() interface{}
}

type scalar struct {
	resolver reflect.Value
}

type object struct {
	fields map[string]typ
}

type parseError string

func NewSchema(schema string, filename string, resolver interface{}) (res *Schema, errRes error) {
	sc := &scanner.Scanner{}
	sc.Filename = filename
	sc.Init(strings.NewReader(schema))

	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(parseError); ok {
				errRes = errors.New(string(err))
				return
			}
			panic(err)
		}
	}()

	return parseFile(sc, reflect.ValueOf(resolver)), nil
}

func parseFile(sc *scanner.Scanner, r reflect.Value) *Schema {
	types := make(map[string]*object)

	scanToken(sc, scanner.Ident)
	switch sc.TokenText() {
	case "type":
		name, obj := parseTypeDecl(sc, r)
		types[name] = obj
	default:
		syntaxError(sc, `"type"`)
	}

	return &Schema{
		types: types,
	}
}

func parseTypeDecl(sc *scanner.Scanner, r reflect.Value) (string, *object) {
	typeName := scanIdent(sc)
	scanToken(sc, '{')

	fields := make(map[string]typ)

	fieldName := scanIdent(sc)
	m := r.MethodByName(strings.ToUpper(fieldName[:1]) + fieldName[1:])
	scanToken(sc, ':')
	fields[fieldName] = parseType(sc, m)

	scanToken(sc, '}')

	return typeName, &object{
		fields: fields,
	}
}

func parseType(sc *scanner.Scanner, r reflect.Value) typ {
	// TODO check args
	// TODO check return type
	scanToken(sc, scanner.Ident)
	return &scalar{
		resolver: r,
	}
}

func scanIdent(sc *scanner.Scanner) string {
	scanToken(sc, scanner.Ident)
	return sc.TokenText()
}

func scanToken(sc *scanner.Scanner, expected rune) {
	if got := sc.Scan(); got != expected {
		syntaxError(sc, scanner.TokenString(expected))
	}
}

func syntaxError(sc *scanner.Scanner, expected string) {
	panic(parseError(fmt.Sprintf("%s:%d: syntax error: unexpected %q, expecting %s", sc.Filename, sc.Line, sc.TokenText(), expected)))
}

func (s *Schema) Exec(query string) (interface{}, error) {
	sc := &scanner.Scanner{}
	sc.Init(strings.NewReader(query))

	res := s.types["Query"].exec(parseSelectionSet(sc))
	return res, nil
}

type selectionSet struct {
	selections []*field
}

func parseSelectionSet(sc *scanner.Scanner) *selectionSet {
	scanToken(sc, '{')
	f := parseField(sc)
	scanToken(sc, '}')
	return &selectionSet{
		selections: []*field{f},
	}
}

type field struct {
	name string
}

func parseField(sc *scanner.Scanner) *field {
	name := scanIdent(sc)
	return &field{
		name: name,
	}
}

func (o *object) exec(sel *selectionSet) interface{} {
	return o.fields[sel.selections[0].name].exec()
}

func (s *scalar) exec() interface{} {
	return s.resolver.Call(nil)[0].Interface()
}
