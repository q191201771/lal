#!/usr/bin/env bash

#set -x

ROOT_DIR=`pwd`
OUT_DIR=bin

if [ ! -d ${ROOT_DIR}/${OUT_DIR} ]; then
  mkdir ${ROOT_DIR}/${OUT_DIR}
fi

#GitTag=`git tag --sort=version:refname | tail -n 1`
GitTag=`git tag | sort -V | tail -n 1`
GitCommitLog=`git log --pretty=oneline -n 1`
# 将 log 原始字符串中的单引号替换成双引号
GitCommitLog=${GitCommitLog//\'/\"}

GitStatus=`git status -s`
BuildTime=`date +'%Y.%m.%d.%H%M%S'`
BuildGoVersion=`go version`

LDFlags=" \
    -X 'github.com/q191201771/naza/pkg/bininfo.GitTag=${GitTag}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.GitCommitLog=${GitCommitLog}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.GitStatus=${GitStatus}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.BuildTime=${BuildTime}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.BuildGoVersion=${BuildGoVersion}' \
"

echo "build" ${ROOT_DIR}/app/lalserver "..."
cd ${ROOT_DIR}/app/lalserver && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/lalserver
#cd ${ROOT_DIR}/app/lalserver && go build -race -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/lalserver.debug

for file in `ls ${ROOT_DIR}/app/demo`
do
  if [ -d ${ROOT_DIR}/app/demo/${file} ]; then
    echo "build" ${ROOT_DIR}/app/demo/${file} "..."
    cd ${ROOT_DIR}/app/demo/${file} && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/${file}
  fi
done

if [ -d "./playground" ]; then
  for file in `ls ${ROOT_DIR}/playground`
  do
    if [ -d ${ROOT_DIR}/playground/${file} ]; then
      echo "build" ${ROOT_DIR}/playgound/${file} "..."
      cd ${ROOT_DIR}/playground/${file} && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/${file}
    fi
  done
fi

${ROOT_DIR}/${OUT_DIR}/lalserver -v &&
ls -lrt ${ROOT_DIR}/${OUT_DIR} &&
echo 'build done.'
