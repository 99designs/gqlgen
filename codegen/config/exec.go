package config

import (
	"fmt"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/99designs/gqlgen/internal/code"
)

type ExecConfig struct {
	Filename string `yaml:"filename,omitempty"`
	Package  string `yaml:"package,omitempty"`
}

func (r *ExecConfig) Check() error {
	if r.Filename == "" {
		return fmt.Errorf("filename must be specified")
	}
	if !strings.HasSuffix(r.Filename, ".go") {
		return fmt.Errorf("filename should be path to a go source file")
	}
	r.Filename = abs(r.Filename)

	if strings.ContainsAny(r.Package, "./\\") {
		return fmt.Errorf("package should be the output package name only, do not include the output filename")
	}

	if r.Package == "" && r.Dir() != "" {
		r.Package = code.NameForDir(r.Dir())
	}

	return nil
}

func (r *ExecConfig) ImportPath() string {
	if r.Dir() == "" {
		return ""
	}
	return code.ImportPathForDir(r.Dir())
}

func (r *ExecConfig) Dir() string {
	if r.Filename == "" {
		return ""
	}
	return filepath.Dir(r.Filename)
}

func (r *ExecConfig) Pkg() *types.Package {
	if r.Dir() == "" {
		return nil
	}
	return types.NewPackage(r.ImportPath(), r.Package)
}

func (r *ExecConfig) IsDefined() bool {
	return r.Filename != ""
}
