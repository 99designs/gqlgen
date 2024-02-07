#!/usr/bin/env sh
cd ./_examples
go generate ./... || return 0
