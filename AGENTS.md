# Repository Guidelines

## Project Structure & Module Organization
`main.go` is the CLI entrypoint. Core runtime code lives in `graphql/`. Code generation logic lives in `codegen/`, with generated-server test fixtures in `codegen/testserver/`. Plugin implementations are under `plugin/` (for example `federation/`, `modelgen/`, `resolvergen/`). The test client lives in `client/`, and internal-only helpers are in `internal/`.

Examples and fixtures are important in this repository: `_examples/` contains runnable samples, `integration/` contains Node+Go integration checks, `testdata/` stores generator fixtures, and `init-templates/` contains init scaffolding templates.

## Build, Test, and Development Commands
- `go test -race ./...` runs core package tests.
- `(cd _examples && go test -race ./...)` runs example test suites.
- `go generate ./...` regenerates generated code and test artifacts.
- `.github/workflows/check-fmt` enforces formatting parity with CI (`go fmt` in root and `_examples`).
- `.github/workflows/check-generate` fails if generated files are out of date.
- `golangci-lint run ./...` runs lint checks locally.
- `cd integration && npm ci && cd .. && .github/workflows/check-integration` runs introspection + integration checks.

## Coding Style & Naming Conventions
Follow `.editorconfig`: LF endings, trimmed trailing whitespace, and final newline. Use tabs for `*.go` and `*.gotpl`. Keep Go package names lowercase; exported identifiers use `CamelCase`. Prefer small, focused files and keep generated outputs committed whenever schema/config/codegen inputs change.

## Testing Guidelines
Keep tests adjacent to implementation as `*_test.go`, with `TestXxx` function names. Prefer unit tests first; full codegen/binary-path tests are slower and should be added only when unit-level coverage is insufficient. For integration/introspection changes, validate through the `integration/` workflow and expected schema diff checks.

## Commit & Pull Request Guidelines
Use short, imperative commit messages; conventional prefixes like `chore:`, `fix:`, and `refactor:` align with history. Include tests with bug fixes. For non-trivial features, open a proposal issue first and target the `next` branch. PRs should clearly describe behavior changes and include minimal schema/config reproduction snippets when relevant.
