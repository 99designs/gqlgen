package main

import (
	"fmt"
	"strconv"
	"strings"
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

func (t kind) FullName() string {
	if t.ImportedAs == "" {
		return t.Name
	}
	return t.ImportedAs + "." + t.Name
}

func (t kind) WriteJson(val string) string {
	return t.doWriteJson(val, t.Modifiers, false)
}

// should be in the template, but its recursive and has a bunch fo args
func (t kind) doWriteJson(val string, remainingMods []string, isPtr bool) string {
	switch {
	case len(remainingMods) > 0 && remainingMods[0] == modPtr:
		return fmt.Sprintf(
			"if %s == nil { ec.json.Null() } else { %s } ",
			val, t.doWriteJson(val, remainingMods[1:], true),
		)

	case len(remainingMods) > 0 && remainingMods[0] == modList:
		if isPtr {
			val = "*" + val
		}

		return strings.Join([]string{
			"ec.json.BeginArray()",
			fmt.Sprintf("for _, val := range %s {", val),
			t.doWriteJson("val", remainingMods[1:], false),
			"}",
			"ec.json.EndArray()",
		}, "\n")

	case t.Scalar:
		if isPtr {
			val = "*" + val
		}
		return fmt.Sprintf("ec.json.%s(%s)", ucFirst(t.Name), val)

	default:
		needPtr := len(t.Implementors) == 0
		if needPtr && !isPtr {
			val = "&" + val
		}
		if !needPtr && isPtr {
			val = "*" + val
		}
		return fmt.Sprintf("ec._%s(field.Selections, %s)", lcFirst(t.GraphQLName), val)
	}
}

type object struct {
	Name      string
	Fields    []Field
	Type      kind
	satisfies []string
	Root      bool
}

type Field struct {
	GraphQLName string
	MethodName  string
	VarName     string
	Type        kind
	Args        []FieldArgument
	NoErr       bool
}

func (f *Field) IsResolver() bool {
	return f.MethodName == "" && f.VarName == ""
}

func (f *Field) ResolverDeclaration(o object) string {
	if !f.IsResolver() {
		return ""
	}
	res := fmt.Sprintf("%s_%s(ctx context.Context", o.Name, f.GraphQLName)

	if !o.Root {
		res += fmt.Sprintf(", it *%s", o.Type.Local())
	}
	for _, arg := range f.Args {
		res += fmt.Sprintf(", %s %s", arg.Name, arg.Type.Local())
	}

	res += fmt.Sprintf(") (%s, error)", f.Type.Local())
	return res
}

func (f *Field) CallArgs(object object) string {
	var args []string

	if f.MethodName == "" {
		args = append(args, "ec.ctx")

		if !object.Root {
			args = append(args, "it")
		}
	}

	for i := range f.Args {
		args = append(args, "arg"+strconv.Itoa(i))
	}

	return strings.Join(args, ", ")
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
			return &e.Objects[i]
		}
	}
	return nil
}

type FieldArgument struct {
	Name string
	Type kind
}
