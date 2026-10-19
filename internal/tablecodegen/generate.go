package tablecodegen

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
)

// This file holds copies of the functions of codegen/generate.go and codegen/incremental.go
// that write the files of the executor, for the Data of table mode. TestCopies fails when
// the originals change.

// generateCode writes the files of the executor of data in its layout: the rest of
// codegen.GenerateCode.
func generateCode(data *Data) error {
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
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
		TemplateFS:      singleFileTemplateFS,
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

	err = addDirectives(data, &builds)
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
			GeneratedHeader: true,
			Packages:        data.Config.Packages,
			TemplateFS:      schemaTemplateFS,
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

// addBuild adds the Data of the file called filename to builds, which holds only some of
// the types of data, with what table mode derives from the whole schema of data, and the
// groups of tables of its own types.
func addBuild(filename string, p *ast.Position, data *Data, builds *map[string]*Data) {
	buildConfig := *data.Config
	if p != nil {
		buildConfig.Sources = []*ast.Source{p.Src}
	}

	// Per-schema builds hold only the objects declared in their own file, so the
	// batch-field set has to come from the whole schema. Without this, an interface
	// whose implementors are spread across files would only register batch parents
	// for the implementors that happen to share the interface's file. A Data computes
	// the set once, from its objects, and keeps it in its copies.
	build := &codegen.Data{Objects: data.Objects}
	build.HasBatchResolverFields()
	build.Objects = nil
	build.Config = &buildConfig
	build.QueryRoot = data.QueryRoot
	build.MutationRoot = data.MutationRoot
	build.SubscriptionRoot = data.SubscriptionRoot
	build.AllDirectives = data.AllDirectives

	info := data.tableInfo()
	(*builds)[filename] = &Data{
		Data:   build,
		shared: data.shared,
		info:   info,
		groups: info.groupsByFile[filename],
	}
}

// Root file contains top-level definitions that should not be duplicated across the generated
// files for each schema file.
func generateRootFile(data *Data) error {
	dir := data.Config.Exec.DirName
	path := filepath.Join(dir, "root_.generated.go")

	return templates.Render(templates.Options{
		PackageName:     data.Config.Exec.Package,
		Filename:        path,
		Data:            data,
		RegionTags:      false,
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
		TemplateFS:      rootTemplateFS,
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
			build.Interfaces = map[string]*codegen.Interface{}
		}
		if build.Interfaces[k] != nil {
			return errors.New("conflicting interface keys")
		}

		build.Interfaces[k] = inf
	}
	return nil
}

// addDirectives ensures a per-schema build exists for every source file that
// declares directives with arguments. Without this, a .graphql file containing
// only directive declarations (no type definitions) would produce no generated
// file, leaving the dir_*_args argument-parsing functions undefined.
//
// No directive data is added to the build struct: Data.Args() calls
// Directives(), which is scoped to the build's Config.Sources, so it
// automatically picks up the right directive args for each per-schema file.
func addDirectives(data *Data, builds *map[string]*Data) error {
	for _, directive := range data.Directives() {
		if len(directive.Args) == 0 {
			continue
		}
		fname := filename(directive.Position, data.Config)
		if (*builds)[fname] == nil {
			addBuild(fname, directive.Position, data, builds)
		}
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

// GenerateCode does in table mode what codegen.GenerateCode does: it fails where plugins
// changed the data since Build in what the runtime reads from the schema, finds the
// marshalers and unmarshalers of the functions mode that the code of the plugins calls,
// has what table mode derives from the data derived again from the data as the plugins
// left it, and writes the files of the executor.
func GenerateCode(d *Data) error {
	data := d.Data
	if !data.Config.Exec.IsDefined() {
		return errors.New("missing exec config")
	}
	if err := d.tableCheckState(); err != nil {
		return err
	}
	calls, err := collectPluginCalls(data)
	if err != nil {
		return err
	}
	d.shared.pluginCalls = calls
	for name := range calls {
		d.addPluginCall(name)
	}
	d.tableRederive()

	root := &Data{
		Data:              copyData(data),
		TableLinkedSchema: d.TableLinkedSchema,
		shared:            d.shared,
	}
	return generateCode(root)
}
