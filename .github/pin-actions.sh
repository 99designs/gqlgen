#!/bin/bash
# pin-actions.sh - pins all actions to Git SHA1, run from repo root

function is_bin_in_path {
  builtin type -P "$1" &> /dev/null
}

export GOBIN="$HOME/go/bin"
mkdir -p "$GOBIN"
# we installed go binaries to $GOBIN
# so we ensure that is in the PATH and takes precedence
export PATH="$GOBIN:$PATH"
! is_bin_in_path yamlfmt && GOBIN=$HOME/go/bin go install -v github.com/sethvargo/ratchet@latest

export SED_COMMAND="gsed"
! is_bin_in_path gsed && export SED_COMMAND="sed"

find . -name '*.y*l' | sort -u | grep '.github/workflows' | xargs -I {} ratchet pin '{}'
cd .github
find . -name '*.y*l' -exec ${SED_COMMAND} -i'' 's/ratchet:.*\/.*\@//g' {} \;
./fmt-actions.sh
cd ..
