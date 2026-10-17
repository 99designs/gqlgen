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

## Helper programs

| Program              | Used by                                                                     |
| -------------------- | --------------------------------------------------------------------------- |
| `apollo-sandbox-sri` | Run by hand to refresh the Apollo Sandbox SRI hash in `graphql/playground`  |
| `changelogabsref`    | `bin/update_gqlparser.sh`, to turn gqlparser release notes into the PR body |

`changelogabsref` reads a GoReleaser release body on stdin and prints only
its changelog, with commit SHAs dropped and `#123` rewritten as
`[owner/repo#123](https://github.com/owner/repo/pull/123)`. Otherwise a gqlgen
PR body would link gqlparser's PR numbers to gqlgen's own PRs:

```sh
gh release view v2.5.63 -R vektah/gqlparser --json body --jq .body |
    go run -C bin/_tools ./changelogabsref https://github.com/vektah/gqlparser
```

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
