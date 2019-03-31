#!/bin/bash

set -eu

echo "### go code formatting"
go fmt ./...

echo "### go generating"
go generate ./...

if [[ $(git --no-pager diff) ]] ; then
    echo "you need to run `go fmt` or `go generate`"
    git --no-pager diff
    exit 1
fi

echo "### running testsuite"
go test -race ./...

echo "### linting"
golangci-lint run
