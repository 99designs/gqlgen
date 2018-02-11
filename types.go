package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

type kind struct {
	GraphQLName  string
	Name         string
	Package      string
	ImportedAs   string
	Modifiers    []string
	Implementors []kind
	Scalar       bool
}

func (t kind) Local() string {
	return strings.Join(t.Modifiers, "") + t.FullName()
}

func (t kind) Ptr() kind {
	t.Modifiers = append(t.Modifiers, modPtr)
	return t
}

func (t kind) IsPtr() bool {
	return len(t.Modifiers) > 0 && t.Modifiers[0] == modPtr
}

func (t kind) IsSlice() bool {
	return len(t.Modifiers) > 0 && t.Modifiers[0] == modList
}

func (t kind) Elem() kind {
	if len(t.Modifiers) == 0 {
		return t
	}

	t.Modifiers = t.Modifiers[1:]
	return t
}

func (t kind) ByRef(name string) string {
	needPtr := len(t.Implementors) == 0
	if needPtr && !t.IsPtr() {
		return "&" + name
	}
	if !needPtr && t.IsPtr() {
		return "*" + name
	}
	return name
}

func (t kind) ByVal(name string) string {
	if t.IsPtr() {
		return "*" + name
	}
	return name
}

func (t kind) FullName() string {
	if t.ImportedAs == "" {
		return t.Name
	}
	return t.ImportedAs + "." + t.Name
}

type object struct {
	Name               string
	Fields             []Field
	Type               kind
	satisfies          []string
	Root               bool
	DisableConcurrency bool
}

type Field struct {
	GraphQLName string
	MethodName  string
	VarName     string
	Type        kind
	Args        []FieldArgument
	NoErr       bool
	Object      *object
}

func (f *Field) IsResolver() bool {
	return f.MethodName == "" && f.VarName == ""
}

func (f *Field) IsConcurrent() bool {
	return f.IsResolver() && !f.Object.DisableConcurrency
}

func (f *Field) ResolverDeclaration() string {
	if !f.IsResolver() {
		return ""
	}
	res := fmt.Sprintf("%s_%s(ctx context.Context", f.Object.Name, f.GraphQLName)

	if !f.Object.Root {
		res += fmt.Sprintf(", it *%s", f.Object.Type.Local())
	}
	for _, arg := range f.Args {
		res += fmt.Sprintf(", %s %s", arg.Name, arg.Type.Local())
	}

	res += fmt.Sprintf(") (%s, error)", f.Type.Local())
	return res
}

func (f *Field) CallArgs() string {
	var args []string

	if f.MethodName == "" {
		args = append(args, "ec.ctx")

		if !f.Object.Root {
			args = append(args, "it")
		}
	}

	for i := range f.Args {
		args = append(args, "arg"+strconv.Itoa(i))
	}

	return strings.Join(args, ", ")
}

// should be in the template, but its recursive and has a bunch of args
func (f *Field) WriteJson(res string) string {
	return f.doWriteJson(res, "res", f.Type.Modifiers, false, 1)
}

func (f *Field) doWriteJson(res string, val string, remainingMods []string, isPtr bool, depth int) string {
	switch {
	case len(remainingMods) > 0 && remainingMods[0] == modPtr:
		return tpl(`
			if {{.val}} == nil {
				{{.res}} = jsonw.Null				
			} else {
				{{.next}}
			}`, map[string]interface{}{
			"res":  res,
			"val":  val,
			"next": f.doWriteJson(res, val, remainingMods[1:], true, depth+1),
		})

	case len(remainingMods) > 0 && remainingMods[0] == modList:
		if isPtr {
			val = "*" + val
		}
		var tmp = "tmp" + strconv.Itoa(depth)
		var arr = "arr" + strconv.Itoa(depth)
		var index = "idx" + strconv.Itoa(depth)

		return tpl(`
			{{.arr}} := jsonw.Array{}
			for {{.index}} := range {{.val}} {
				var {{.tmp}} jsonw.Writer
				{{.next}}
				{{.arr}} = append({{.arr}}, {{.tmp}})
			}
			{{.res}} = {{.arr}}`, map[string]interface{}{
			"res":   res,
			"val":   val,
			"tmp":   tmp,
			"arr":   arr,
			"index": index,
			"next":  f.doWriteJson(tmp, val+"["+index+"]", remainingMods[1:], false, depth+1),
		})

	case f.Type.Scalar:
		if isPtr {
			val = "*" + val
		}
		return fmt.Sprintf("%s = jsonw.%s(%s)", res, ucFirst(f.Type.Name), val)

	default:
		if !isPtr {
			val = "&" + val
		}
		return fmt.Sprintf("%s = ec._%s(field.Selections, %s)", res, lcFirst(f.Type.GraphQLName), val)
	}
}

func (o *object) GetField(name string) *Field {
	for i, field := range o.Fields {
		if strings.EqualFold(field.GraphQLName, name) {
			return &o.Fields[i]
		}
	}
	return nil
}

func (o *object) Implementors() string {
	satisfiedBy := strconv.Quote(o.Type.GraphQLName)
	for _, s := range o.satisfies {
		satisfiedBy += ", " + strconv.Quote(s)
	}
	return "[]string{" + satisfiedBy + "}"
}

func (e *extractor) GetObject(name string) *object {
	for i, o := range e.Objects {
		if strings.EqualFold(o.Name, name) {
			return e.Objects[i]
		}
	}
	return nil
}

type FieldArgument struct {
	Name string
	Type kind
}

func tpl(tpl string, vars map[string]interface{}) string {
	b := &bytes.Buffer{}
	template.Must(template.New("inline").Parse(tpl)).Execute(b, vars)
	return b.String()
}
