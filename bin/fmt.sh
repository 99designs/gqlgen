#!/bin/bash
# Script to format files and regenerate
set -o errexit
set -o nounset
set -o xtrace
set -o pipefail

# set -euxo pipefail is short for:
# set -e, -o errexit: stop the script when an error occurs
# set -u, -o nounset: detects uninitialised variables in your script and exits with an error (including Env variables)
# set -x, -o xtrace: prints every expression before executing it
# set -o pipefail: If any command in a pipeline fails, use that return code for whole pipeline instead of final success


gci write -s standard -s default -s "prefix(github.com/99designs)" --skip-generated .
gofumpt -w .
go generate ./...