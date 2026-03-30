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

// bufPool reuses buffers across Prune calls to reduce allocations.
var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// getBuffer returns a buffer and a release function.
// If usePool is true, the buffer is obtained from the pool and release returns it.
// If usePool is false, a new buffer is allocated and release is a no-op.
func getBuffer(usePool bool) (*bytes.Buffer, func()) {
	if usePool {
		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()
		return buf, func() { bufPool.Put(buf) }
	}
	return new(bytes.Buffer), func() {}
}

// defaultTabWidth is the standard Go tab width used by gofmt.
const defaultTabWidth = 8

type visitFn func(node ast.Node)

func (fn visitFn) Visit(node ast.Node) ast.Visitor {
	fn(node)
	return fn
}

// PruneOptions configures the behavior of the Prune function.
type PruneOptions struct {
	// SkipImportGrouping uses format.Source instead of imports.Process.
	// Faster but doesn't group imports by stdlib/external/internal.
	SkipImportGrouping bool
	// UseBufferPooling reuses buffers via sync.Pool to reduce GC pressure.
	UseBufferPooling bool
}

// Prune removes any unused imports from Go source code.
func Prune(
	filename string,
	src []byte,
	packages *code.Packages,
	opts PruneOptions,
) ([]byte, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, err
	}

	unused := getUnusedImports(file, packages)
	for ipath, name := range unused {
		astutil.DeleteNamedImport(fset, file, name, ipath)
	}

	buf, release := getBuffer(opts.UseBufferPooling)
	defer release()

	printConfig := &printer.Config{Mode: printer.TabIndent, Tabwidth: defaultTabWidth}
	if err := printConfig.Fprint(buf, fset, file); err != nil {
		return nil, err
	}

	if opts.SkipImportGrouping {
		return formatSourceFast(buf.Bytes())
	}
	return formatSourceWithGrouping(filename, buf.Bytes())
}

// formatSourceFast formats source code using go/format.Source.
// This is fast but doesn't group imports by category.
func formatSourceFast(src []byte) ([]byte, error) {
	return format.Source(src)
}

// formatSourceWithGrouping formats source code using imports.Process.
// This groups imports by stdlib/external/internal but is slower.
func formatSourceWithGrouping(filename string, src []byte) ([]byte, error) {
	opts := &imports.Options{
		FormatOnly: true,
		Comments:   true,
		TabIndent:  true,
		TabWidth:   defaultTabWidth,
	}
	return imports.Process(filename, src, opts)
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
