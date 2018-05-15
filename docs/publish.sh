#!/bin/bash

set -euo pipefail

hugo
gsutil rsync -d public gs://gqlgen.com
