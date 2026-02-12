# MODELGEN KNOWLEDGE BASE

## OVERVIEW
`plugin/modelgen/` generates Go model types from schema/config mappings. It supports hooks, tag controls, and many fixture variants to validate output permutations.

## STRUCTURE
```text
plugin/modelgen/
├── models.go              # Core model generation plugin logic
├── models.gotpl           # Model template
├── testdata/              # Input schemas + gqlgen.yml variants
├── out*/                  # Committed generated outputs per variant
└── *_test.go              # Behavior and interface embedding tests
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Plugin behavior/hook points | `models.go` | `MutateConfig`, hooks, field handling |
| Template rendering shape | `models.gotpl` | Generated struct/tag layout |
| Scenario inputs | `testdata/*.yml`, `testdata/**/*.yml` | Variant matrix |
| Golden outputs | `out*/generated*.go` | Expected generator results |

## CONVENTIONS
- Source-of-truth is plugin code + testdata config, not `out*/` artifacts.
- Keep variant coverage focused: update only affected `out*` directories.
- Validate with local package tests after any hook/template change.

## ANTI-PATTERNS (MODELGEN)
- Manual edits to `out*/generated*.go` files.
- Template updates without fixture regeneration.
- Silent behavior changes across JSON/omitzero variants.

## COMMANDS
```bash
go test ./plugin/modelgen/...
go test ./plugin/modelgen -run Test
```
