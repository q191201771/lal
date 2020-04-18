#!/usr/bin/env bash

#set -x

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

if [ -d ${ROOT_DIR}/app/analyse_upstream ]; then
  cd ${ROOT_DIR}/app/analyse_upstream && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/analyse_upstream
fi

if [ -d ${ROOT_DIR}/app/flvfile2es ]; then
  cd ${ROOT_DIR}/app/flvfile2es && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/flvfile2es
fi

if [ -d ${ROOT_DIR}/app/flvfile2rtmppush ]; then
  cd ${ROOT_DIR}/app/flvfile2rtmppush && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/flvfile2rtmppush
fi

if [ -d ${ROOT_DIR}/app/httpflvpull ]; then
  cd ${ROOT_DIR}/app/httpflvpull && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/httpflvpull
fi

if [ -d ${ROOT_DIR}/app/learnhls ]; then
  cd ${ROOT_DIR}/app/learnhls && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/learnhls
fi

if [ -d ${ROOT_DIR}/app/learnmp4 ]; then
  cd ${ROOT_DIR}/app/learnmp4 && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/learnmp4
fi

if [ -d ${ROOT_DIR}/app/modflvfile ]; then
  cd ${ROOT_DIR}/app/modflvfile && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/modflvfile
fi

if [ -d ${ROOT_DIR}/app/rtmppull ]; then
  cd ${ROOT_DIR}/app/modflvfile && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/modflvfile
fi

cd ${ROOT_DIR}/app/lals && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/bin/lals &&
${ROOT_DIR}/bin/lals -v &&
ls -lrt ${ROOT_DIR}/bin &&
echo 'build done.'
