#!/usr/bin/env bash

set -x

export GOOS=linux
export GOARCH=amd64
./build.sh
