package shared

import "embed"

//go:embed "schema.graphqls"
var sourcesFS embed.FS
