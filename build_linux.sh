#!/usr/bin/env bash

set -x

ROOT_DIR=`pwd`
echo ${ROOT_DIR}/bin

if [ ! -d ${ROOT_DIR}/bin ]; then
  mkdir bin
fi

cd ${ROOT_DIR}/app/lal && \
  GOOS=linux GOARCH=amd64 go build -ldflags " \
    -X 'github.com/q191201771/lal/pkg/util/bininfo.GitCommitID=`git log --pretty=format:'%h' -n 1`' \
    -X 'github.com/q191201771/lal/pkg/util/bininfo.BuildTime=`date +'%Y.%m.%d.%H%M%S'`' \
    -X 'github.com/q191201771/lal/pkg/util/bininfo.BuildGoVersion=`go version`' \
  " -o ${ROOT_DIR}/bin/lal

cd ${ROOT_DIR}/app/flvfile2es && GOOS=linux GOARCH=amd64 go build -o ${ROOT_DIR}/bin/flvfile2es

cd ${ROOT_DIR}/app/flvfile2rtmppush && GOOS=linux GOARCH=amd64 go build -o ${ROOT_DIR}/bin/flvfile2rtmppush

cd ${ROOT_DIR}/app/httpflvpull && GOOS=linux GOARCH=amd64 go build -o ${ROOT_DIR}/bin/httpflvpull

cd ${ROOT_DIR}/app/modflvfile && GOOS=linux GOARCH=amd64 go build -o ${ROOT_DIR}/bin/modflvfile

cd ${ROOT_DIR}/app/rtmppull && GOOS=linux GOARCH=amd64 go build -o ${ROOT_DIR}/bin/rtmppull
