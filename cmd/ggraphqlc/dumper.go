package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

type writer struct {
	extractor
	out    io.Writer
	indent int
}

func write(extractor extractor, out io.Writer) {
	wr := writer{extractor, out, 0}

	wr.writePackage()
	wr.writeImports()
	wr.writeInterface()
	for _, object := range wr.Objects {
		wr.writeObjectResolver(object)
	}
	wr.writeSchema()
}

func (w *writer) emit(format string, args ...interface{}) {
	io.WriteString(w.out, fmt.Sprintf(format, args...))
}

func (w *writer) emitIndent() {
	io.WriteString(w.out, strings.Repeat("	", w.indent))
}

func (w *writer) begin(format string, args ...interface{}) {
	w.emitIndent()
	w.emit(format, args...)
	w.lf()
	w.indent++
}

func (w *writer) end(format string, args ...interface{}) {
	w.indent--
	w.emitIndent()
	w.emit(format, args...)
	w.lf()
}

func (w *writer) line(format string, args ...interface{}) {
	w.emitIndent()
	w.emit(format, args...)
	w.lf()
}

func (w *writer) lf() {
	w.out.Write([]byte("\n"))
}

func (w *writer) writePackage() {
	w.line("package %s", w.PackageName)
	w.lf()
}

func (w *writer) writeImports() {
	w.begin("import (")
	for local, pkg := range w.Imports {
		if local == filepath.Base(pkg) {
			w.line(strconv.Quote(pkg))
		} else {
			w.line("%s %s", local, strconv.Quote(pkg))
		}

	}
	w.end(")")
	w.lf()
}

func (w *writer) writeInterface() {
	w.begin("type Resolvers interface {")
	for _, o := range w.Objects {
		for _, f := range o.Fields {
			if f.VarName != "" || f.MethodName != "" {
				continue
			}

			w.emitIndent()
			w.emit("%s_%s(", o.Name, f.GraphQLName)

			first := true
			for _, arg := range f.Args {
				if !first {
					w.emit(",")
				}
				first = false
				w.emit("%s %s", arg.Name, arg.Type.Local())
			}
			w.emit(") (%s, error)", f.Type.Local())
			w.lf()
		}
	}
	w.end("}")
	w.lf()
}

func (w *writer) writeObjectResolver(object object) {
	objectName := "it"
	if object.Type.Name != "interface{}" {
		objectName = "object"
	}

	w.line("type %sType struct {}", lcFirst(object.Type.GraphQLName))
	w.lf()

	w.begin("func (%sType) accepts(name string) bool {", lcFirst(object.Type.GraphQLName))
	w.line("return true")
	w.end("}")
	w.lf()

	w.begin("func (%sType) resolve(ec *executionContext, %s interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {", lcFirst(object.Type.GraphQLName), objectName)
	if object.Type.Name != "interface{}" {
		w.line("it := object.(*%s)", object.Type.Local())
	}
	w.line("if it == nil {")
	w.line("	return jsonw.Null")
	w.line("}")

	w.line("switch field {")

	for _, field := range object.Fields {
		w.begin("case %s:", strconv.Quote(field.GraphQLName))

		if field.VarName != "" {
			w.writeVarResolver(field)
		} else {
			w.writeMethodResolver(object, field)
		}

		w.end("")
	}

	w.line("}")
	w.line(`panic("unknown field " + field)`)
	w.end("}")
	w.lf()
}

func (w *writer) writeMethodResolver(object object, field Field) {
	var methodName string
	if field.MethodName != "" {
		methodName = field.MethodName
	} else {
		methodName = fmt.Sprintf("ec.resolvers.%s_%s", object.Name, field.GraphQLName)
	}

	if field.NoErr {
		w.emitIndent()
		w.emit("res := %s", methodName)
		w.writeFuncArgs(field)
	} else {
		w.emitIndent()
		w.emit("res, err := %s", methodName)
		w.writeFuncArgs(field)
		w.line("if err != nil {")
		w.line("	ec.Error(err)")
		w.line("	return jsonw.Null")
		w.line("}")
	}

	w.writeJsonType("json", field.Type, "res")

	w.line("return json")
}

func (w *writer) writeVarResolver(field Field) {
	w.writeJsonType("res", field.Type, field.VarName)
	w.line("return res")
}

func (w *writer) writeFuncArgs(field Field) {
	if len(field.Args) == 0 {
		w.emit("()")
		w.lf()
	} else {
		w.indent++
		w.emit("(")
		w.lf()
		for _, arg := range field.Args {
			w.line("arguments[%s].(%s),", strconv.Quote(arg.Name), arg.Type.Local())
		}
		w.end(")")
	}
}

func (w *writer) writeJsonType(result string, t Type, val string) {
	w.doWriteJsonType(result, t, val, t.Modifiers, false)
}

func (w *writer) doWriteJsonType(result string, t Type, val string, remainingMods []string, isPtr bool) {
	for i := 0; i < len(remainingMods); i++ {
		switch remainingMods[i] {
		case modPtr:
			w.line("var %s jsonw.Encodable = jsonw.Null", result)
			w.begin("if %s != nil {", val)
			w.doWriteJsonType(result+"1", t, val, remainingMods[i+1:], true)
			w.line("%s = %s", result, result+"1")
			w.end("}")
			return
		case modList:
			if isPtr {
				val = "*" + val
			}
			w.line("%s := jsonw.Array{}", result)
			w.begin("for _, val := range %s {", val)

			w.doWriteJsonType(result+"1", t, "val", remainingMods[i+1:], false)
			w.line("%s = append(%s, %s)", result, result, result+"1")
			w.end("}")
			return
		}
	}

	if t.Basic {
		if isPtr {
			val = "*" + val
		}
		w.line("%s := jsonw.%s(%s)", result, ucFirst(t.Name), val)
	} else {
		if !isPtr {
			val = "&" + val
		}
		w.line("%s := ec.executeSelectionSet(sels, %sType{}, %s)", result, lcFirst(t.GraphQLName), val)
	}
}

func ucFirst(s string) string {
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func lcFirst(s string) string {
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func (w *writer) writeSchema() {
	w.line("var parsedSchema = schema.MustParse(%s)", strconv.Quote(w.schemaRaw))
}
