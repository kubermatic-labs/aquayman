#!/usr/bin/env bash

set -euo pipefail

cd $(dirname $0)/../..

git remote add origin git@github.com:kubermatic-labs/aquayman.git

export GITHUB_TOKEN=$(cat /etc/github/oauth | tr -d '\n')
goreleaser release
