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

cd ${ROOT_DIR}
rm -rf coverage.txt
rm -rf *.laldump *.log
rm -rf ./release ./bin ./lal_record ./logs
find ./pkg -name 'lal_record' | xargs rm -rf
find ./pkg -name 'logs' | xargs rm -rf
