#!/usr/bin/env bash

set -x

export CGO_ENABLED=0
export GOOS=linux
./build.sh
