// Wrapper around x/tools/imports that only removes imports, never adds new ones.

package imports

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"sync"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"

	"github.com/99designs/gqlgen/internal/code"
)

// bufPool reuses buffers across Prune calls to reduce allocations
var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

type visitFn func(node ast.Node)

func (fn visitFn) Visit(node ast.Node) ast.Visitor {
	fn(node)
	return fn
}

// Prune removes any unused imports.
// If skipImportGrouping is true, uses format.Source (faster, no import grouping).
// If false, uses imports.Process (slower, groups imports by stdlib/external/internal).
func Prune(filename string, src []byte, packages *code.Packages, skipImportGrouping bool) ([]byte, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, err
	}

	unused := getUnusedImports(file, packages)
	for ipath, name := range unused {
		astutil.DeleteNamedImport(fset, file, name, ipath)
	}
	printConfig := &printer.Config{Mode: printer.TabIndent, Tabwidth: 8}

	// Reuse buffer from pool to reduce allocations
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if err := printConfig.Fprint(buf, fset, file); err != nil {
		return nil, err
	}

	if skipImportGrouping {
		// format.Source is faster - we already removed unused imports above
		return format.Source(buf.Bytes())
	}

	// imports.Process groups imports by stdlib/external/internal but is slower
	return imports.Process(
		filename,
		buf.Bytes(),
		&imports.Options{FormatOnly: true, Comments: true, TabIndent: true, TabWidth: 8},
	)
}

func getUnusedImports(file ast.Node, packages *code.Packages) map[string]string {
	imported := map[string]*ast.ImportSpec{}
	used := map[string]bool{}

	ast.Walk(visitFn(func(node ast.Node) {
		if node == nil {
			return
		}
		switch v := node.(type) {
		case *ast.ImportSpec:
			if v.Name != nil {
				imported[v.Name.Name] = v
				break
			}
			ipath := strings.Trim(v.Path.Value, `"`)
			if ipath == "C" {
				break
			}

			local := packages.NameForPackage(ipath)

			imported[local] = v
		case *ast.SelectorExpr:
			xident, ok := v.X.(*ast.Ident)
			if !ok {
				break
			}
			if xident.Obj != nil {
				// if the parser can resolve it, it's not a package ref
				break
			}
			used[xident.Name] = true
		}
	}), file)

	for pkg := range used {
		delete(imported, pkg)
	}

	unusedImport := map[string]string{}
	for pkg, is := range imported {
		if !used[pkg] && pkg != "_" && pkg != "." {
			name := ""
			if is.Name != nil {
				name = is.Name.Name
			}
			unusedImport[strings.Trim(is.Path.Value, `"`)] = name
		}
	}

	return unusedImport
}
