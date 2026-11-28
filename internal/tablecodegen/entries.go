package tablecodegen

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/templates"
)

// TableCallArgs is CallArgs for the Call function of an tableimpl.DirectiveDef of the directive
// dir, which receives the object as directiveObj and the arguments as directiveArgs: names
// that a package of the Go types of the arguments is unlikely to have, which they would
// hide. An argument of the empty interface type is passed as it is, since it is nil when
// it is absent or null. An argument of another interface type is nil in the map when it is
// absent, as the functions mode passes nil then, so it is read without a type assertion
// that would panic on nil.
func (d *Data) TableCallArgs(dir *codegen.Directive) string {
	args := []string{"ctx", "directiveObj", "n"}
	for _, arg := range dir.Args {
		goType := templates.CurrentImports.LookupType(arg.TypeReference.GO)
		switch iface, ok := types.Unalias(arg.TypeReference.GO).(*types.Interface); {
		case ok && iface.Empty():
			args = append(args, fmt.Sprintf("directiveArgs[%q]", arg.Name))
		case types.IsInterface(arg.TypeReference.GO):
			args = append(args, fmt.Sprintf(
				"func() (v %s) { v, _ = directiveArgs[%q].(%s); return v }()",
				goType, arg.Name, goType,
			))
		default:
			args = append(args, fmt.Sprintf("directiveArgs[%q].(%s)", arg.Name, goType))
		}
	}
	return strings.Join(args, ", ")
}

// TableDirectiveVar returns the name of the field of the tables that holds the
// tableimpl.DirectiveDef generated for the directive dir.
func (d *Data) TableDirectiveVar(dir *codegen.Directive) string {
	return "directive" + dir.CallName()
}

// TableImplemented returns the implemented function of the tableimpl.DirectiveDef generated for
// the directive dir: nil for a built-in directive, which always has an implementation.
func (d *Data) TableImplemented(dir *codegen.Directive) string {
	if dir.IsBuiltIn() {
		return "nil"
	}
	return "func(ec *executionContext) bool { return " + dir.CallPath() + " != nil }"
}

// TableBatch reports whether the field f is resolved by the function of its batch
// resolver, which the functions mode generates for the fields of the objects that are not
// roots. The table of the object calls it in place of the Resolve of the object, with the
// field as the request selects it. A field that returns a root operation type resolves to
// an empty value of the type instead, as in the functions mode.
func (d *Data) TableBatch(f *codegen.Field) bool {
	return f.IsBatch() && !f.Object.Root && !f.TypeReference.IsRoot
}

// tableObj is the expression of the object of a field: obj, which the tableimpl.Resolve of the
// object asserts to its Go type once, before it switches on the field.
const tableObj = "obj"

// tableGetExpr returns the expression that reads the field f from obj, the object with its
// Go type, for fields bound to a struct field, or to a method that takes no context or
// arguments and returns no error. It returns "" for other fields, and for those whose Go
// type the marshaler of the field does not take, such as an Omittable that marshals
// itself: the functions mode returns the value as any, and the field checks its type when
// it resolves.
func (d *Data) tableGetExpr(f *codegen.Field) string {
	if f.IsBatch() || f.HasHaser || f.VOkFunc || f.IsResolver || f.GoReceiverName != "obj" {
		return ""
	}
	if goType := d.shared.goTypes[f]; goType != nil && f.TypeReference != nil &&
		!types.AssignableTo(goType, f.TypeReference.GO) {
		return ""
	}
	switch {
	case f.IsVariable():
		return tableObj + "." + f.GoFieldName
	case f.IsMethod() && f.NoErr && !f.MethodHasContext && len(f.Args) == 0:
		return tableObj + "." + f.GoFieldName + "()"
	}
	return ""
}

// tableResolveBody returns the body of the case of the field f in the tableimpl.Resolve of its
// object, written as a single statement that reads the object from obj and the arguments
// from fieldArgs. It returns "" when the field needs more than one statement; the template
// then writes the body the functions mode uses, which reads the arguments from the field
// context: the runtime passes fieldArgs from the field context, so they are the same.
func tableResolveBody(f *codegen.Field) string {
	if f.IsBatch() || f.HasHaser || f.VOkFunc {
		return ""
	}
	recv := f.GoReceiverName
	if recv == "obj" {
		recv = tableObj
	}
	var body string
	switch {
	case f.IsResolver:
		args := []string{"ctx"}
		if !f.Object.Root {
			args = append(args, tableObj)
		}
		args = append(args, callArgExpressions(f, "", "fieldArgs")...)
		caser := cases.Title(language.English, cases.NoLower)
		body = "return ec.Resolvers." + caser.String(
			f.Object.Name,
		) + "()." + f.GoFieldName + "(" + strings.Join(args, ", ") + ")"
	case f.IsMethod():
		call := recv + "." + f.GoFieldName + "(" + callArgs(f, "fieldArgs") + ")"
		if f.NoErr {
			body = "return " + call + ", nil"
		} else {
			body = "return " + call
		}
	case f.IsVariable():
		body = "return " + recv + "." + f.GoFieldName + ", nil"
	default:
		return ""
	}
	if strings.Contains(body, "\n") {
		return ""
	}
	return body
}

