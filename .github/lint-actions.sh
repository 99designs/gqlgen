#!/bin/bash

function is_bin_in_path {
  builtin type -P "$1" &> /dev/null
}

export GOBIN="$HOME/go/bin"
! is_bin_in_path yamllint && go install -v github.com/wasilibs/go-yamllint/cmd/yamllint@latest
! is_bin_in_path actionlint && go install -v github.com/rhysd/actionlint/cmd/actionlint@latest
! is_bin_in_path shellcheck && go install -v github.com/wasilibs/go-shellcheck/cmd/shellcheck@latest
! is_bin_in_path ghalint && go install -v github.com/suzuki-shunsuke/ghalint/cmd/ghalint@latest
export PATH="$GOBIN:$PATH"
# Note that due to the sandboxing of the filesystem when using Wasm,
# currently only files that descend from the current directory when executing the tool
# are accessible to it, i.e., ../yaml/my.yaml or /separate/root/my.yaml will not be found.
yamllint -c ./linters/.yamllint.yaml .

# https://www.shellcheck.net/wiki/SC2086 https://www.shellcheck.net/wiki/SC2129
export SHELLCHECK_OPTS='-e SC2086 -e SC2129'
actionlint -config-file=./linters/actionlint.yaml -shellcheck="$(which shellcheck)"
cd ..
ghalint run
