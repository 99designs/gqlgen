#!/bin/bash
# This script should update all the modules to the latest version in preparation for a new release
set -euo pipefail

cd "$(dirname "$0")/.."

go_version=$(cat .github/workflows/GOVERSION.txt)

# in_each_module CMD... runs CMD in every module directory, stopping at the first
# failure (find -execdir would carry on and let the error scroll past). Each step
# runs across all modules before the next starts, because the nested modules
# replace gqlgen with the root and must be tidied after the root is updated.
in_each_module() {
	find . -name go.mod -print0 | sort -z | while IFS= read -r -d '' mod; do
		(cd "$(dirname "$mod")" && "$@")
	done
}

in_each_module gomajor get all
in_each_module go mod edit -go="$go_version"
in_each_module go mod tidy
in_each_module go generate ./...
