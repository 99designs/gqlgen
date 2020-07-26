#!/bin/bash

set -euo pipefail

mkdir /tmp/golangci
curl -sL --fail https://github.com/golangci/golangci-lint/releases/download/v1.29.0/golangci-lint-1.29.0-linux-amd64.tar.gz | tar zxv --strip-components=1 --dir=/tmp/golangci

/tmp/golangci/golangci-lint run
