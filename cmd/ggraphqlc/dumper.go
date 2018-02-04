package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
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
	wr.writeResolver()
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
		w.line("%s %s", local, strconv.Quote(pkg))
	}
	w.end(")")
	w.lf()
}

func (w *writer) writeInterface() {
	w.begin("type Resolvers interface {")
	for _, o := range w.Objects {
		for _, f := range o.Fields {
			if f.Bind != "" {
				continue
			}

			w.emitIndent()
			w.emit("%s_%s(", o.Name, f.Name)

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

func (w *writer) writeResolver() {
	w.begin("func NewResolver(r Resolvers) exec.Root {")
	w.line("return &resolvers{r}")
	w.end("}")
	w.lf()

	w.begin("type resolvers struct {")
	w.line("resolvers Resolvers")
	w.end("}")
	w.lf()

	for _, object := range w.Objects {
		w.writeObjectResolver(object)
	}
}

func (w *writer) writeObjectResolver(object object) {
	objectName := "it"
	if object.Type.Name != "interface{}" {
		objectName = "object"
	}

	w.begin("func (r *resolvers) %s(ec *exec.ExecutionContext, %s interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable {", object.Type.GraphQLName, objectName)
	if object.Type.Name != "interface{}" {
		w.line("it := object.(*%s)", object.Type.Local())
	}

	w.line("switch field {")

	for _, field := range object.Fields {
		w.begin("case %s:", strconv.Quote(field.Name))

		if field.Bind != "" {
			w.writeFieldBind(field)
		} else {
			w.writeFieldResolver(object, field)
		}

		w.end("")
	}

	w.line("}")
	w.line(`panic("unknown field " + field)`)
	w.end("}")
	w.lf()
}

func (w *writer) writeFieldBind(field Field) {
	w.line("return jsonw.%s(it.%s)", field.Type.GraphQLName, field.Bind)
}

func (w *writer) writeFieldResolver(object object, field Field) {
	call := fmt.Sprintf("result, err := r.resolvers.%s_%s", object.Name, field.Name)
	if len(field.Args) == 0 {
		w.line(call + "()")
	} else {
		w.begin(call + "(")
		for _, arg := range field.Args {
			w.line("arguments[%s].(%s),", strconv.Quote(arg.Name), arg.Type.Local())
		}
		w.end(")")
	}

	w.line("if err != nil {")
	w.line("	ec.Error(err)")
	w.line("	return jsonw.Null")
	w.line("}")

	result := "result"
	if !strings.HasPrefix(field.Type.Prefix, "*") {
		result = "&result"
	}
	w.line("return ec.ExecuteSelectionSet(sels, r.%s, %s)", field.Type.Name, result)
}

func (w *writer) writeSchema() {
	w.line("var Schema = schema.MustParse(%s)", strconv.Quote(w.schemaRaw))
}
