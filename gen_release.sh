#!/usr/bin/env bash

#set -x

ROOT_DIR=`pwd`
OUT_DIR=release

rm -rf ${ROOT_DIR}/${OUT_DIR}
mkdir -p ${ROOT_DIR}/${OUT_DIR}/linux/bin
mkdir -p ${ROOT_DIR}/${OUT_DIR}/linux/conf
mkdir -p ${ROOT_DIR}/${OUT_DIR}/macos/bin
mkdir -p ${ROOT_DIR}/${OUT_DIR}/macos/conf
mkdir -p ${ROOT_DIR}/${OUT_DIR}/windows/bin
mkdir -p ${ROOT_DIR}/${OUT_DIR}/windows/conf

cp conf/lals.conf.json ${ROOT_DIR}/${OUT_DIR}/linux/conf
cp conf/lals.conf.json ${ROOT_DIR}/${OUT_DIR}/macos/conf
cp conf/lals.conf.json ${ROOT_DIR}/${OUT_DIR}/windows/conf

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

export CGO_ENABLED=0
export GOARCH=amd64

echo "build linux..."
export GOOS=linux
cd ${ROOT_DIR}/app/lals && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/linux/bin/lals

echo "build macos..."
export GOOS=darwin
cd ${ROOT_DIR}/app/lals && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/macos/bin/lals

echo "build windows..."
export GOOS=windows
cd ${ROOT_DIR}/app/lals && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/windows/bin/lals

cd ${ROOT_DIR}/${OUT_DIR}
v=`git tag | tail -n 1`
zip -r lal_${v}_linux.zip linux
zip -r lal_${v}_macos.zip macos
zip -r lal_${v}_windows.zip windows
