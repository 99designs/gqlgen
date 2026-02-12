# PLUGIN KNOWLEDGE BASE

## OVERVIEW
`plugin/` contains optional codegen behaviors. Plugins implement shared interfaces from `plugin/plugin.go` and hook into schema mutation, config mutation, and code generation phases.

## STRUCTURE
```text
plugin/
├── plugin.go          # Plugin contracts (Plugin/ConfigMutator/CodeGenerator)
├── modelgen/          # Go model generation plugin
├── resolvergen/       # Resolver file generation plugin
├── federation/        # Apollo federation plugin and runtime glue
├── servergen/         # Init/server scaffolding plugin
└── stubgen/           # Stub generation helpers
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Plugin interface behavior | `plugin.go` | Source of lifecycle contracts |
| Model output behavior | `modelgen/` | Generated model conventions + hooks |
| Resolver file behavior | `resolvergen/` | Layout, template, preserve semantics |
| Federation behavior | `federation/` | Directive/schema injection + templates |

## CONVENTIONS
- Treat each plugin as a distinct domain; avoid cross-plugin coupling.
- Keep plugin testdata fixtures updated with behavior changes.
- Preserve plugin-specific docs/tests near plugin code.

## ANTI-PATTERNS (PLUGINS)
- Implementing plugin behavior in unrelated package paths.
- Editing committed generated fixtures by hand.
- Bundling behavior changes for multiple plugins in one opaque refactor.

## SUBDIRECTORY AGENTS
- `plugin/modelgen/AGENTS.md`
- `plugin/resolvergen/AGENTS.md`
- `plugin/federation/AGENTS.md`
