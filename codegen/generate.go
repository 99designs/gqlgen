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

func generatePerSchema(data *Data) error {
	err := generateRootFile(data)
	if err != nil {
		return err
	}

	builds := map[string]*Data{}

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

	for filename, build := range builds {
		if filename == "" {
			continue
		}

		dir := data.Config.Exec.DirName
		path := filepath.Join(dir, filename)

		err = templates.Render(templates.Options{
			PackageName:     data.Config.Exec.Package,
			Filename:        path,
			Data:            build,
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

func addBuild(filename string, p *ast.Position, data *Data, builds *map[string]*Data) {
	buildConfig := *data.Config
	if p != nil {
		buildConfig.Sources = []*ast.Source{p.Src}
	}

	(*builds)[filename] = &Data{
		Config:                 &buildConfig,
		QueryRoot:              data.QueryRoot,
		MutationRoot:           data.MutationRoot,
		SubscriptionRoot:       data.SubscriptionRoot,
		AllDirectives:          data.AllDirectives,
		SkipLocationDirectives: true,
	}
}

//go:embed root_.gotpl
var rootTemplate string

//go:embed directives.gotpl
var directivesTemplate string

// Root file contains top-level definitions that should not be duplicated across the generated
// files for each schema file.
// In follow-schema layout, location directive middleware (_fieldMiddleware etc.)
// and orphan directive args are generated here instead of per-schema files.
func generateRootFile(data *Data) error {
	dir := data.Config.Exec.DirName
	path := filepath.Join(dir, "root_.generated.go")

	return templates.Render(templates.Options{
		PackageName:     data.Config.Exec.Package,
		Template:        rootTemplate + "\n" + directivesTemplate,
		Filename:        path,
		Data:            data,
		RegionTags:      false,
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
		TemplateFS:      codegenTemplates,
		PruneOptions:    data.Config.GetPruneOptions(),
	})
}

func addObjects(data *Data, builds *map[string]*Data) error {
	for _, o := range data.Objects {
		filename := filename(o.Position, data.Config)
		if (*builds)[filename] == nil {
			addBuild(filename, o.Position, data, builds)
		}

		(*builds)[filename].Objects = append((*builds)[filename].Objects, o)
	}
	return nil
}

func addInputs(data *Data, builds *map[string]*Data) error {
	for _, in := range data.Inputs {
		filename := filename(in.Position, data.Config)
		if (*builds)[filename] == nil {
			addBuild(filename, in.Position, data, builds)
		}

		(*builds)[filename].Inputs = append((*builds)[filename].Inputs, in)
	}
	return nil
}

func addInterfaces(data *Data, builds *map[string]*Data) error {
	for k, inf := range data.Interfaces {
		filename := filename(inf.Position, data.Config)
		if (*builds)[filename] == nil {
			addBuild(filename, inf.Position, data, builds)
		}
		build := (*builds)[filename]

		if build.Interfaces == nil {
			build.Interfaces = map[string]*Interface{}
		}
		if build.Interfaces[k] != nil {
			return errors.New("conflicting interface keys")
		}

		build.Interfaces[k] = inf
	}
	return nil
}

func addReferencedTypes(data *Data, builds *map[string]*Data) error {
	for k, rt := range data.ReferencedTypes {
		filename := filename(rt.Definition.Position, data.Config)
		if (*builds)[filename] == nil {
			addBuild(filename, rt.Definition.Position, data, builds)
		}
		build := (*builds)[filename]

		if build.ReferencedTypes == nil {
			build.ReferencedTypes = map[string]*config.TypeReference{}
		}
		if build.ReferencedTypes[k] != nil {
			return errors.New("conflicting referenced type keys")
		}

		build.ReferencedTypes[k] = rt
	}
	return nil
}