// TableCaseBody returns the body of the case of the field f in the tableimpl.Resolve of its
// object. A field of a root operation type resolves to a fresh value of the type. A field
// read from its object declares the value with the Go type of the field's marshaler before
// it returns it as any, so that the compiler checks that the field has the type the
// marshaler marshals, as the typed call of the functions mode does. It returns "" when the
// field needs the body that the functions mode uses, which the template then writes.
func (d *Data) TableCaseBody(f *codegen.Field) string {
	if f.TypeReference.IsRoot {
		if f.TypeReference.IsPtr() {
			return "return &" + tableType(f.TypeReference.Elem().GO) + "{}, nil"
		}
		return "return " + tableType(f.TypeReference.GO) + "{}, nil"
	}
	if get := d.tableGetExpr(f); get != "" {
		return "var v " + tableType(f.TypeReference.GO) + " = " + get + "\nreturn v, nil"
	}
	return tableResolveBody(f)
}

// TableSetStmt returns the statement of the case of the input field f in the Set of its
// input, which stores v into the input: m, its map, or p, the pointer to its Go type. It
// asserts v, which the runtime passes as any, to the Go type of the field's unmarshaler, so
// that the compiler checks that the field has the type the unmarshaler unmarshals, as the
// typed call of the functions mode does. A nil stores the zero value.
func (d *Data) TableSetStmt(f *codegen.Field) string {
	goType := tableType(f.TypeReference.GO)
	switch {
	case f.Object.IsMap():
		return fmt.Sprintf("m[%q], _ = v.(%s)", f.Name, goType)
	case f.TypeReference.IsOmittable:
		return fmt.Sprintf("x, _ := v.(%s)\np.%s = graphql.OmittableOf(x)", goType, f.GoFieldName)
	default:
		return fmt.Sprintf("p.%s, _ = v.(%s)", f.GoFieldName, goType)
	}
}

// TableInputInvocation returns the resolver call that sets the input field f from
// fieldValue, where the input being unmarshaled is the pointer inputValue, of type any.
// The functions mode calls it with data, which table mode does not name its variable, as
// it refers to the Go type of the input afterwards, which a package called data would hide.
func (d *Data) TableInputInvocation(f *codegen.Field) string {
	caser := cases.Title(language.English, cases.NoLower)
	return fmt.Sprintf(
		"%s().%s(ctx, inputValue.(*%s), fieldValue)",
		caser.String(f.Object.Name),
		f.GoFieldName,
		templates.CurrentImports.LookupType(f.Object.Type),
	)
}

// TableObjectVar returns the name of the field of the tables that holds the table
// generated for the object or input o.
func (d *Data) TableObjectVar(o *codegen.Object) string {
	if o.Kind == ast.InputObject {
		return "input" + o.Name
	}
	return "object" + o.Name
}

// TableFunc returns the name of the function that table mode generates for the object or
// input o: the tableimpl.Resolve that resolves the fields of an object by their index, or the
// Set that stores the fields of an input.
func (d *Data) TableFunc(o *codegen.Object) string {
	if name := d.shared.funcs[o]; name != "" {
		return name
	}
	return "_" + tableFuncName(o)
}

// tableFuncName is the name of the function of TableFunc without its leading underscore.
func tableFuncName(o *codegen.Object) string {
	if o.Kind == ast.InputObject {
		return o.Name + "_set"
	}
	return o.Name + "_resolve"
}

// tableFuncNames sets in funcs the names of the functions that table mode generates for
// objects: _User_resolve and _Pair_set, with underscores added while a type of schema has
// the name, whose function would take it, as the function of a union called User_resolve
// does.
func tableFuncNames(schema *ast.Schema, objects codegen.Objects, funcs map[*codegen.Object]string) {
	for _, o := range objects {
		name := tableFuncName(o)
		for schema.Types[name] != nil {
			name += "_"
		}
		funcs[o] = "_" + name
	}
}

// TableReadsObject reports whether a case of the tableimpl.Resolve of the object o reads the
// object: that of every field but those that return a root operation type, which resolve
// to an empty value of the type, and those of batch resolvers, which have no case. The
// Resolve of an object that has no other fields does not assert the object to its Go type.
func (d *Data) TableReadsObject(o *codegen.Object) bool {
	if o.Root {
		return false
	}
	for _, f := range o.Fields {
		if !f.TypeReference.IsRoot && !d.TableBatch(f) {
			return true
		}
	}
	return false
}

// TableSetsFields reports whether the Set of the input o stores any field: those that a
// resolver sets are stored by the resolver instead.
func (d *Data) TableSetsFields(o *codegen.Object) bool {
	for _, f := range o.Fields {
		if !f.IsResolver {
			return true
		}
	}
	return false
}
