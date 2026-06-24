package config

import (
	"errors"
	"fmt"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/99designs/gqlgen/internal/code"
)

type ResolverConfig struct {
	Filename            string              `yaml:"filename,omitempty"`
	FilenameTemplate    string              `yaml:"filename_template,omitempty"`
	Package             string              `yaml:"package,omitempty"`
	Type                string              `yaml:"type,omitempty"`
	Layout              ResolverLayout      `yaml:"layout,omitempty"`
	DirName             string              `yaml:"dir"`
	Batch               ResolverBatchConfig `yaml:"batch,omitempty"`
	OmitTemplateComment bool                `yaml:"omit_template_comment,omitempty"`
	ResolverTemplate    string              `yaml:"resolver_template,omitempty"`
	PreserveResolver    bool                `yaml:"preserve_resolver,omitempty"`
}

// ResolverBatchConfig enables batch resolver generation for resolver fields as if they
// had @goField(batch: true). Root types (Query, Mutation, Subscription), input objects,
// and introspection types (__*) are always excluded. When federation is enabled, federation
// _Service and Entity are also excluded. Global batch does not convert struct-bound fields
// into resolvers. Individual fields can opt out with @goField(batch: false).
type ResolverBatchConfig struct {
	Enabled bool
}

func (b *ResolverBatchConfig) UnmarshalYAML(unmarshal func(any) error) error {
	var enabled bool
	if err := unmarshal(&enabled); err == nil {
		b.Enabled = enabled
		return nil
	}

	var long struct {
		Enabled bool `yaml:"enabled"`
	}
	if err := unmarshal(&long); err != nil {
		return err
	}

	b.Enabled = long.Enabled
	return nil
}

type ResolverLayout string

var (
	LayoutSingleFile   ResolverLayout = "single-file"
	LayoutFollowSchema ResolverLayout = "follow-schema"
)

func (r *ResolverConfig) Check() error {
	if r.Layout == "" {
		r.Layout = LayoutSingleFile
	}
	if r.Type == "" {
		r.Type = "Resolver"
	}

	switch r.Layout {
	case LayoutSingleFile:
		if r.Filename == "" {
			return fmt.Errorf("filename must be specified with layout=%s", r.Layout)
		}
		if !strings.HasSuffix(r.Filename, ".go") {
			return fmt.Errorf(
				"filename should be path to a go source file with layout=%s",
				r.Layout,
			)
		}
		r.Filename = abs(r.Filename)
	case LayoutFollowSchema:
		if r.DirName == "" {
			return fmt.Errorf("dirname must be specified with layout=%s", r.Layout)
		}
		r.DirName = abs(r.DirName)
		if r.Filename == "" {
			r.Filename = filepath.Join(r.DirName, "resolver.go")
		} else {
			r.Filename = abs(r.Filename)
		}
	default:
		return fmt.Errorf(
			"invalid layout %s. must be %s or %s",
			r.Layout,
			LayoutSingleFile,
			LayoutFollowSchema,
		)
	}

	if strings.ContainsAny(r.Package, "./\\") {
		return errors.New(
			"package should be the output package name only, do not include the output filename",
		)
	}

	if r.Package == "" && r.Dir() != "" {
		r.Package = code.NameForDir(r.Dir())
	}

	return nil
}

func (r *ResolverConfig) ImportPath() string {
	if r.Dir() == "" {
		return ""
	}
	return code.ImportPathForDir(r.Dir())
}

func (r *ResolverConfig) Dir() string {
	switch r.Layout {
	case LayoutSingleFile:
		if r.Filename == "" {
			return ""
		}
		return filepath.Dir(r.Filename)
	case LayoutFollowSchema:
		return r.DirName
	default:
		panic("invalid layout " + r.Layout)
	}
}

func (r *ResolverConfig) Pkg() *types.Package {
	if r.Dir() == "" {
		return nil
	}
	return types.NewPackage(r.ImportPath(), r.Package)
}

func (r *ResolverConfig) IsDefined() bool {
	return r.Filename != "" || r.DirName != ""
}
