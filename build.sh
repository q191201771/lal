#!/usr/bin/env bash

#set -x

ROOT_DIR=`pwd`
OUT_DIR=bin

if [ ! -d ${ROOT_DIR}/${OUT_DIR} ]; then
  mkdir ${ROOT_DIR}/${OUT_DIR}
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

for file in `ls ${ROOT_DIR}/app`
do
  if [ -d ${ROOT_DIR}/app/${file} ]; then
    echo "build" ${ROOT_DIR}/app/${file} "..."
    cd ${ROOT_DIR}/app/${file} && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/${file}
  fi
done

for file in `ls ${ROOT_DIR}/playground`
do
  if [ -d ${ROOT_DIR}/playground/${file} ]; then
    echo "build" ${ROOT_DIR}/playgound/${file} "..."
    cd ${ROOT_DIR}/playground/${file} && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/${file}
  fi
done


${ROOT_DIR}/${OUT_DIR}/lals -v &&
ls -lrt ${ROOT_DIR}/${OUT_DIR} &&
echo 'build done.'
