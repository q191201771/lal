#!/usr/bin/env bash

#set -x

ROOT_DIR=`pwd`
OUT_DIR=release

v=`git tag --sort=version:refname | tail -n 1`
prefix=lal_${v}_

rm -rf ${ROOT_DIR}/${OUT_DIR}
mkdir -p ${ROOT_DIR}/${OUT_DIR}/${prefix}linux/bin
mkdir -p ${ROOT_DIR}/${OUT_DIR}/${prefix}linux/conf
mkdir -p ${ROOT_DIR}/${OUT_DIR}/${prefix}macos/bin
mkdir -p ${ROOT_DIR}/${OUT_DIR}/${prefix}macos/conf
mkdir -p ${ROOT_DIR}/${OUT_DIR}/${prefix}windows/bin
mkdir -p ${ROOT_DIR}/${OUT_DIR}/${prefix}windows/conf

echo ${v} >> ${ROOT_DIR}/${OUT_DIR}/${prefix}linux/README.txt
echo ${v} >> ${ROOT_DIR}/${OUT_DIR}/${prefix}macos/README.txt
echo ${v} >> ${ROOT_DIR}/${OUT_DIR}/${prefix}windows/README.txt
echo 'github: https://github.com/q191201771/lal' >> ${ROOT_DIR}/${OUT_DIR}/${prefix}linux/README.txt
echo 'github: https://github.com/q191201771/lal' >> ${ROOT_DIR}/${OUT_DIR}/${prefix}macos/README.txt
echo 'github: https://github.com/q191201771/lal' >> ${ROOT_DIR}/${OUT_DIR}/${prefix}windows/README.txt
echo 'doc: https://pengrl.com/lal' >> ${ROOT_DIR}/${OUT_DIR}/${prefix}linux/README.txt
echo 'doc: https://pengrl.com/lal' >> ${ROOT_DIR}/${OUT_DIR}/${prefix}macos/README.txt
echo 'doc: https://pengrl.com/lal' >> ${ROOT_DIR}/${OUT_DIR}/${prefix}windows/README.txt

cp conf/lalserver.conf.json conf/cert.pem conf/key.pem ${ROOT_DIR}/${OUT_DIR}/${prefix}linux/conf
cp conf/lalserver.conf.json conf/cert.pem conf/key.pem ${ROOT_DIR}/${OUT_DIR}/${prefix}macos/conf
cp conf/lalserver.conf.json conf/cert.pem conf/key.pem ${ROOT_DIR}/${OUT_DIR}/${prefix}windows/conf

GitTag=`git tag --sort=version:refname | tail -n 1`
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

export CGO_ENABLED=0
export GOARCH=amd64

echo "build linux..."
export GOOS=linux
cd ${ROOT_DIR}/app/lalserver && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/${prefix}linux/bin/lalserver

echo "build macos..."
export GOOS=darwin
cd ${ROOT_DIR}/app/lalserver && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/${prefix}macos/bin/lalserver

echo "build windows..."
export GOOS=windows
cd ${ROOT_DIR}/app/lalserver && go build -ldflags "$LDFlags" -o ${ROOT_DIR}/${OUT_DIR}/${prefix}windows/bin/lalserver.exe

cd ${ROOT_DIR}/${OUT_DIR}
zip -r ${prefix}linux.zip ${prefix}linux
zip -r ${prefix}macos.zip ${prefix}macos
zip -r ${prefix}windows.zip ${prefix}windows
