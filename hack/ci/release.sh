#!/usr/bin/env bash
#
# This script is creating release binaries and
# Docker images via goreleaser. It's meant to
# run in the Kubermatic CI environment only,
# as it requires GitHub and quay.io credentials.

set -euo pipefail

cd $(dirname $0)/../..

git remote add origin git@github.com:kubermatic-labs/aquayman.git
export GITHUB_TOKEN=$(cat /etc/github/oauth | tr -d '\n')

goreleaser release
