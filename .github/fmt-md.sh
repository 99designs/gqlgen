#!/bin/bash
# fmt-md - Intended to be run from root directory of repository
# to format markdown to pass linting rules.
# Works on all machines, but will install missing requirements
# using homebrew as those who use linux will have no need because:
# 1. Linux users will manage their own requirement installation
# 2. Linux users will not make markdown formatting mistakes :)

# Requirements:
# uv - to install python tools idempotently
# go - any recent version
# shfmt - mvdan.cc/sh/v3/cmd/shfmt
# mdformat and extensions

function is_bin_in_path {
  builtin type -P "$1" &> /dev/null
}

export GOBIN="$HOME/go/bin"
mkdir -p "$GOBIN"
# uv installs things to $HOME/.local/bin
# we installed go binaries to $GOBIN
# so we ensure those both are in the PATH and take precedence
export PATH="$HOME/.local/bin:$GOBIN:$PATH"
! is_bin_in_path uv && brew install uv
! is_bin_in_path shfmt && go install mvdan.cc/sh/v3/cmd/shfmt@latest
! is_bin_in_path mdformat && uv tool install --with mdformat-gfm --with mdformat-shfmt --with mdformat-tables --with mdformat-toc --with mdformat-config --with mdformat-gofmt mdformat

# clean all Script files (possibly makes mistakes?):
# find .. -name '*.sh' -type f -print0 | xargs -0 -n1 -P4 shfmt -bn -ci -d -i 2 -ln bash -s -sr

# ensure all files have trailing line endings
# find -type f | while read f; do tail -n1 $f | read -r _ || echo >> $f; done
# clean all markdown files 
find . -type d -name node_modules -prune -o -name '*.md' -type f -print0 | xargs -0 -n1 -P4 mdformat --wrap keep --number
