#!/usr/bin/env bash

set -x

export CGO_ENABLED=0
export GOOS=linux
export GOARCH=arm64
./build.sh
