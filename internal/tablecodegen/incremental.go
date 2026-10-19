package tablecodegen

import (
	"errors"
	"log"

	"github.com/99designs/gqlgen/codegen"
)

// GenerateCodeIncremental is codegen.GenerateCodeIncremental in table mode.
func GenerateCodeIncremental(d *Data, opts codegen.IncrementalOptions) error {
	data := d.Data
	if !data.Config.Exec.IsDefined() {
		return errors.New("missing exec config")
	}

	if opts.Verbose && len(opts.ChangedSchemas) > 0 {
		// Build dependency graph for informational logging
		depGraph := codegen.BuildDependencyGraph(data.Config.Schema)
		affectedSchemas := depGraph.GetAffectedSchemas(opts.ChangedSchemas)
		affectedTypes := depGraph.GetTypesForSchemas(affectedSchemas)
		log.Printf("[incremental] Changed: %d, Affected: %d schemas, %d types\n",
			len(opts.ChangedSchemas), len(affectedSchemas), len(affectedTypes))
	}

	// Perform full generation - the content-based write() in templates
	// will skip writing files whose content hasn't changed, preserving
	// their mtime and keeping Go's build cache valid.
	return GenerateCode(d)
}
