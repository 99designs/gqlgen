#!/bin/bash

set -euo pipefail

hugo
aws-vault exec platform -- aws s3 sync public s3://gqlgen.com
