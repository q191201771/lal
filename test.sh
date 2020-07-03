#!/usr/bin/env bash

#set -x

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
# 从网上下载测试用的flv文件
if [ ! -s "./testdata/test.flv" ]; then
    if [ ! -d "./testdata" ]; then
        mkdir "./testdata"
    fi
    wget https://github.com/q191201771/doc/raw/master/stuff/source.200kbps.768x320.flv -O ./testdata/test.flv
    if [ ! -s "./testdata/test.flv" ]; then
        wget https://pengrl.com/images/other/source.200kbps.768x320.flv -O ./testdata/test.flv
    fi
fi

# 将测试的flv文件分别拷贝到logic，rtmp，httpflv，hls的testdata目录下
if [ ! -s "./pkg/logic/testdata/test.flv" ]; then
    if [ ! -d "./pkg/logic/testdata" ]; then
        mkdir "./pkg/logic/testdata"
    fi
    cp ./testdata/test.flv ./pkg/logic/testdata/test.flv
fi
if [ ! -s "./pkg/rtmp/testdata/test.flv" ]; then
    if [ ! -d "./pkg/rtmp/testdata" ]; then
        mkdir "./pkg/rtmp/testdata"
    fi
    cp ./testdata/test.flv ./pkg/rtmp/testdata/test.flv
fi
if [ ! -s "./pkg/httpflv/testdata/test.flv" ]; then
    if [ ! -d "./pkg/httpflv/testdata" ]; then
        mkdir "./pkg/httpflv/testdata"
    fi
    cp ./testdata/test.flv ./pkg/httpflv/testdata/test.flv
fi
if [ ! -s "./pkg/hls/testdata/test.flv" ]; then
    if [ ! -d "./pkg/hls/testdata" ]; then
        mkdir "./pkg/hls/testdata"
    fi
    cp ./testdata/test.flv ./pkg/hls/testdata/test.flv
fi

# 将配置文件分别拷贝到logic，rtmp，httpflv，hls的testdata目录下
cp ./conf/lalserver.conf.json.tmpl ./pkg/logic/testdata/lalserver.conf.json
cp ./conf/lalserver.conf.json.tmpl ./pkg/rtmp/testdata/lalserver.conf.json
cp ./conf/lalserver.conf.json.tmpl ./pkg/httpflv/testdata/lalserver.conf.json
cp ./conf/lalserver.conf.json.tmpl ./pkg/hls/testdata/lalserver.conf.json

echo "" > coverage.txt
for d in $(go list ./... | grep -v vendor | grep pkg | grep -v innertest); do
    go test -race -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done

rm -rf ./pkg/logic/logs ./pkg/rtmp/logs ./pkg/httpflv/logs ./pkg/hls/logs
#rm -rf ./pkg/logic/testdata ./pkg/rtmp/testdata ./pkg/httpflv/testdata ./pkg/hls/testdata

echo 'done.'
