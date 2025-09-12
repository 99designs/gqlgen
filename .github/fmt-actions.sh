#!/bin/bash
# fmt-actions - reformats github actions
# intended to be invoked from this directory
#
# see https://til.simonwillison.net/yaml/yamlfmt

function is_bin_in_path {
  builtin type -P "$1" &> /dev/null
}

export GOBIN="$HOME/go/bin"
mkdir -p "$GOBIN"
# we installed go binaries to $GOBIN
# so we ensure that is in the PATH and takes precedence
export PATH="$GOBIN:$PATH"
! is_bin_in_path yamlfmt && GOBIN=$HOME/go/bin go install -v github.com/google/yamlfmt/cmd/yamlfmt@latest

# -formatter indentless_arrays=true,retain_line_breaks=true
yamlfmt \
  -conf ./linters/.yamlfmt.yaml ./workflows/*.y*ml

# -formatter indentless_arrays=true,retain_line_breaks=true
yamlfmt \
  -conf ./linters/.yamlfmt.yaml ./linters/*.y*ml

# -formatter indentless_arrays=true,retain_line_breaks=true
yamlfmt \
  -conf ./linters/.yamlfmt.yaml ./*.y*ml


