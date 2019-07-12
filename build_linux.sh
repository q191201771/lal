#!/usr/bin/env bash

set -x

if [ ! -d "bin" ]; then
  mkdir bin
fi

cd app/lal && \
  GOOS=linux GOARCH=amd64 go build -ldflags " \
    -X 'github.com/q191201771/lal/pkg/bininfo.BuildTime=`date +'%Y.%m.%d.%H%M%S'`' \
    -X 'github.com/q191201771/lal/pkg/bininfo.GitCommitID=`git log --pretty=format:'%h' -n 1`' \
    -X 'github.com/q191201771/lal/pkg/bininfo.GoVersion=`go version`' \
  " -o ../../bin/lal_linux
