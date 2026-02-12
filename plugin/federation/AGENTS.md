# FEDERATION PLUGIN KNOWLEDGE BASE

## OVERVIEW
`plugin/federation/` adds federation directives/types and generates federation runtime glue. It supports federation version-specific behavior and explicit/computed requires flows.

## STRUCTURE
```text
plugin/federation/
├── federation.go          # Core federation plugin logic
├── federation.gotpl       # Main generated federation template
├── requires.gotpl         # Explicit-requires generation template
├── fedruntime/            # Federation runtime helper package
├── fieldset/              # Directive field parsing helpers
└── testdata/              # Scenario matrix with generated outputs
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Directive/schema injection | `federation.go` | Config/schema mutation path |
| Requires behavior | `federation.go`, `requires.gotpl` | explicit/computed requires variants |
| Entity resolver semantics | `testdata/entityresolver/` | `@entityResolver(multi: true)` cases |
| Federation v2 behavior | `testdata/federation2/` | Version-specific coverage |

## CONVENTIONS
- Keep federation plugin changes paired with scenario fixture updates.
- Validate both regular federation and explicit/computed requires paths.
- Keep generated testdata outputs reproducible via go generate/test workflow.

## ANTI-PATTERNS (FEDERATION)
- Editing `testdata/*/generated/*` outputs by hand.
- Shipping version-specific changes without opposite-version regression checks.
- Changing directive behavior without updating scenario matrix.

## COMMANDS
```bash
go test ./plugin/federation/...
(cd plugin/federation && go generate)
```
