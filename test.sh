#!/usr/bin/env bash

set -e
echo "" > coverage.txt

for d in $(go list ./... | grep -v vendor | grep lal/pkg); do
    go test -race -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done

# go test -race -coverprofile=profile.out -covermode=atomic && go tool cover -html=profile.out -o coverage.html && open coverage.html
# go test -test.bench=".*"
