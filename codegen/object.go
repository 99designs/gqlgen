package codegen

import (
	"bytes"
	"go/types"
	"log"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/99designs/gqlgen/codegen/config"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

type GoFieldType int

const (
	GoFieldUndefined GoFieldType = iota
	GoFieldMethod
	GoFieldVariable
)

type Object struct {
	*ast.Definition

	Type               types.Type
	ResolverInterface  types.Type
	Root               bool
	Fields             []*Field
	Implements         []*ast.Definition
	DisableConcurrency bool
	Stream             bool
	Directives         []*Directive
}

func (b *builder) buildObject(typ *ast.Definition) (*Object, error) {
	dirs, err := b.getDirectives(typ.Directives)
	if err != nil {
		return nil, errors.Wrap(err, typ.Name)
	}

	obj := &Object{
		Definition:         typ,
		Root:               b.Schema.Query == typ || b.Schema.Mutation == typ || b.Schema.Subscription == typ,
		DisableConcurrency: typ == b.Schema.Mutation,
		Stream:             typ == b.Schema.Subscription,
		Directives:         dirs,
		ResolverInterface: types.NewNamed(
			types.NewTypeName(0, b.Config.Exec.Pkg(), typ.Name+"Resolver", nil),
			nil,
			nil,
		),
	}

	if !obj.Root {
		goObject, err := b.Binder.FindUserObject(typ.Name)
		if err != nil {
			return nil, err
		}
		obj.Type = goObject
	}

	for _, intf := range b.Schema.GetImplements(typ) {
		obj.Implements = append(obj.Implements, b.Schema.Types[intf.Name])
	}

	for _, field := range typ.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}

		var f *Field
		f, err = b.buildField(obj, field)
		if err != nil {
			return nil, errors.Wrap(err, typ.Name+"."+field.Name)
		}

		if typ.Kind == ast.InputObject && !f.TypeReference.Definition.GQLDefinition.IsInputType() {
			return nil, errors.Errorf(
				"%s.%s: cannot use %s because %s is not a valid input type",
				typ.Name,
				field.Name,
				f.Definition.GQLDefinition.Name,
				f.TypeReference.Definition.GQLDefinition.Kind,
			)
		}

		obj.Fields = append(obj.Fields, f)

		if obj.Root {
			f.IsResolver = true
		} else if !f.IsResolver {
			// first try binding to a method
			methodErr := b.bindMethod(obj.Type, f)
			if methodErr == nil {
				continue
			}

			// otherwise try binding to a var
			varErr := b.bindVar(obj.Type, f)

			// if both failed, add a resolver
			if varErr != nil {
				f.IsResolver = true

				log.Printf("\nadding resolver method for %s.%s to %s\n  %s\n  %s",
					obj.Name,
					field.Name,
					obj.Type.String(),
					methodErr.Error(),
					varErr.Error())
			}
		}
	}

	return obj, nil
}

type Objects []*Object

func (o *Object) Implementors() string {
	satisfiedBy := strconv.Quote(o.Name)
	for _, s := range o.Definition.Interfaces {
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

func (o *Object) HasUnmarshal() bool {
	if o.Type == config.MapType {
		return true
	}
	for i := 0; i < o.Type.(*types.Named).NumMethods(); i++ {
		switch o.Type.(*types.Named).Method(i).Name() {
		case "UnmarshalGQL":
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
	return strings.HasPrefix(o.Definition.Name, "__")
}

func (o *Object) Description() string {
	return o.Definition.Description
}

func (os Objects) ByName(name string) *Object {
	for i, o := range os {
		if strings.EqualFold(o.Definition.Name, name) {
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
