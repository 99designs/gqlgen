#!/bin/bash

set -eu

echo "### go generating"
go generate $(go list ./... | grep -v codegen/tests)

if [[ $(git --no-pager diff) ]] ; then
    echo "you need to run go generate"
    git --no-pager diff
    exit 1
fi

echo "### running testsuite"
go test -race $(go list ./... | grep -v codegen/tests)

echo "### linting"
gometalinter --vendor ./...
