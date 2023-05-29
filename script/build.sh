#!/usr/bin/env bash

#set -x
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct
export GO111MODULE=on
export GOPROXY=https://goproxy.cn,https://goproxy.io,direct
THIS_FILE=$(readlink -f $0)
# readlink have no -f param in some macos
if [ $? -ne 0 ]; then
  cd `dirname $0`
  TARGET_FILE=`basename $0`
  PHYS_DIR=`pwd -P`
  THIS_FILE=$PHYS_DIR/$TARGET_FILE
  cd -
fi
THIS_DIR=$(dirname $THIS_FILE)
ROOT_DIR=${THIS_DIR}/..

OUT_DIR=${ROOT_DIR}/bin

if [ ! -d ${OUT_DIR} ]; then
  mkdir ${OUT_DIR}
fi

#GitTag=`git tag --sort=version:refname | tail -n 1`
GitTag=`git tag | sort -V | tail -n 1`
GitCommitLog=`git log --pretty=oneline -n 1`
# 将 log 原始字符串中的单引号替换成双引号
GitCommitLog=${GitCommitLog//\'/\"}

GitStatus=`git status -s`
BuildTime=`date +'%Y.%m.%d.%H%M%S'`
BuildGoVersion=`go version`
#WebUITpl=`cat lal.html`

LDFlags=" \
    -X 'github.com/q191201771/naza/pkg/bininfo.GitTag=${GitTag}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.GitCommitLog=${GitCommitLog}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.GitStatus=${GitStatus}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.BuildTime=${BuildTime}' \
    -X 'github.com/q191201771/naza/pkg/bininfo.BuildGoVersion=${BuildGoVersion}' \
"

#-X 'github.com/q191201771/lal/pkg/logic.webUITpl=${WebUITpl}' \

echo "build" ${ROOT_DIR}/app/lalserver "..."
cd ${ROOT_DIR}/app/lalserver && go build -ldflags "$LDFlags" -o ${OUT_DIR}/lalserver
#cd ${ROOT_DIR}/app/lalserver && go build -race -ldflags "$LDFlags" -o ${OUT_DIR}/lalserver.debug

for file in `ls ${ROOT_DIR}/app/demo`
do
  if [ -d ${ROOT_DIR}/app/demo/${file} ]; then
    echo "build" ${ROOT_DIR}/app/demo/${file} "..."
    cd ${ROOT_DIR}/app/demo/${file} && go build -ldflags "$LDFlags" -o ${OUT_DIR}/${file}
  fi
done

if [ -d "./playground" ]; then
  for file in `ls ${ROOT_DIR}/playground`
  do
    if [ -d ${ROOT_DIR}/playground/${file} ]; then
      echo "build" ${ROOT_DIR}/playgound/${file} "..."
      cd ${ROOT_DIR}/playground/${file} && go build -ldflags "$LDFlags" -o ${OUT_DIR}/${file}
    fi
  done
fi

${OUT_DIR}/lalserver -v &&
ls -lrt ${OUT_DIR} &&
echo 'build done.'
