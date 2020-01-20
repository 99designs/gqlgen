package config

import (
	"fmt"
	"go/types"
	"path/filepath"

	"github.com/99designs/gqlgen/internal/code"
)

type ResolverConfig struct {
	PackageConfig `yaml:",inline"`
	Layout        ResolverLayout `yaml:"layout,omitempty"`
	DirName       string         `yaml:"dir"`
}

type ResolverLayout string

var (
	LayoutSingleFile   ResolverLayout = "single-file"
	LayoutFollowSchema ResolverLayout = "follow-schema"
)

func (r *ResolverConfig) Check(filesMap map[string]bool, pkgConfigsByDir map[string]*PackageConfig) error {
	if r.DirName != "" {
		r.DirName = abs(r.DirName)
	}

	if r.Layout == "" {
		r.Layout = "single-file"
	}
	if r.Layout != LayoutFollowSchema && r.Layout != LayoutSingleFile {
		return fmt.Errorf("invalid layout %s. must be single-file or follow-schema", r.Layout)
	}

	if r.Layout == "follow-schema" && r.DirName == "" {
		return fmt.Errorf("must specify dir when using laout:follow-schema")
	}

	return r.PackageConfig.Check(filesMap, pkgConfigsByDir)
}

func (r *ResolverConfig) ImportPath() string {
	return code.ImportPathForDir(r.Dir())
}

func (r *ResolverConfig) Dir() string {
	switch r.Layout {
	case LayoutSingleFile:
		return filepath.Dir(r.Filename)
	case LayoutFollowSchema:
		return r.DirName
	default:
		panic("invalid layout " + r.Layout)
	}
}

func (r *ResolverConfig) Pkg() *types.Package {
	return types.NewPackage(r.ImportPath(), r.Dir())
}

func (r *ResolverConfig) IsDefined() bool {
	switch r.Layout {
	case LayoutSingleFile, "":
		return r.Filename != ""
	case LayoutFollowSchema:
		return r.DirName != ""
	default:
		panic(fmt.Errorf("invalid layout %s", r.Layout))
	}
}
