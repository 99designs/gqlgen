#!/bin/bash

set -euo pipefail

hugo
gsutil -m rsync -dr public gs://gqlgen.com
