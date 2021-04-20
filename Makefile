.PHONY: build
build: deps
	./build.sh

.PHONY: test
test: deps
	./test.sh

.PHONY: deps
deps:
	go get -t -v ./...

.PHONY: clean
clean:
	rm -rf ./bin ./release ./logs

.PHONY: all
all: build test
