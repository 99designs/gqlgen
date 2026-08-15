package graph

// This file does not compile, on purpose.
//
// It exists so that TestGenerate can assert that Generate's validation step really
// compiles its output. Validation used to build an import-path pattern, whose "..."
// wildcard skips testdata directories, so it matched no packages and passed silently.
// If validation regresses to that, this fixture starts generating without error and
// TestGenerateValidatesOutput fails.
//
// Nothing else notices this file: the repository's own ./... patterns skip testdata.
var _ ThisTypeDoesNotExist
