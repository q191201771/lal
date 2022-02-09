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
	rm -rf ./bin ./release ./logs ./lal_record ./pkg/base/logs ./pkg/base/lal_record ./pkg/httpts/logs ./pkg/httpts/lal_record
	rm -rf ./pkg/mpegts/logs ./pkg/mpegts/lal_record ./pkg/remux/logs ./pkg/remux/lal_record ./pkg/rtprtcp/logs ./pkg/rtprtcp/lal_record
	rm -rf ./pkg/rtsp/logs ./pkg/rtsp/lal_record ./pkg/sdp/logs ./pkg/sdp/lal_record

.PHONY: all
all: build test
