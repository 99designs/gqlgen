#!/bin/bash

set -eu

echo "### running jest integration spec"
./node_modules/.bin/jest --color


echo "### validating introspected schema"
./node_modules/.bin/graphql get-schema

if ! diff <(tail -n +3 schema-expected.graphql) <(tail -n +3 schema-fetched.graphql) ; then
    echo "The expected schema has changed, you need to update schema-expected.graphql with any expected changes"
    exit 1
fi


