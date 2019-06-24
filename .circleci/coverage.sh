#!/bin/bash

set -eu

go test -coverprofile=/tmp/coverage.out -coverpkg=./... $(go list github.com/99designs/gqlgen/... | grep -v server)
goveralls -coverprofile=/tmp/coverage.out -service=circle-ci -repotoken=$REPOTOKEN -ignore='example/*/*,example/*/*/*,integration/*,integration/*/*,codegen/testserver/*'
