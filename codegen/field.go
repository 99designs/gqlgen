package codegen

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/ast"
)

type Field struct {
	*TypeReference
	GQLName          string           // The name of the field in graphql
	GoFieldType      GoFieldType      // The field type in go, if any
	GoReceiverName   string           // The name of method & var receiver in go, if any
	GoFieldName      string           // The name of the method or var in go, if any
	IsResolver       bool             // Does this field need a resolver
	Args             []*FieldArgument // A list of arguments to be passed to this field
	MethodHasContext bool             // If this is bound to a go method, does the method also take a context
	NoErr            bool             // If this is bound to a go method, does that method have an error as the second argument
	Object           *Object          // A link back to the parent object
	Default          interface{}      // The default value
	Directives       []*Directive
}

func (f *Field) HasDirectives() bool {
	return len(f.Directives) > 0
}

func (f *Field) IsReserved() bool {
	return strings.HasPrefix(f.GQLName, "__")
}

func (f *Field) IsMethod() bool {
	return f.GoFieldType == GoFieldMethod
}

func (f *Field) IsVariable() bool {
	return f.GoFieldType == GoFieldVariable
}

func (f *Field) IsConcurrent() bool {
	if f.Object.DisableConcurrency {
		return false
	}
	return f.MethodHasContext || f.IsResolver
}

func (f *Field) GoNameUnexported() string {
	return templates.ToGoPrivate(f.GQLName)
}

func (f *Field) ShortInvocation() string {
	return fmt.Sprintf("%s().%s(%s)", f.Object.Definition.GQLDefinition.Name, f.GoFieldName, f.CallArgs())
}

func (f *Field) ArgsFunc() string {
	if len(f.Args) == 0 {
		return ""
	}

	return "field_" + f.Object.Definition.GQLDefinition.Name + "_" + f.GQLName + "_args"
}

func (f *Field) ResolverType() string {
	if !f.IsResolver {
		return ""
	}

	return fmt.Sprintf("%s().%s(%s)", f.Object.Definition.GQLDefinition.Name, f.GoFieldName, f.CallArgs())
}

func (f *Field) ShortResolverDeclaration() string {
	if !f.IsResolver {
		return ""
	}
	res := fmt.Sprintf("%s(ctx context.Context", f.GoFieldName)

	if !f.Object.Root {
		res += fmt.Sprintf(", obj *%s", templates.CurrentImports.LookupType(f.Object.Definition.GoType))
	}
	for _, arg := range f.Args {
		res += fmt.Sprintf(", %s %s", arg.GoVarName, templates.CurrentImports.LookupType(arg.GoType))
	}

	result := templates.CurrentImports.LookupType(f.GoType)
	if f.Object.Stream {
		result = "<-chan " + result
	}

	res += fmt.Sprintf(") (%s, error)", result)
	return res
}

func (f *Field) ComplexitySignature() string {
	res := fmt.Sprintf("func(childComplexity int")
	for _, arg := range f.Args {
		res += fmt.Sprintf(", %s %s", arg.GoVarName, templates.CurrentImports.LookupType(arg.GoType))
	}
	res += ") int"
	return res
}

func (f *Field) ComplexityArgs() string {
	var args []string
	for _, arg := range f.Args {
		args = append(args, "args["+strconv.Quote(arg.GQLName)+"].("+templates.CurrentImports.LookupType(arg.GoType)+")")
	}

	return strings.Join(args, ", ")
}

func (f *Field) CallArgs() string {
	var args []string

	if f.IsResolver {
		args = append(args, "rctx")

		if !f.Object.Root {
			args = append(args, "obj")
		}
	} else {
		if f.MethodHasContext {
			args = append(args, "ctx")
		}
	}

	for _, arg := range f.Args {
		args = append(args, "args["+strconv.Quote(arg.GQLName)+"].("+templates.CurrentImports.LookupType(arg.GoType)+")")
	}

	return strings.Join(args, ", ")
}

// should be in the template, but its recursive and has a bunch of args
func (f *Field) WriteJson() string {
	return f.doWriteJson("res", f.GoType, f.ASTType, false, 1)
}

func (f *Field) doWriteJson(val string, destType types.Type, astType *ast.Type, isPtr bool, depth int) string {
	switch destType := destType.(type) {
	case *types.Pointer:
		return tpl(`
			if {{.val}} == nil {
				{{- if .nonNull }}
					if !ec.HasError(rctx) {
						ec.Errorf(ctx, "must not be null")
					}
				{{- end }}
				return graphql.Null
			}
			{{.next }}`, map[string]interface{}{
			"val":     val,
			"nonNull": astType.NonNull,
			"next":    f.doWriteJson(val, destType.Elem(), astType, true, depth+1),
		})

	case *types.Slice:
		if isPtr {
			val = "*" + val
		}
		var arr = "arr" + strconv.Itoa(depth)
		var index = "idx" + strconv.Itoa(depth)
		var usePtr bool
		if !isPtr {
			switch destType.Elem().(type) {
			case *types.Pointer, *types.Array:
			default:
				usePtr = true
			}
		}

		return tpl(`
			{{.arr}} := make(graphql.Array, len({{.val}}))
			{{ if and .top (not .isScalar) }} var wg sync.WaitGroup {{ end }}
			{{ if not .isScalar }}
				isLen1 := len({{.val}}) == 1
				if !isLen1 {
					wg.Add(len({{.val}}))
				}
			{{ end }}
			for {{.index}} := range {{.val}} {
				{{- if not .isScalar }}
					{{.index}} := {{.index}}
					rctx := &graphql.ResolverContext{
						Index: &{{.index}},
						Result: {{ if .usePtr }}&{{end}}{{.val}}[{{.index}}],
					}
					ctx := graphql.WithResolverContext(ctx, rctx)
					f := func({{.index}} int) {
						if !isLen1 {
							defer wg.Done()
						}
						{{.arr}}[{{.index}}] = func() graphql.Marshaler {
							{{ .next }}
						}()
					}
					if isLen1 {
						f({{.index}})
					} else {
						go f({{.index}})
					}
				{{ else }}
					{{.arr}}[{{.index}}] = func() graphql.Marshaler {
						{{ .next }}
					}()
				{{- end}}
			}
			{{ if and .top (not .isScalar) }} wg.Wait() {{ end }}
			return {{.arr}}`, map[string]interface{}{
			"val":      val,
			"arr":      arr,
			"index":    index,
			"top":      depth == 1,
			"arrayLen": len(val),
			"isScalar": f.Definition.GQLDefinition.Kind == ast.Scalar || f.Definition.GQLDefinition.Kind == ast.Enum,
			"usePtr":   usePtr,
			"next":     f.doWriteJson(val+"["+index+"]", destType.Elem(), astType.Elem, false, depth+1),
		})

	default:
		if f.Definition.GQLDefinition.Kind == ast.Scalar || f.Definition.GQLDefinition.Kind == ast.Enum {
			if isPtr {
				val = "*" + val
			}
			return f.Marshal(val)
		}

		if !isPtr {
			val = "&" + val
		}
		return tpl(`
			return ec._{{.type}}(ctx, field.Selections, {{.val}})`, map[string]interface{}{
			"type": f.Definition.GQLDefinition.Name,
			"val":  val,
		})
	}
}
