#!/bin/bash
#
# This was adapted from https://github.com/dgraph-io/dgraph/blob/master/wiki/scripts/build.sh
#

set -e

GREEN='\033[32;1m'
RESET='\033[0m'
HOST=https://gqlgen.com

VERSIONS_ARRAY=(
    'v0.10.2'
	'master'
    'v0.9.3'
    'v0.8.3'
    'v0.7.2'
    'v0.6.0'
    'v0.5.1'
    'v0.4.4'
)

joinVersions() {
	versions=$(printf ",%s" "${VERSIONS_ARRAY[@]}")
	echo "${versions:1}"
}

function version { echo "$@" | gawk -F. '{ printf("%03d%03d%03d\n", $1,$2,$3); }'; }

rebuild() {
	echo -e "$(date) $GREEN Updating docs for branch: $1.$RESET"


	VERSION_STRING=$(joinVersions)
	export CURRENT_VERSION=${1}
	export VERSIONS=${VERSION_STRING}

    hugo --quiet --destination="public/$CURRENT_VERSION" --baseURL="$HOST/$CURRENT_VERSION"

    if [[ $1 == "${VERSIONS_ARRAY[0]}" ]]; then
        hugo --quiet --destination=public/ --baseURL="$HOST/"
    fi
}


currentBranch=$(git rev-parse --abbrev-ref HEAD)

git fetch origin

for version in "${VERSIONS_ARRAY[@]}" ; do
    git checkout $branch
    rebuild "$version"
done

git checkout -q "$currentBranch"

