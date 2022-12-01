.PHONY: build
build:
	export GO111MODULE=on && export GOPROXY=https://goproxy.cn,https://goproxy.io,direct && ./build.sh

.PHONY: build_for_linux
build_for_linux:
	export CGO_ENABLED=0 && export GOS=linux && ./build.sh

.PHONY: build_for_linux_amd64
build_for_linux_amd64:
	export CGO_ENABLED=0 && export GOS=linux && export GOARCH=amd64 && ./build.sh

.PHONY: build_for_linux_arm64
build_for_linux_arm64:
	export CGO_ENABLED=0 && export GOS=linux && export GOARCH=arm64 && ./build.sh

.PHONY: test
test:
	./test.sh

.PHONY: deps
deps:
	go get -t -v ./...

.PHONY: image
image:
	docker build -f Dockerfile -t lal:latest .

.PHONY: clean
clean:
	rm -rf ./bin ./lal_record ./logs coverage.txt
	find ./pkg -name 'lal_record' | xargs rm -rf
	find ./pkg -name 'logs' | xargs rm -rf

.PHONY: all
all: build test
