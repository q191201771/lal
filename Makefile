.PHONY: build
build: deps
	./build.sh

.PHONY: build_for_linux
build_for_linux: deps
	./build_for_linux.sh

.PHONY: test
test: deps
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
