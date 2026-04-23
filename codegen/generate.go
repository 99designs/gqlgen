package codegen

import (
	"embed"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
)

//go:embed *.gotpl
var codegenTemplates embed.FS

func GenerateCode(data *Data) error {
	if !data.Config.Exec.IsDefined() {
		return errors.New("missing exec config")
	}

	switch data.Config.Exec.Layout {
	case config.ExecLayoutSingleFile:
		return generateSingleFile(data)
	case config.ExecLayoutFollowSchema:
		return generatePerSchema(data)
	}

	return fmt.Errorf("unrecognized exec layout %s", data.Config.Exec.Layout)
}

func generateSingleFile(data *Data) error {
	return templates.Render(templates.Options{
		PackageName:     data.Config.Exec.Package,
		Filename:        data.Config.Exec.Filename,
		Data:            data,
		RegionTags:      true,
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
		TemplateFS:      codegenTemplates,
		PruneOptions:    data.Config.GetPruneOptions(),
	})
}

type build struct {
	name string // original case filename for file I/O
	data *Data
}

func generatePerSchema(data *Data) error {
	err := generateRootFile(data)
	if err != nil {
		return err
	}

	builds := map[string]*build{}

	err = addObjects(data, &builds)
	if err != nil {
		return err
	}

	err = addInputs(data, &builds)
	if err != nil {
		return err
	}

	err = addInterfaces(data, &builds)
	if err != nil {
		return err
	}

	err = addReferencedTypes(data, &builds)
	if err != nil {
		return err
	}

	for _, b := range builds {
		if b.name == "" {
			continue
		}

		dir := data.Config.Exec.DirName
		path := filepath.Join(dir, b.name)

		err = templates.Render(templates.Options{
			PackageName:     data.Config.Exec.Package,
			Filename:        path,
			Data:            b.data,
			RegionTags:      true,
			GeneratedHeader: true,
			Packages:        data.Config.Packages,
			TemplateFS:      codegenTemplates,
			PruneOptions:    data.Config.GetPruneOptions(),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func filename(p *ast.Position, config *config.Config) string {
	name := "common!"
	if p != nil && p.Src != nil {
		gqlname := filepath.Base(p.Src.Name)
		ext := filepath.Ext(p.Src.Name)
		name = strings.TrimSuffix(gqlname, ext)
	}

	filenameTempl := config.Exec.FilenameTemplate
	if filenameTempl == "" {
		filenameTempl = "{name}.generated.go"
	}

	return strings.ReplaceAll(filenameTempl, "{name}", name)
}

func addBuild(fnCase string, fnKey string, p *ast.Position, data *Data, builds *map[string]*build) {
	buildConfig := *data.Config
	if p != nil {
		buildConfig.Sources = []*ast.Source{p.Src}
	}

	(*builds)[fnKey] = &build{
		name: fnCase,
		data: &Data{
			Config:           &buildConfig,
			QueryRoot:        data.QueryRoot,
			MutationRoot:     data.MutationRoot,
			SubscriptionRoot: data.SubscriptionRoot,
			AllDirectives:    data.AllDirectives,
		},
	}
}

//go:embed root_.gotpl
var rootTemplate string

// Root file contains top-level definitions that should not be duplicated across the generated
// files for each schema file.
func generateRootFile(data *Data) error {
	dir := data.Config.Exec.DirName
	path := filepath.Join(dir, "root_.generated.go")

	return templates.Render(templates.Options{
		PackageName:     data.Config.Exec.Package,
		Template:        rootTemplate,
		Filename:        path,
		Data:            data,
		RegionTags:      false,
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
		TemplateFS:      codegenTemplates,
		PruneOptions:    data.Config.GetPruneOptions(),
	})
}

func addObjects(data *Data, builds *map[string]*build) error {
	for _, o := range data.Objects {
		fnCase := filename(o.Position, data.Config)
		fn := strings.ToLower(fnCase)
		if (*builds)[fn] == nil {
			addBuild(fnCase, fn, o.Position, data, builds)
		}

		(*builds)[fn].data.Objects = append((*builds)[fn].data.Objects, o)
	}
	return nil
}

func addInputs(data *Data, builds *map[string]*build) error {
	for _, in := range data.Inputs {
		fnCase := filename(in.Position, data.Config)
		fn := strings.ToLower(fnCase)
		if (*builds)[fn] == nil {
			addBuild(fnCase, fn, in.Position, data, builds)
		}

		(*builds)[fn].data.Inputs = append((*builds)[fn].data.Inputs, in)
	}
	return nil
}

func addInterfaces(data *Data, builds *map[string]*build) error {
	for k, inf := range data.Interfaces {
		fnCase := filename(inf.Position, data.Config)
		fn := strings.ToLower(fnCase)
		if (*builds)[fn] == nil {
			addBuild(fnCase, fn, inf.Position, data, builds)
		}
		b := (*builds)[fn]

		if b.data.Interfaces == nil {
			b.data.Interfaces = map[string]*Interface{}
		}
		if b.data.Interfaces[k] != nil {
			return errors.New("conflicting interface keys")
		}

		b.data.Interfaces[k] = inf
	}
	return nil
}

func addReferencedTypes(data *Data, builds *map[string]*build) error {
	for k, rt := range data.ReferencedTypes {
		fnCase := filename(rt.Definition.Position, data.Config)
		fn := strings.ToLower(fnCase)
		if (*builds)[fn] == nil {
			addBuild(fnCase, fn, rt.Definition.Position, data, builds)
		}
		b := (*builds)[fn]

		if b.data.ReferencedTypes == nil {
			b.data.ReferencedTypes = map[string]*config.TypeReference{}
		}
		if b.data.ReferencedTypes[k] != nil {
			return errors.New("conflicting referenced type keys")
		}

		b.data.ReferencedTypes[k] = rt
	}
	return nil
}
