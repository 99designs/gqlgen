#!/bin/bash
# Run `go generate ./...` in every nested module under _examples.
#
# _examples is its own module, and `go generate ./...` stops at module
# boundaries, so the sweep in the root main.go's //go:generate never reaches the
# examples that carry their own go.mod. Their //go:generate directives are
# therefore never executed by anything, and check-generate cannot see them
# drift: the modules silently rot until somebody builds one by hand.
set -euo pipefail

cd "$(dirname "$0")"

find . -mindepth 2 -name go.mod -print0 | sort -z | while IFS= read -r -d '' mod; do
	dir=${mod#./}
	dir=${dir%/go.mod}
	echo "==> _examples/$dir"
	(cd "$dir" && go generate ./...)
done
