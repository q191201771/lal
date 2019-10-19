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
if [ ! -f "pkg/rtmp/testdata/test.flv" ]; then
    echo "CHEFERASEME test.flv not exist."
    if [ ! -d "pkg/rtmp/testdata" ]; then
        echo "CHEFERASEME mkdir."
        mkdir "pkg/rtmp/testdata"
    fi
    wget https://pengrl.com/images/other/source.200kbps.768x320.flv -O pkg/rtmp/testdata/test.flv
else
    echo "CHEFERASEME test.flv exist."
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
