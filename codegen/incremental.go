package codegen

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
)

// IncrementalOptions configures incremental generation
type IncrementalOptions struct {
	// ChangedSchemas is the list of schema file paths that have changed.
	// If empty, performs full generation.
	ChangedSchemas []string

	// Verbose enables detailed logging
	Verbose bool
}

// GenerateCodeIncremental generates code only for schemas affected by changes.
// If no changed schemas are specified, falls back to full generation.
func GenerateCodeIncremental(data *Data, opts IncrementalOptions) error {
	if !data.Config.Exec.IsDefined() {
		return fmt.Errorf("missing exec config")
	}

	// Only follow-schema layout supports incremental generation
	if data.Config.Exec.Layout != config.ExecLayoutFollowSchema {
		if opts.Verbose {
			log.Println("[incremental] Only follow-schema layout supported, using full generation")
		}
		return GenerateCode(data)
	}

	// No changed schemas specified = full generation
	if len(opts.ChangedSchemas) == 0 {
		if opts.Verbose {
			log.Println("[incremental] No changed schemas specified, using full generation")
		}
		return GenerateCode(data)
	}

	// Build dependency graph and compute affected schemas
	depGraph := BuildDependencyGraph(data.Config.Schema)
	affectedSchemas := depGraph.GetAffectedSchemas(opts.ChangedSchemas)
	affectedTypes := depGraph.GetTypesForSchemas(affectedSchemas)

	if opts.Verbose {
		log.Printf("[incremental] Changed: %d, Affected: %d schemas, %d types\n",
			len(opts.ChangedSchemas), len(affectedSchemas), len(affectedTypes))
	}

	// Generate root file (always needed)
	if err := generateRootFile(data); err != nil {
		return err
	}

	// Build per-schema data, filtered to affected schemas only
	builds := make(map[string]*Data)
	affectedSet := makeSet(affectedSchemas)

	for _, o := range data.Objects {
		if !isAffected(o.Position, affectedSet) {
			continue
		}
		fn := filename(o.Position, data.Config)
		if builds[fn] == nil {
			addBuild(fn, o.Position, data, &builds)
		}
		builds[fn].Objects = append(builds[fn].Objects, o)
	}

	for _, in := range data.Inputs {
		if !isAffected(in.Position, affectedSet) {
			continue
		}
		fn := filename(in.Position, data.Config)
		if builds[fn] == nil {
			addBuild(fn, in.Position, data, &builds)
		}
		builds[fn].Inputs = append(builds[fn].Inputs, in)
	}

	for k, inf := range data.Interfaces {
		if !isAffected(inf.Position, affectedSet) {
			continue
		}
		fn := filename(inf.Position, data.Config)
		if builds[fn] == nil {
			addBuild(fn, inf.Position, data, &builds)
		}
		if builds[fn].Interfaces == nil {
			builds[fn].Interfaces = make(map[string]*Interface)
		}
		builds[fn].Interfaces[k] = inf
	}

	for k, rt := range data.ReferencedTypes {
		if !isAffected(rt.Definition.Position, affectedSet) {
			continue
		}
		fn := filename(rt.Definition.Position, data.Config)
		if builds[fn] == nil {
			addBuild(fn, rt.Definition.Position, data, &builds)
		}
		if builds[fn].ReferencedTypes == nil {
			builds[fn].ReferencedTypes = make(map[string]*config.TypeReference)
		}
		builds[fn].ReferencedTypes[k] = rt
	}

	// Render affected files
	for fn, build := range builds {
		if fn == "" {
			continue
		}
		path := filepath.Join(data.Config.Exec.DirName, fn)
		if opts.Verbose {
			log.Printf("[incremental] Generating: %s\n", fn)
		}
		if err := templates.Render(templates.Options{
			PackageName:     data.Config.Exec.Package,
			Filename:        path,
			Data:            build,
			RegionTags:      true,
			GeneratedHeader: true,
			Packages:        data.Config.Packages,
			TemplateFS:      codegenTemplates,
		}); err != nil {
			return err
		}
	}

	return nil
}

func isAffected(pos *ast.Position, affectedSet map[string]bool) bool {
	if pos == nil || pos.Src == nil {
		return true // Unknown source, include to be safe
	}
	return affectedSet[pos.Src.Name]
}

func makeSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}
