#!/usr/bin/env bash

set -x

ROOT_DIR=`pwd`
echo ${ROOT_DIR}/bin

if [ ! -d ${ROOT_DIR}/bin ]; then
  mkdir bin
fi

#GitCommitID=`git log --pretty=format:'%h' -n 1`
GitCommitLog=`git log --pretty=oneline -n 1`
GitStatus=`git status -s`
BuildTime=`date +'%Y.%m.%d.%H%M%S'`
BuildGoVersion=`go version`

cd ${ROOT_DIR}/app/lal && \
  go build -ldflags " \
    -X 'github.com/q191201771/nezha/pkg/bininfo.GitCommitLog=${GitCommitLog}' \
    -X 'github.com/q191201771/nezha/pkg/bininfo.GitStatus=${GitStatus}' \
    -X 'github.com/q191201771/nezha/pkg/bininfo.BuildTime=${BuildTime}' \
    -X 'github.com/q191201771/nezha/pkg/bininfo.BuildGoVersion=${BuildGoVersion}' \
  " -o ${ROOT_DIR}/bin/lal

cd ${ROOT_DIR}/app/flvfile2es && go build -o ${ROOT_DIR}/bin/flvfile2es

cd ${ROOT_DIR}/app/flvfile2rtmppush && go build -o ${ROOT_DIR}/bin/flvfile2rtmppush

cd ${ROOT_DIR}/app/httpflvpull && go build -o ${ROOT_DIR}/bin/httpflvpull

cd ${ROOT_DIR}/app/modflvfile && go build -o ${ROOT_DIR}/bin/modflvfile

cd ${ROOT_DIR}/app/rtmppull && go build -o ${ROOT_DIR}/bin/rtmppull

${ROOT_DIR}/bin/lal -v
ls -lrt ${ROOT_DIR}/bin
