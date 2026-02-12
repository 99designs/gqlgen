# RESOLVERGEN KNOWLEDGE BASE

## OVERVIEW
`plugin/resolvergen/` generates resolver scaffolding with configurable layouts (`single-file`, `follow-schema`), template overrides, and preserve/merge behavior.

## STRUCTURE
```text
plugin/resolvergen/
├── resolver.go            # Core resolver generation flow
├── config.go              # Resolver config interpretation
├── resolver.gotpl         # Default resolver template
└── testdata/              # Scenario matrix with expected outputs
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Layout behavior | `resolver.go` | `GenerateCode` layout branching |
| Preserve semantics | `resolver.go` | `PreserveResolver` and rewrite behavior |
| Template override behavior | `testdata/resolvertemplate/` | Custom template path coverage |
| Output naming behavior | `testdata/filetemplate/` | Custom filename template scenarios |

## CONVENTIONS
- Resolver behavior changes require updating matching `testdata/*/out` artifacts.
- Keep preserve-mode behavior stable; existing resolver implementations are intentionally retained.
- Use fixture-specific tests for targeted regressions before full suite.

## ANTI-PATTERNS (RESOLVERGEN)
- Editing generated resolver outputs directly instead of regenerating.
- Breaking preserve mode while only testing non-preserve paths.
- Changing file/template naming behavior without fixture updates.

## COMMANDS
```bash
go test ./plugin/resolvergen/...
go test ./plugin/resolvergen -run TestLayout
```
