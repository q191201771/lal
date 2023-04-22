#!/usr/bin/env bash

#set -x
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,https://goproxy.io,direct
export GO111MODULE=on
export GOPROXY=https://goproxy.cn,https://goproxy.io,direct
THIS_FILE=$(readlink -f $0)
THIS_DIR=$(dirname $THIS_FILE)
ROOT_DIR=${THIS_DIR}/..

cd ${ROOT_DIR}

echo '-----add_src_license-----'
if command -v add_src_license >/dev/null 2>&1; then
    add_src_license -d ${ROOT_DIR} -e 191201771@qq.com -n Chef
else
    echo 'CHEFNOTICEME add_src_license not exist!'
fi

echo '-----gofmt-----'
if command -v gofmt >/dev/null 2>&1; then
    gofmt -s -l -w ${ROOT_DIR}
else
    echo 'CHEFNOTICEME gofmt not exist!'
fi

#echo '-----goimports-----'
#if command -v goimports >/dev/null 2>&1; then
#    goimports -l ${ROOT_DIR}
#    goimports -w ${ROOT_DIR}
#else
#    echo 'CHEFNOTICEME goimports not exist!'
#fi

echo '-----go vet-----'
for d in $(go list ${ROOT_DIR}/... | grep -v vendor); do
    if command -v go >/dev/null 2>&1; then
        go vet $d
    else
        echo 'CHEFNOTICEME go vet not exist'
    fi
done

# 跑 go test 生成测试覆盖率
echo "-----CI coverage-----"

## 从网上下载测试用的flv文件
if [ ! -s "${ROOT_DIR}/testdata/test.flv" ]; then
    if [ ! -d "${ROOT_DIR}/testdata" ]; then
        mkdir "${ROOT_DIR}/testdata"
    fi
    wget https://github.com/q191201771/doc/raw/master/av/wontcry30s.flv -O ${ROOT_DIR}/testdata/test.flv
    if [ ! -s "${ROOT_DIR}/testdata/test.flv" ]; then
        wget https://pengrl.com/images/other/wontcry30s.flv -O ${ROOT_DIR}/testdata/test.flv
    fi
fi

## 拷贝测试依赖的文件
cp ${ROOT_DIR}/conf/lalserver.conf.json.tmpl ${ROOT_DIR}/testdata/lalserver.conf.json
mkdir ${ROOT_DIR}/testdata/conf
cp ${ROOT_DIR}/conf/cert.pem ${ROOT_DIR}/conf/key.pem ${ROOT_DIR}/testdata/conf/
cp ${ROOT_DIR}/conf/cert.pem ${ROOT_DIR}/conf/key.pem ${ROOT_DIR}/testdata/conf/

## 执行所有pkg里的单元测试，并生成测试覆盖文件
echo "" > coverage.txt
for d in $(go list ${ROOT_DIR}/... | grep -v vendor | grep pkg); do
    go test -race -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done

## 删除测试生成的垃圾文件
find ${ROOT_DIR}/pkg -name 'lal_record' | xargs rm -rf
find ${ROOT_DIR}/pkg -name 'logs' | xargs rm -rf

echo 'done.'
