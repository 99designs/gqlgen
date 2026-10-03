# bin/\_tools

A nested Go module that pins the third-party code generators gqlgen runs via
`go:generate`, plus a few one-off helper programs we maintain ourselves.

The directory name starts with `_`, so the Go tool ignores it: `go build ./...`,
`go test ./...` and `go generate ./...` in the root module never descend here,
and the generators' dependencies stay out of the root `go.mod`.

## Pinned generators

| Tool                                                                | Used by                                                  |
| ------------------------------------------------------------------- | -------------------------------------------------------- |
| [`github.com/vektra/mockery/v3`](https://github.com/vektra/mockery) | `graphql/executable_schema.go` (`graphql/.mockery.yaml`) |

Each one is recorded as a `tool` directive in `go.mod`, so the exact version is
locked by `go.sum` rather than resolved from the network at generate time.

## Running a pinned tool

Invoke it from the package that owns the generated file, pointing `-modfile` at
this module:

```go
//go:generate go tool -modfile=../bin/_tools/go.mod mockery --config=.mockery.yaml
```

`-modfile` keeps the working directory where it is, so paths in the tool's own
config file stay relative to the package being generated.

## Bumping a pinned tool

```sh
cd bin/_tools
go get -tool github.com/vektra/mockery/v3@latest
go mod tidy
cd ../..
go generate ./...
```

Commit the resulting `go.mod`, `go.sum` and regenerated files together.
Dependabot also watches this module and opens the same bump as a PR.

## Adding a new tool

Beyond `go get -tool`, give the tool its own `allow` entry in
`.github/dependabot.yml` under the `/bin/_tools` update:

```yaml
allow:
  - dependency-name: github.com/vektra/mockery/v3
    dependency-type: all
```

A `tool` directive records its module as `// indirect` in `go.mod`, and gomod
version updates only consider direct dependencies by default, so a tool without
an `allow` entry is pinned forever and never gets an update PR. See
[dependabot-core#12050](https://github.com/dependabot/dependabot-core/issues/12050).
