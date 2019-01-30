package codegen

import (
	"bytes"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"go/types"

	"github.com/99designs/gqlgen/codegen/templates"
)

type GoFieldType int

const (
	GoFieldUndefined GoFieldType = iota
	GoFieldMethod
	GoFieldVariable
)

type Object struct {
	Definition         *TypeDefinition
	Fields             []*Field
	Implements         []*TypeDefinition
	ResolverInterface  types.Type
	Root               bool
	DisableConcurrency bool
	Stream             bool
	Directives         []*Directive
	InTypemap          bool
}

type Objects []*Object

func (o *Object) Implementors() string {
	satisfiedBy := strconv.Quote(o.Definition.GQLDefinition.Name)
	for _, s := range o.Definition.GQLDefinition.Interfaces {
		satisfiedBy += ", " + strconv.Quote(s)
	}
	return "[]string{" + satisfiedBy + "}"
}

func (o *Object) HasResolvers() bool {
	for _, f := range o.Fields {
		if f.IsResolver {
			return true
		}
	}
	return false
}
func (o *Object) HasDirectives() bool {
	if len(o.Directives) > 0 {
		return true
	}
	for _, f := range o.Fields {
		if f.HasDirectives() {
			return true
		}
	}

	return false
}

func (o *Object) IsConcurrent() bool {
	for _, f := range o.Fields {
		if f.IsConcurrent() {
			return true
		}
	}
	return false
}

func (o *Object) IsReserved() bool {
	return strings.HasPrefix(o.Definition.GQLDefinition.Name, "__")
}

func (o *Object) Description() string {
	return o.Definition.GQLDefinition.Description
}

func (os Objects) ByName(name string) *Object {
	for i, o := range os {
		if strings.EqualFold(o.Definition.GQLDefinition.Name, name) {
			return os[i]
		}
	}
	return nil
}

func tpl(tpl string, vars map[string]interface{}) string {
	b := &bytes.Buffer{}
	err := template.Must(template.New("inline").Funcs(templates.Funcs()).Parse(tpl)).Execute(b, vars)
	if err != nil {
		panic(err)
	}
	return b.String()
}

func ucFirst(s string) string {
	if s == "" {
		return ""
	}

	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
