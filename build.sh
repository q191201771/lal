#!/usr/bin/env bash

set -x

if [ ! -d "bin" ]; then
  mkdir bin
fi

cd app/lal && \
  go build -ldflags " \
    -X 'github.com/q191201771/lal/pkg/util/bininfo.GitCommitID=`git log --pretty=format:'%h' -n 1`' \
    -X 'github.com/q191201771/lal/pkg/util/bininfo.BuildTime=`date +'%Y.%m.%d.%H%M%S'`' \
    -X 'github.com/q191201771/lal/pkg/util/bininfo.BuildGoVersion=`go version`' \
  " -o ../../bin/lal

cd -
cd app/flvfile2rtmppush && go build -o ../../bin/flvfile2rtmppush

cd -
cd app/modflvfile && go build -o ../../bin/modflvfile
