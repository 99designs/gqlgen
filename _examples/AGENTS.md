# EXAMPLES KNOWLEDGE BASE

## OVERVIEW
`_examples/` contains runnable gqlgen sample apps. Each example demonstrates a focused pattern (transport, dataloader, federation, project structure, scalar handling, etc.).

## STRUCTURE
```text
_examples/
├── todo/                      # Minimal end-to-end example
├── dataloader/                # N+1 mitigation pattern
├── federation/                # Multi-service federation samples
├── websocket-initfunc/        # WS init hook pattern
├── large-project-structure/   # Multi-module example layout
└── ...                        # Many independent scenario demos
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Basic gqlgen pattern | `todo/`, `uuid/` | Smallest reproducible examples |
| Advanced transport behavior | `websocket-initfunc/`, `chat/` | WS/SSE patterns |
| Architecture layout examples | `large-project-structure/` | Shared/main/integration module split |
| Federation demos | `federation/` | Subgraph-style composition |

## CONVENTIONS
- Treat each example as self-contained; many have their own `go.mod`.
- Preserve generated files for reproducibility in examples.
- Prefer small targeted edits in one example at a time.

## ANTI-PATTERNS (EXAMPLES)
- Applying one example's conventions globally without verification.
- Mixing unrelated example updates in one commit.
- Editing generated example files without running generation workflow.

## COMMANDS
```bash
(cd _examples && go test -race ./...)
go generate ./...
```
