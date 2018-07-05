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
	SchemaFilename string  `yaml:"schema,omitempty"`
	SchemaStr      string  `yaml:"-"`
	Typemap        TypeMap `yaml:"models,omitempty"`

	schema *schema.Schema `yaml:"-"`

	ExecFilename    string `yaml:"output,omitempty"`
	ExecPackageName string `yaml:"package,omitempty"`
	execPackagePath string `yaml:"-"`
	execDir         string `yaml:"-"`

	ModelFilename    string `yaml:"modeloutput,omitempty"`
	ModelPackageName string `yaml:"modelpackage,omitempty"`
	modelPackagePath string `yaml:"-"`
	modelDir         string `yaml:"-"`
}

func (cfg *Config) Check() error {
	err := cfg.Typemap.Check()
	if err != nil {
		return fmt.Errorf("config: %s", err.Error())
	}
	return nil
}

type TypeMap map[string]TypeMapEntry

func (tm TypeMap) Exists(typeName string) bool {
	return tm.Get(typeName) != nil
}

func (tm TypeMap) Get(typeName string) *TypeMapEntry {
	entry, ok := tm[typeName]
	if !ok {
		return nil
	}
	return &entry
}

func (tm TypeMap) Check() error {
	for typeName, entry := range tm {
		if entry.Model == "" {
			return fmt.Errorf("model %s: entityPath is not defined", typeName)
		}
	}
	return nil
}

type TypeMapEntry struct {
	Model  string                  `yaml:"model"`
	Fields map[string]TypeMapField `yaml:"fields,omitempty"`
}

type TypeMapField struct {
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
			cfg.Typemap[model.GQLType] = TypeMapEntry{
				Model: cfg.modelPackagePath + "." + model.GoType,
			}
		}

		for _, enum := range modelsBuild.Enums {
			cfg.Typemap[enum.GQLType] = TypeMapEntry{
				Model: cfg.modelPackagePath + "." + enum.GoType,
			}
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

	if err = cfg.validate(); err != nil {
		return errors.Wrap(err, "validation failed")
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
		"__Directive":  {Model: "github.com/vektah/gqlgen/neelance/introspection.Directive"},
		"__Type":       {Model: "github.com/vektah/gqlgen/neelance/introspection.Type"},
		"__Field":      {Model: "github.com/vektah/gqlgen/neelance/introspection.Field"},
		"__EnumValue":  {Model: "github.com/vektah/gqlgen/neelance/introspection.EnumValue"},
		"__InputValue": {Model: "github.com/vektah/gqlgen/neelance/introspection.InputValue"},
		"__Schema":     {Model: "github.com/vektah/gqlgen/neelance/introspection.Schema"},
		"Int":          {Model: "github.com/vektah/gqlgen/graphql.Int"},
		"Float":        {Model: "github.com/vektah/gqlgen/graphql.Float"},
		"String":       {Model: "github.com/vektah/gqlgen/graphql.String"},
		"Boolean":      {Model: "github.com/vektah/gqlgen/graphql.Boolean"},
		"ID":           {Model: "github.com/vektah/gqlgen/graphql.ID"},
		"Time":         {Model: "github.com/vektah/gqlgen/graphql.Time"},
		"Map":          {Model: "github.com/vektah/gqlgen/graphql.Map"},
	}

	if cfg.Typemap == nil {
		cfg.Typemap = TypeMap{}
	}
	for typeName, entry := range builtins {
		if !cfg.Typemap.Exists(typeName) {
			cfg.Typemap[typeName] = entry
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
