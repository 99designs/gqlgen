# GRAPHQL RUNTIME KNOWLEDGE BASE

## OVERVIEW
`graphql/` is runtime execution surface: schema execution, handler orchestration, transport implementations (HTTP/WS/SSE), extension middleware, and introspection.

## STRUCTURE
```text
graphql/
├── handler/transport/      # POST/GET/WS/SSE transport adapters
├── handler/extension/      # APQ, introspection, complexity middleware
├── executor/               # Execution engine internals
├── introspection/          # Introspection resolvers
└── *.go                    # Scalars, context helpers, runtime primitives
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Handler assembly | `handler/server.go` | `AddTransport`, `Use`, defaults |
| Transport behavior | `handler/transport/*.go` | One file per transport variant |
| Transport regressions | `handler/transport/*_test.go` | Canonical expected behavior |
| Extension lifecycle | `handler/extension/*.go` | Request/operation middleware hooks |
| Shard runtime wiring | `executor/shardruntime/` | Registry-driven execution handlers |

## CONVENTIONS
- Keep transport changes paired with transport tests in same subtree.
- Maintain stable response/error behavior; tests assert exact payload shapes.
- Keep extension behavior explicit; avoid hidden global state.
- Prefer additive transport options over changing defaults silently.

## ANTI-PATTERNS (GRAPHQL)
- Changing default handler behavior without updating example/test coverage.
- Mixing transport parsing changes with unrelated scalar/runtime refactors.
- Updating websocket protocol handling without both protocol path tests.

## COMMANDS
```bash
go test ./graphql/...
go test ./graphql/handler/transport/...
go test ./graphql/handler/extension/...
```
