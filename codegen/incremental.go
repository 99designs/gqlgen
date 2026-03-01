package codegen

import (
	"fmt"
	"log"
)

// IncrementalOptions configures incremental generation
type IncrementalOptions struct {
	// ChangedSchemas is the list of schema file paths that have changed.
	// If empty, performs full generation.
	ChangedSchemas []string

	// Verbose enables detailed logging
	Verbose bool
}

// GenerateCodeIncremental generates code with content-based file writing.
// Files are only written if their content has changed, preserving mtimes
// for unchanged files and allowing Go's build cache to remain valid.
//
// The changedSchemas parameter is used for logging purposes to show what
// triggered the regeneration. The actual optimization comes from the
// content-based file writing in templates.write().
func GenerateCodeIncremental(data *Data, opts IncrementalOptions) error {
	if !data.Config.Exec.IsDefined() {
		return fmt.Errorf("missing exec config")
	}

	if opts.Verbose && len(opts.ChangedSchemas) > 0 {
		// Build dependency graph for informational logging
		depGraph := BuildDependencyGraph(data.Config.Schema)
		affectedSchemas := depGraph.GetAffectedSchemas(opts.ChangedSchemas)
		affectedTypes := depGraph.GetTypesForSchemas(affectedSchemas)
		log.Printf("[incremental] Changed: %d, Affected: %d schemas, %d types\n",
			len(opts.ChangedSchemas), len(affectedSchemas), len(affectedTypes))
	}

	// Perform full generation - the content-based write() in templates
	// will skip writing files whose content hasn't changed, preserving
	// their mtime and keeping Go's build cache valid.
	return GenerateCode(data)
}
