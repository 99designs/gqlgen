package codegen

import (
	"go/types"

	"io/ioutil"

	"path/filepath"

	"fmt"

	"github.com/99designs/gqlgen/internal/edit"
	"github.com/99designs/gqlgen/internal/gopath"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/loader"
)

func Stubs(cfg *Config) error {
	if !cfg.Resolver.IsDefined() {
		return errors.New("resolver is required in config for stub generation")
	}

	conf := loader.Config{}
	conf.Import(cfg.Resolver.ImportPath())
	conf.Import(cfg.Exec.ImportPath())
	prog, err := conf.Load()
	if err != nil {
		return errors.Wrap(err, "failed to load existing resolver package")
	}

	srcInterface, err := findGoType(prog, cfg.Exec.ImportPath(), "ResolverRoot")
	if err != nil {
		return errors.Wrapf(err, "failed to find source root resolver interface")
	}

	fe := fileEdits{
		prog:  prog,
		files: map[string]*edit.Buffer{},
	}
	fe.updateRoot(srcInterface.Type().(*types.Named).Underlying().(*types.Interface), cfg.Resolver.Filename, cfg.Resolver.Type)

	for filename, file := range fe.files {
		fmt.Println(filename)
		fmt.Println("=============")
		fmt.Println(file.Result())
	}

	return nil
}

type fileEdits struct {
	prog  *loader.Program
	files map[string]*edit.Buffer
}

func (f *fileEdits) updateRoot(src *types.Interface, filename string, typeName string) {
	importPath := gopath.MustDir2Import(filepath.Dir(filename))

	dest, err := findGoType(f.prog, importPath, typeName)
	if err != nil {
		file := f.getFile(filename)
		file.Append(tpl("\ntype {{.ResolverType}} struct {}", map[string]interface{}{
			"ResolverType": typeName,
		}))
	}

	destNamed, _ := dest.Type().(*types.Named)
	for i := 0; i < src.NumMethods(); i++ {
		srcMethod := src.Method(i)

		var destMethod *types.Func
		if destNamed != nil {
			destMethod = findMethod(destNamed, srcMethod.Name())
		}

		f.updateResolverType(srcMethod, destMethod, filename, typeName)
	}
}

func (f *fileEdits) updateResolverType(src *types.Func, dest *types.Func, filename string, typeName string) {
	if dest == nil {
		f.getFile(filename).Append(tpl(`
		func (r *{{.ResolverType}}) {{.MethodName}}() {{ .ResolverTypeName }} {
			return &{{.ResolverTypeName}}{r}
		}
		type {{ .ResolverTypeName }} { *{{.ResolverType}} }
		`, map[string]interface{}{
			"ResolverType":     typeName,
			"MethodName":       src.Name(),
			"ResolverTypeName": src.Name() + "Resolver",
		}))
	} else {

	}
}

func (f fileEdits) getFile(filename string) *edit.Buffer {
	if file := f.files[filename]; file != nil {
		return file
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	file := edit.New(string(b))
	f.files[filename] = file
	return file
}
