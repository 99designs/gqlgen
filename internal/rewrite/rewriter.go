package rewrite

import (
	"fmt"
	"go/ast"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"golang.org/x/tools/go/packages"
)

type Rewriter struct {
	pkg   *packages.Package
	files map[string]string
}

func New(importPath string) (*Rewriter, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypes,
	}, importPath)
	if err != nil {
		return nil, err
	}

	return &Rewriter{
		pkg:   pkgs[0],
		files: map[string]string{},
	}, nil
}

func (r *Rewriter) getSource(start, end token.Pos) string {
	startPos := r.pkg.Fset.Position(start)
	endPos := r.pkg.Fset.Position(end)

	if startPos.Filename != endPos.Filename {
		panic("cant get source spanning multiple files")
	}

	file := r.getFile(startPos.Filename)
	return file[startPos.Offset:endPos.Offset]
}

func (r *Rewriter) getFile(filename string) string {
	if _, ok := r.files[filename]; !ok {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(fmt.Errorf("unable to load file, already exists: %s", err.Error()))
		}

		r.files[filename] = string(b)

	}

	return r.files[filename]
}

func (r *Rewriter) GetMethodBody(structname string, methodname string) string {
	for _, f := range r.pkg.Syntax {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Name.Name != methodname {
					continue
				}
				if d.Recv == nil || d.Recv.List == nil {
					continue
				}
				recv := d.Recv.List[0].Type
				if star, isStar := d.Recv.List[0].Type.(*ast.StarExpr); isStar {
					recv = star.X
				}
				ident, ok := recv.(*ast.Ident)
				if !ok {
					continue
				}

				if ident.Name != structname {
					continue
				}

				return r.getSource(d.Body.Pos()+1, d.Body.End()-1)
			}
		}
	}

	return ""
}

func (r *Rewriter) ExistingImports(filename string) []Import {
	filename, err := filepath.Abs(filename)
	if err != nil {
		panic(err)
	}
	for _, f := range r.pkg.Syntax {
		pos := r.pkg.Fset.Position(f.Pos())

		if filename != pos.Filename {
			continue
		}

		var imps []Import
		for _, i := range f.Imports {
			name := ""
			if i.Name != nil {
				name = i.Name.Name
			}
			path, err := strconv.Unquote(i.Path.Value)
			if err != nil {
				panic(err)
			}
			imps = append(imps, Import{name, path})
		}
		return imps
	}
	return nil
}

type Import struct {
	Alias      string
	ImportPath string
}
