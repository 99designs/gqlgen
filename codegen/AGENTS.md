# CODEGEN KNOWLEDGE BASE

## OVERVIEW
`codegen/` owns gqlgen executable code generation: config-driven model building, template rendering, and output layout (`single-file`, `follow-schema`, `split-packages`).

## STRUCTURE
```text
codegen/
├── generate.go           # Main exec generation entry
├── generate_split.go     # Split-packages generation flow
├── config/               # Config loading, validation, schema binding
├── templates/            # Shared render helpers
└── testserver/           # Fixture-heavy generation matrix
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Exec layout behavior | `generate.go`, `generate_split.go` | `Exec.Layout` switch is canonical |
| Config parsing/mutation | `config/` | `LoadConfig`, `Init`, exec/model/resolver settings |
| Template emission | `templates/` + `*.gotpl` | Rendering behavior + generated headers |
| Split ownership/sharding | `split_ownership.go` | Cross-file ownership map for split output |
| Regression fixtures | `testserver/` | Golden-style generated artifacts |

## CONVENTIONS
- Keep generator logic and fixture updates in the same PR.
- Prefer minimal changes to templates; they fan out to many fixtures.
- Preserve existing layout semantics unless change is intentional and tested.
- Run focused package tests first (`go test ./codegen/...`) before global suite.

## ANTI-PATTERNS (CODEGEN)
- Hand-editing `*.generated.go` fixtures under `testserver/`.
- Refactoring unrelated helpers while fixing generator output bugs.
- Updating one layout path without parity checks in other layouts.
- Ignoring `split-packages` regressions when touching exec generation.

## COMMANDS
```bash
go test ./codegen/...
go test ./codegen/testserver/...
go generate ./...
.github/workflows/check-generate
```
