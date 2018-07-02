package codegen

import (
	"bytes"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/vektah/gqlgen/codegen/templates"
	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/imports"
)

type Config struct {
	SchemaStr string
	Typemap   TypeMap

	schema *schema.Schema

	ExecFilename    string
	ExecPackageName string
	execPackagePath string
	execDir         string

	ModelFilename    string
	ModelPackageName string
	modelPackagePath string
	modelDir         string
}

type TypeMap []TypeMapEntry

func (tm TypeMap) Exists(typeName string) bool {
	return tm.Get(typeName) != nil
}

func (tm TypeMap) Get(typeName string) *TypeMapEntry {
	for _, entry := range tm {
		if entry.TypeName == typeName {
			return &entry
		}
	}

	return nil
}

type TypeMapEntry struct {
	TypeName   string `yaml:"typeName"`
	EntityPath string `yaml:"entityPath"`
	Fields     []TypeMapField
}

type TypeMapField struct {
	FieldName string `yaml:"fieldName"`
}

func Generate(cfg Config) error {
	if err := cfg.normalize(); err != nil {
		return err
	}

	_ = syscall.Unlink(cfg.ExecFilename)
	_ = syscall.Unlink(cfg.ModelFilename)

	modelsBuild, err := cfg.models()
	if err != nil {
		return errors.Wrap(err, "model plan failed")
	}
	if len(modelsBuild.Models) > 0 || len(modelsBuild.Enums) > 0 {
		modelsBuild.PackageName = cfg.ModelPackageName
		var buf *bytes.Buffer
		buf, err = templates.Run("models.gotpl", modelsBuild)
		if err != nil {
			return errors.Wrap(err, "model generation failed")
		}

		if err = write(cfg.ModelFilename, buf.Bytes()); err != nil {
			return err
		}
		for _, model := range modelsBuild.Models {
			cfg.Typemap = append(cfg.Typemap, TypeMapEntry{
				TypeName:   model.GQLType,
				EntityPath: cfg.modelPackagePath + "." + model.GoType,
			})
		}

		for _, enum := range modelsBuild.Enums {
			cfg.Typemap = append(cfg.Typemap, TypeMapEntry{
				TypeName:   enum.GQLType,
				EntityPath: cfg.modelPackagePath + "." + enum.GoType,
			})
		}
	}

	build, err := cfg.bind()
	if err != nil {
		return errors.Wrap(err, "exec plan failed")
	}
	build.SchemaRaw = cfg.SchemaStr
	build.PackageName = cfg.ExecPackageName

	var buf *bytes.Buffer
	buf, err = templates.Run("generated.gotpl", build)
	if err != nil {
		return errors.Wrap(err, "exec codegen failed")
	}

	if err = write(cfg.ExecFilename, buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) normalize() error {
	if cfg.ModelFilename == "" {
		return errors.New("ModelFilename is required")
	}
	cfg.ModelFilename = abs(cfg.ModelFilename)
	cfg.modelDir = filepath.ToSlash(filepath.Dir(cfg.ModelFilename))
	if cfg.ModelPackageName == "" {
		cfg.ModelPackageName = filepath.Base(cfg.modelDir)
	}
	cfg.ModelPackageName = sanitizePackageName(cfg.ModelPackageName)
	cfg.modelPackagePath = fullPackageName(cfg.modelDir, cfg.ModelPackageName)

	if cfg.ExecFilename == "" {
		return errors.New("ModelFilename is required")
	}
	cfg.ExecFilename = abs(cfg.ExecFilename)
	cfg.execDir = filepath.ToSlash(filepath.Dir(cfg.ExecFilename))
	if cfg.ExecPackageName == "" {
		cfg.ExecPackageName = filepath.Base(cfg.execDir)
	}
	cfg.ExecPackageName = sanitizePackageName(cfg.ExecPackageName)
	cfg.execPackagePath = fullPackageName(cfg.execDir, cfg.ExecPackageName)

	builtins := TypeMap{
		{TypeName: "__Directive", EntityPath: "github.com/vektah/gqlgen/neelance/introspection.Directive"},
		{TypeName: "__Type", EntityPath: "github.com/vektah/gqlgen/neelance/introspection.Type"},
		{TypeName: "__Field", EntityPath: "github.com/vektah/gqlgen/neelance/introspection.Field"},
		{TypeName: "__EnumValue", EntityPath: "github.com/vektah/gqlgen/neelance/introspection.EnumValue"},
		{TypeName: "__InputValue", EntityPath: "github.com/vektah/gqlgen/neelance/introspection.InputValue"},
		{TypeName: "__Schema", EntityPath: "github.com/vektah/gqlgen/neelance/introspection.Schema"},
		{TypeName: "Int", EntityPath: "github.com/vektah/gqlgen/graphql.Int"},
		{TypeName: "Float", EntityPath: "github.com/vektah/gqlgen/graphql.Float"},
		{TypeName: "String", EntityPath: "github.com/vektah/gqlgen/graphql.String"},
		{TypeName: "Boolean", EntityPath: "github.com/vektah/gqlgen/graphql.Boolean"},
		{TypeName: "ID", EntityPath: "github.com/vektah/gqlgen/graphql.ID"},
		{TypeName: "Time", EntityPath: "github.com/vektah/gqlgen/graphql.Time"},
		{TypeName: "Map", EntityPath: "github.com/vektah/gqlgen/graphql.Map"},
	}

	if cfg.Typemap == nil {
		cfg.Typemap = TypeMap{}
	}
	for _, entry := range builtins {
		if !cfg.Typemap.Exists(entry.TypeName) {
			cfg.Typemap = append(cfg.Typemap, entry)
		}
	}

	cfg.schema = schema.New()
	return cfg.schema.Parse(cfg.SchemaStr)
}

var invalidPackageNameChar = regexp.MustCompile(`[^\w]`)

func sanitizePackageName(pkg string) string {
	return invalidPackageNameChar.ReplaceAllLiteralString(filepath.Base(pkg), "_")
}

func abs(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return filepath.ToSlash(absPath)
}

func fullPackageName(dir string, pkgName string) string {
	fullPkgName := filepath.Join(filepath.Dir(dir), pkgName)

	for _, gopath := range filepath.SplitList(build.Default.GOPATH) {
		gopath = filepath.Join(gopath, "src") + string(os.PathSeparator)
		if len(gopath) > len(fullPkgName) {
			continue
		}
		if strings.EqualFold(gopath, fullPkgName[0:len(gopath)]) {
			fullPkgName = fullPkgName[len(gopath):]
			break
		}
	}
	return filepath.ToSlash(fullPkgName)
}

func gofmt(filename string, b []byte) ([]byte, error) {
	out, err := imports.Process(filename, b, nil)
	if err != nil {
		return b, errors.Wrap(err, "unable to gofmt")
	}
	return out, nil
}

func write(filename string, b []byte) error {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create directory")
	}

	formatted, err := gofmt(filename, b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gofmt failed: %s\n", err.Error())
		formatted = b
	}

	err = ioutil.WriteFile(filename, formatted, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", filename)
	}

	return nil
}
