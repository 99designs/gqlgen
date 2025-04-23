#!/bin/bash

set -euo pipefail

export GOVERSION="$(cat GOVERSION.txt)"
export GOTOOLCHAIN="go${GOVERSION}"
go get go@${GOVERSION} || true 
go get toolchain@none || true
go mod tidy || true

STATUS=$( git status --porcelain go.mod go.sum )
if [ ! -z "$STATUS" ]; then
    echo "Running go mod tidy modified go.mod and/or go.sum"
    echo "Please run the following then make a git commit:"
    echo "go get go@${GOVERSION}"
    echo "go get toolchain@none"
    echo "go mod tidy"
    echo "export GOTOOLCHAIN=${GOTOOLCHAIN}"
    exit 1
fi
exit 0