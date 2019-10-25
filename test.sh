#!/usr/bin/env bash

echo '-----add_go_license-----'
if command -v add_go_license >/dev/null 2>&1; then
    add_go_license -d ./ -e 191201771@qq.com -n Chef
else
    echo 'CHEFNOTICEME add_go_license not exist!'
fi

echo '-----gofmt-----'
if command -v gofmt >/dev/null 2>&1; then
    gofmt -l ./
    gofmt -w ./
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
if [ ! -f "./testdata/test.flv" ]; then
    if [ ! -d "./testdata" ]; then
        mkdir "./testdata"
    fi
    wget https://pengrl.com/images/other/source.200kbps.768x320.flv -O ./testdata/test.flv
fi
if [ ! -f "./pkg/rtmp/testdata/test.flv" ]; then
    if [ ! -d "./pkg/rtmp/testdata" ]; then
        mkdir "./pkg/rtmp/testdata"
    fi
    cp ./testdata/test.flv ./pkg/rtmp/testdata/test.flv
fi
if [ ! -f "./pkg/logic/testdata/test.flv" ]; then
    if [ ! -d "./pkg/logic/testdata" ]; then
        mkdir "./pkg/logic/testdata"
    fi
    cp ./testdata/test.flv ./pkg/logic/testdata/test.flv
fi
if [ ! -f "./pkg/logic/testdata/lals.default.conf.json" ]; then
    cp ./conf/lals.default.conf.json ./pkg/logic/testdata/lals.default.conf.json
fi

echo "" > coverage.txt
for d in $(go list ./... | grep -v vendor | grep pkg); do
    go test -race -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
echo 'done.'
