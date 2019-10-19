#!/usr/bin/env bash

set -x

ROOT_DIR=`pwd`

if [ ! -d ${ROOT_DIR}/bin ]; then
  mkdir bin
fi

GitCommitLog=`git log --pretty=oneline -n 1`
# 将 log 原始字符串中的单引号替换成双引号
GitCommitLog=${GitCommitLog//\'/\"}

GitStatus=`git status -s`
BuildTime=`date +'%Y.%m.%d.%H%M%S'`
BuildGoVersion=`go version`

LDFlags=" \
    -X 'github.com/q191201771/naza/pkg/bininfo.GitCommitLog=${GitCommitLog}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.GitStatus=${GitStatus}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.BuildTime=${BuildTime}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.BuildGoVersion=${BuildGoVersion}' \
"

cd ${ROOT_DIR}/app/lals && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/lals &&
cd ${ROOT_DIR}/app/flvfile2rtmppush && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/flvfile2rtmppush &&
cd ${ROOT_DIR}/app/flvfile2es && go build -o ${ROOT_DIR}/bin/flvfile2es &&
cd ${ROOT_DIR}/app/httpflvpull && go build -o ${ROOT_DIR}/bin/httpflvpull &&
cd ${ROOT_DIR}/app/modflvfile && go build -o ${ROOT_DIR}/bin/modflvfile &&
cd ${ROOT_DIR}/app/rtmppull && go build -o ${ROOT_DIR}/bin/rtmppull &&
${ROOT_DIR}/bin/lals -v &&
ls -lrt ${ROOT_DIR}/bin &&
echo 'build done.'
