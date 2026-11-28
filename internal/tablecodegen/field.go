package tablecodegen

import (
	"fmt"
	"go/types"
	"slices"
	"strconv"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/templates"
)

// federationRequiresArgName is the name of the argument of the fields that the federation
// plugin adds for @requires: see codegen.federationRequiresArgName.
const federationRequiresArgName = "_federationRequires"

// callArgs is codegen.Field.CallArgs, reading the arguments of the field from the map
// called argsVar.
func callArgs(f *codegen.Field, argsVar string) string {
	args := make([]string, 0, len(f.Args)+2)

	if f.IsResolver {
		args = append(args, "ctx")

		if !f.Object.Root {
			args = append(args, "obj")
		}
	} else if f.MethodHasContext {
		args = append(args, "ctx")
	}

	args = append(args, callArgExpressions(f, "", argsVar)...)
	return strings.Join(args, ", ")
}

// fieldArgExpression is the method of codegen.Field, which returns the expression that
// reads arg from the map called argsVar.
func fieldArgExpression(
	arg *codegen.FieldArgument,
	federationRequiresReplacement string,
	argsVar string,
) string {
	if arg.Name == federationRequiresArgName && federationRequiresReplacement != "" {
		return federationRequiresReplacement
	}

	tmp := argsVar + "[" + strconv.Quote(
		arg.Name,
	) + "].(" + templates.CurrentImports.LookupType(
		arg.TypeReference.GO,
	) + ")"

	if iface, ok := types.Unalias(arg.TypeReference.GO).(*types.Interface); ok && iface.Empty() {
		tmp = fmt.Sprintf(`
				func () any {
					if %[1]s["%[2]s"] == nil {
						return nil
					}
					return %[1]s["%[2]s"].(any)
				}()`, argsVar, arg.Name,
		)
	}
	return tmp
}

// callArgExpressions is the method of codegen.Field, which returns the expressions of the
// arguments of the field, read from the map called argsVar.
func callArgExpressions(f *codegen.Field, federationRequiresReplacement, argsVar string) []string {
	args := make([]string, 0, len(f.Args))
	var inlineInfo *codegen.InlineArgsInfo
	if f.Object != nil && f.Object.Definition != nil {
		inlineInfo = codegen.GetInlineArgsMetadata(f.Object.Name, f.Name)
	}
	if inlineInfo != nil {
		isMap := strings.Contains(inlineInfo.GoType, "map[")

		var entries []string
		for _, argName := range inlineInfo.ExpandedArgs {
			var argRef *codegen.FieldArgument
			for _, arg := range f.Args {
				if arg.Name == argName {
					argRef = arg
					break
				}
			}
			if argRef != nil {
				goType := templates.CurrentImports.LookupType(argRef.TypeReference.GO)
				var entry string
				if isMap {
					entry = fmt.Sprintf("%q: %s[%q].(%s)", argName, argsVar, argName, goType)
				} else {
					fieldName := templates.ToGo(argName)
					entry = fmt.Sprintf("%s: %s[%q].(%s)", fieldName, argsVar, argName, goType)
				}
				entries = append(entries, entry)
			}
		}

		goType := formatGoType(inlineInfo.GoType)
		bundled := fmt.Sprintf("%s{\n\t\t%s,\n\t}", goType, strings.Join(entries, ",\n\t\t"))
		args = append(args, bundled)

		for _, arg := range f.Args {
			if !slices.Contains(inlineInfo.ExpandedArgs, arg.Name) {
				args = append(
					args,
					fieldArgExpression(arg, federationRequiresReplacement, argsVar),
				)
			}
		}
	} else {
		for _, arg := range f.Args {
			args = append(args, fieldArgExpression(arg, federationRequiresReplacement, argsVar))
		}
	}

	return args
}

func formatGoType(goType string) string {
	if strings.Contains(goType, "/") {
		lastDot := strings.LastIndex(goType, ".")
		if lastDot == -1 {
			return goType
		}

		packagePath := goType[:lastDot]
		typeName := goType[lastDot+1:]

		alias := templates.CurrentImports.Lookup(packagePath)
		if alias == "" {
			return typeName
		}

		return alias + "." + typeName
	}

	return goType
}
