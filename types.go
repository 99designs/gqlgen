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

// should be in the template, but its recursive and has a bunch fo args
func (f *Field) WriteJson() string {
	return f.doWriteJson("t."+ucFirst(f.GraphQLName), f.Type.Modifiers, false)
}

func (f *Field) doWriteJson(val string, remainingMods []string, isPtr bool) string {
	switch {
	case len(remainingMods) > 0 && remainingMods[0] == modPtr:
		return fmt.Sprintf(
			"if %s == nil { w.Null() } else { %s } ",
			val, f.doWriteJson(val, remainingMods[1:], true),
		)

	case len(remainingMods) > 0 && remainingMods[0] == modList:
		if isPtr {
			val = "*" + val
		}

		return strings.Join([]string{
			"w.BeginArray()",
			fmt.Sprintf("for _, val := range %s {", val),
			f.doWriteJson("val", remainingMods[1:], false),
			"}",
			"w.EndArray()",
		}, "\n")

	case f.Type.Scalar:
		if isPtr {
			val = "*" + val
		}
		return fmt.Sprintf("w.%s(%s)", ucFirst(f.Type.Name), val)

	default:
		return fmt.Sprintf("%s.WriteJson(w)", val)
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
			return &e.Objects[i]
		}
	}
	return nil
}

type FieldArgument struct {
	Name string
	Type kind
}
