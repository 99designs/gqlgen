# Federation plugin

Add support for graphql federation in your graphql Go server!

TODO(miguel): add details.

# Tests
There are several different tests. Some will process the configuration file directly.  You can see those in the `federation_test.go`.  There are also tests for entity resolvers, which will simulate requests from a federation server like Apollo Federation.

Running entity resolver tests.
1. Go to `plugin/federation`
2. Run the command `go run github.com/99designs/gqlgen --config testdata/entityresolver/gqlgen.yml`
3. Run the tests with `go test ./...`.

# Architecture

TODO(miguel): add details.
