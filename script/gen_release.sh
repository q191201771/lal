#!/usr/bin/env bash

# linux, windows, macos, macos_arm, arm32, arm64
#
# name           GOOS    GOARCH
# linux       -> linux   (amd64)
# windows     -> windows (amd64)
# macos       -> darwin  (amd64)
# macos_arm64 -> darwin  arm64
# arm32       -> (linux) arm
# arm64       -> (linux) arm64

NAMES=("linux" "windows" "macos" "macos_arm64" "arm32" "arm64")
MAPPING_GOOS=("linux" "windows" "darwin" "darwin" "linux" "linux")
MAPPING_GOARCH=( "amd64" "amd64" "amd64" "arm64" "arm" "arm64")
MAPPING_EXE=("lalserver" "lalserver.exe" "lalserver" "lalserver" "lalserver" "lalserver")

#######################################################################################################################

#set -x
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct
export GO111MODULE=on
export GOPROXY=https://goproxy.cn,https://goproxy.io,direct
THIS_FILE=$(readlink -f $0)
THIS_DIR=$(dirname $THIS_FILE)
ROOT_DIR=${THIS_DIR}/..

OUT_DIR=${ROOT_DIR}/release

v=`git tag --sort=version:refname | tail -n 1`
prefix=lal_${v}_

rm -rf ${OUT_DIR}

# 创建目录
for name in ${NAMES[@]};
do
  mkdir -p ${OUT_DIR}/${prefix}${name}/bin
  mkdir -p ${OUT_DIR}/${prefix}${name}/conf
done

# README.txt
for name in ${NAMES[@]};
do
  echo ${v} >> ${OUT_DIR}/${prefix}${name}/README.txt
  echo 'github: https://github.com/q191201771/lal ' >> ${OUT_DIR}/${prefix}${name}/README.txt
  echo 'doc: https://pengrl.com/lal ' >> ${OUT_DIR}/${prefix}${name}/README.txt
done

# conf/
for name in ${NAMES[@]};
do
  cp conf/lalserver.conf.json conf/cert.pem conf/key.pem ${OUT_DIR}/${prefix}${name}/conf
done

# 编译不同架构和操作系统
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

for i in "${!NAMES[@]}";
do
  printf "build %s(%s %s)...\n" "${NAMES[$i]}" "${MAPPING_GOOS[$i]}" "${MAPPING_GOARCH[$i]}"
  export GOOS=${MAPPING_GOOS[$i]}
  export GOARCH=${MAPPING_GOARCH[$i]}
  cd ${ROOT_DIR}/app/lalserver && go build -ldflags "$LDFlags" -o ${OUT_DIR}/${prefix}${NAMES[$i]}/bin/${MAPPING_EXE[$i]}
done

# 打zip包
cd ${OUT_DIR}
for name in ${NAMES[@]};
do
  zip -r ${prefix}${name}.zip ${prefix}${name}
done
