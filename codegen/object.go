package codegen

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

type Object struct {
	*NamedType

	Fields             []Field
	Satisfies          []string
	Root               bool
	DisableConcurrency bool
	Stream             bool
}

type Field struct {
	*Type

	GQLName      string          // The name of the field in graphql
	GoMethodName string          // The name of the method in go, if any
	GoVarName    string          // The name of the var in go, if any
	Args         []FieldArgument // A list of arguments to be passed to this field
	NoErr        bool            // If this is bound to a go method, does that method have an error as the second argument
	Object       *Object         // A link back to the parent object
}

type FieldArgument struct {
	*Type

	GQLName string // The name of the argument in graphql
}

type Objects []*Object

func (o *Object) GetField(name string) *Field {
	for i, field := range o.Fields {
		if strings.EqualFold(field.GQLName, name) {
			return &o.Fields[i]
		}
	}
	return nil
}

func (o *Object) Implementors() string {
	satisfiedBy := strconv.Quote(o.GQLType)
	for _, s := range o.Satisfies {
		satisfiedBy += ", " + strconv.Quote(s)
	}
	return "[]string{" + satisfiedBy + "}"
}

func (f *Field) IsResolver() bool {
	return f.GoMethodName == "" && f.GoVarName == ""
}

func (f *Field) IsConcurrent() bool {
	return f.IsResolver() && !f.Object.DisableConcurrency
}

func (f *Field) ResolverDeclaration() string {
	if !f.IsResolver() {
		return ""
	}
	res := fmt.Sprintf("%s_%s(ctx context.Context", f.Object.GQLType, f.GQLName)

	if !f.Object.Root {
		res += fmt.Sprintf(", it *%s", f.Object.FullName())
	}
	for _, arg := range f.Args {
		res += fmt.Sprintf(", %s %s", arg.GQLName, arg.Signature())
	}

	result := f.Signature()
	if f.Object.Stream {
		result = "<-chan " + result
	}

	res += fmt.Sprintf(") (%s, error)", result)
	return res
}

func (f *Field) CallArgs() string {
	var args []string

	if f.GoMethodName == "" {
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
				{{.res}} = graphql.Null
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
			{{.arr}} := graphql.Array{}
			for {{.index}} := range {{.val}} {
				var {{.tmp}} graphql.Marshaler
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

	case f.IsScalar:
		if isPtr {
			val = "*" + val
		}
		return f.Marshal(res, val)

	default:
		if !isPtr {
			val = "&" + val
		}
		return fmt.Sprintf("%s = ec._%s(field.Selections, %s)", res, lcFirst(f.GQLType), val)
	}
}

func (os Objects) ByName(name string) *Object {
	for i, o := range os {
		if strings.EqualFold(o.GQLType, name) {
			return os[i]
		}
	}
	return nil
}

func tpl(tpl string, vars map[string]interface{}) string {
	b := &bytes.Buffer{}
	template.Must(template.New("inline").Parse(tpl)).Execute(b, vars)
	return b.String()
}

func lcFirst(s string) string {
	if s == "" {
		return ""
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}
