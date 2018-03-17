package codegen

import (
	"fmt"
	"go/build"
	"go/types"
	"os"
	"path/filepath"

	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/go/loader"
)

type Build struct {
	PackageName      string
	Objects          Objects
	Inputs           Objects
	Interfaces       []*Interface
	Imports          Imports
	QueryRoot        *Object
	MutationRoot     *Object
	SubscriptionRoot *Object
	SchemaRaw        string
}

type ModelBuild struct {
	PackageName string
	Imports     Imports
	Models      []Model
}

// Create a list of models that need to be generated
func Models(schema *schema.Schema, userTypes map[string]string, destDir string) *ModelBuild {
	namedTypes := buildNamedTypes(schema, userTypes)

	imports := buildImports(namedTypes, destDir)
	prog, err := loadProgram(imports, true)
	if err != nil {
		panic(err)
	}

	bindTypes(imports, namedTypes, prog)

	models := buildModels(namedTypes, schema, prog)
	return &ModelBuild{
		PackageName: filepath.Base(destDir),
		Models:      models,
		Imports:     buildImports(namedTypes, destDir),
	}
}

// Bind a schema together with some code to generate a Build
func Bind(schema *schema.Schema, userTypes map[string]string, destDir string) (*Build, error) {
	namedTypes := buildNamedTypes(schema, userTypes)

	imports := buildImports(namedTypes, destDir)
	prog, err := loadProgram(imports, false)
	if err != nil {
		return nil, err
	}

	bindTypes(imports, namedTypes, prog)

	objects := buildObjects(namedTypes, schema, prog, imports)
	inputs := buildInputs(namedTypes, schema, prog, imports)

	b := &Build{
		PackageName: filepath.Base(destDir),
		Objects:     objects,
		Interfaces:  buildInterfaces(namedTypes, schema, prog),
		Inputs:      inputs,
		Imports:     imports,
	}

	if qr, ok := schema.EntryPoints["query"]; ok {
		b.QueryRoot = b.Objects.ByName(qr.TypeName())
	}

	if mr, ok := schema.EntryPoints["mutation"]; ok {
		b.MutationRoot = b.Objects.ByName(mr.TypeName())
	}

	if sr, ok := schema.EntryPoints["subscription"]; ok {
		b.SubscriptionRoot = b.Objects.ByName(sr.TypeName())
	}

	if b.QueryRoot == nil {
		return b, fmt.Errorf("query entry point missing")
	}

	// Poke a few magic methods into query
	q := b.Objects.ByName(b.QueryRoot.GQLType)
	q.Fields = append(q.Fields, Field{
		Type:         &Type{namedTypes["__Schema"], []string{modPtr}, ""},
		GQLName:      "__schema",
		NoErr:        true,
		GoMethodName: "ec.introspectSchema",
		Object:       q,
	})
	q.Fields = append(q.Fields, Field{
		Type:         &Type{namedTypes["__Type"], []string{modPtr}, ""},
		GQLName:      "__type",
		NoErr:        true,
		GoMethodName: "ec.introspectType",
		Args: []FieldArgument{
			{GQLName: "name", Type: &Type{namedTypes["String"], []string{}, ""}, Object: &Object{}},
		},
		Object: q,
	})

	return b, nil
}

func loadProgram(imports Imports, allowErrors bool) (*loader.Program, error) {
	conf := loader.Config{}
	if allowErrors {
		conf = loader.Config{
			AllowErrors: true,
			TypeChecker: types.Config{
				Error: func(e error) {},
			},
		}
	}
	for _, imp := range imports {
		if imp.Package != "" {
			conf.Import(imp.Package)
		}
	}

	return conf.Load()
}

func resolvePkg(pkgName string) (string, error) {
	cwd, _ := os.Getwd()

	pkg, err := build.Default.Import(pkgName, cwd, build.FindOnly)
	if err != nil {
		return "", err
	}

	return pkg.ImportPath, nil
}
