#!/usr/bin/env bash

#set -x

echo '-----add_src_license-----'
if command -v add_src_license >/dev/null 2>&1; then
    add_src_license -d ./ -e 191201771@qq.com -n Chef
else
    echo 'CHEFNOTICEME add_src_license not exist!'
fi

echo '-----gofmt-----'
if command -v gofmt >/dev/null 2>&1; then
    gofmt -s -l -w ./
else
    echo 'CHEFNOTICEME gofmt not exist!'
fi

echo '-----goimports-----'
if command -v goimports >/dev/null 2>&1; then
    goimports -l ./
    goimports -w ./
else
    echo 'CHEFNOTICEME goimports not exist!'
fi

echo '-----go vet-----'
for d in $(go list ./... | grep -v vendor); do
    if command -v go >/dev/null 2>&1; then
        go vet $d
    else
        echo 'CHEFNOTICEME go vet not exist'
    fi
done

# 跑 go test 生成测试覆盖率
echo "-----CI coverage-----"

## 从网上下载测试用的flv文件
if [ ! -s "./testdata/test.flv" ]; then
    if [ ! -d "./testdata" ]; then
        mkdir "./testdata"
    fi
    wget https://github.com/q191201771/doc/raw/master/av/wontcry30s.flv -O ./testdata/test.flv
    if [ ! -s "./testdata/test.flv" ]; then
        wget https://pengrl.com/images/other/wontcry30s.flv -O ./testdata/test.flv
    fi
fi

## 拷贝测试依赖的文件
cp ./conf/lalserver.conf.json.tmpl ./testdata/lalserver.conf.json
mkdir "./testdata/conf"
cp ./conf/cert.pem ./conf/key.pem ./testdata/conf/
cp ./conf/cert.pem ./conf/key.pem ./testdata/conf/

## 执行所有pkg里的单元测试，并生成测试覆盖文件
echo "" > coverage.txt
for d in $(go list ./... | grep -v vendor | grep pkg); do
    go test -race -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done

## 删除测试生成的垃圾文件
find ./pkg -name 'lal_record' | xargs rm -rf
find ./pkg -name 'logs' | xargs rm -rf

echo 'done.'
