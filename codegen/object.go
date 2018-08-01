package codegen

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

type GoFieldType int

const (
	GoFieldUndefined GoFieldType = iota
	GoFieldMethod
	GoFieldVariable
)

type Object struct {
	*NamedType

	Fields             []Field
	Satisfies          []string
	ResolverInterface  *Ref
	Root               bool
	DisableConcurrency bool
	Stream             bool
}

type Field struct {
	*Type

	GQLName        string          // The name of the field in graphql
	GoFieldType    GoFieldType     // The field type in go, if any
	GoReceiverName string          // The name of method & var receiver in go, if any
	GoFieldName    string          // The name of the method or var in go, if any
	Args           []FieldArgument // A list of arguments to be passed to this field
	ForceResolver  bool            // Should be emit Resolver method
	NoErr          bool            // If this is bound to a go method, does that method have an error as the second argument
	Object         *Object         // A link back to the parent object
	Default        interface{}     // The default value
}

type FieldArgument struct {
	*Type

	GQLName   string      // The name of the argument in graphql
	GoVarName string      // The name of the var in go
	Object    *Object     // A link back to the parent object
	Default   interface{} // The default value
}

type Objects []*Object

func (o *Object) Implementors() string {
	satisfiedBy := strconv.Quote(o.GQLType)
	for _, s := range o.Satisfies {
		satisfiedBy += ", " + strconv.Quote(s)
	}
	return "[]string{" + satisfiedBy + "}"
}

func (o *Object) HasResolvers() bool {
	for _, f := range o.Fields {
		if f.IsResolver() {
			return true
		}
	}
	return false
}

func (f *Field) IsResolver() bool {
	return f.ForceResolver || f.GoFieldName == ""
}

func (f *Field) IsMethod() bool {
	return f.GoFieldType == GoFieldMethod
}

func (f *Field) IsVariable() bool {
	return f.GoFieldType == GoFieldVariable
}

func (f *Field) IsConcurrent() bool {
	return f.IsResolver() && !f.Object.DisableConcurrency
}

func (f *Field) ShortInvocation() string {
	if !f.IsResolver() {
		return ""
	}
	shortName := strings.ToUpper(f.GQLName[:1]) + f.GQLName[1:]

	return fmt.Sprintf("%s().%s(%s)", f.Object.GQLType, shortName, f.CallArgs())
}

func (f *Field) ResolverType() string {
	if !f.IsResolver() {
		return ""
	}
	shortName := strings.ToUpper(f.GQLName[:1]) + f.GQLName[1:]

	return fmt.Sprintf("%s().%s(%s)", f.Object.GQLType, shortName, f.CallArgs())
}

func (f *Field) ShortResolverDeclaration() string {
	if !f.IsResolver() {
		return ""
	}
	decl := strings.TrimPrefix(f.ResolverDeclaration(), f.Object.GQLType+"_")
	return strings.ToUpper(decl[:1]) + decl[1:]
}

func (f *Field) ResolverDeclaration() string {
	if !f.IsResolver() {
		return ""
	}
	res := fmt.Sprintf("%s_%s(ctx context.Context", f.Object.GQLType, f.GQLName)

	if !f.Object.Root {
		res += fmt.Sprintf(", obj *%s", f.Object.FullName())
	}
	for _, arg := range f.Args {
		res += fmt.Sprintf(", %s %s", arg.GoVarName, arg.Signature())
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

	if f.IsResolver() {
		args = append(args, "ctx")

		if !f.Object.Root {
			args = append(args, "obj")
		}
	}

	for _, arg := range f.Args {
		args = append(args, "args["+strconv.Quote(arg.GQLName)+"].("+arg.Signature()+")")
	}

	return strings.Join(args, ", ")
}

// should be in the template, but its recursive and has a bunch of args
func (f *Field) WriteJson() string {
	return f.doWriteJson("res", f.Type.Modifiers, false, 1)
}

func (f *Field) doWriteJson(val string, remainingMods []string, isPtr bool, depth int) string {
	switch {
	case len(remainingMods) > 0 && remainingMods[0] == modPtr:
		return fmt.Sprintf("if %s == nil { return graphql.Null }\n%s", val, f.doWriteJson(val, remainingMods[1:], true, depth+1))

	case len(remainingMods) > 0 && remainingMods[0] == modList:
		if isPtr {
			val = "*" + val
		}
		var arr = "arr" + strconv.Itoa(depth)
		var index = "idx" + strconv.Itoa(depth)

		return tpl(`{{.arr}} := graphql.Array{}
			for {{.index}} := range {{.val}} {
				{{.arr}} = append({{.arr}}, func() graphql.Marshaler {
					rctx := graphql.GetResolverContext(ctx)
					rctx.PushIndex({{.index}})
					defer rctx.Pop()
					{{ .next }} 
				}())
			}
			return {{.arr}}`, map[string]interface{}{
			"val":   val,
			"arr":   arr,
			"index": index,
			"next":  f.doWriteJson(val+"["+index+"]", remainingMods[1:], false, depth+1),
		})

	case f.IsScalar:
		if isPtr {
			val = "*" + val
		}
		return f.Marshal(val)

	default:
		if !isPtr {
			val = "&" + val
		}
		return fmt.Sprintf("return ec._%s(ctx, field.Selections, %s)", f.GQLType, val)
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
	err := template.Must(template.New("inline").Parse(tpl)).Execute(b, vars)
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
