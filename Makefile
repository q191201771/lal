.PHONY: build
build:
	./script/build.sh

.PHONY: build_for_linux
build_for_linux:
	export CGO_ENABLED=0 && export GOS=linux && ./script/build.sh

.PHONY: build_for_linux_amd64
build_for_linux_amd64:
	export CGO_ENABLED=0 && export GOS=linux && export GOARCH=amd64 && ./script/build.sh

.PHONY: build_for_linux_arm64
build_for_linux_arm64:
	export CGO_ENABLED=0 && export GOS=linux && export GOARCH=arm64 && ./script/build.sh

.PHONY: test
test:
	./script/test.sh

.PHONY: image
image:
	./script/build_docker_image.sh

.PHONY: clean
clean:
	./script/clean.sh

.PHONY: check_versions
check_versions:
	./script/check_versions.sh

.PHONY: gen_release
gen_release:
	./script/gen_release.sh

.PHONY: gen_tag
gen_tag:
	./script/gen_tag.sh

.PHONY: all
all: build test
