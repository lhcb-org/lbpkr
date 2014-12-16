## simple makefile to log workflow
.PHONY: all test clean build install tag dist release

export LBPKR_VERSION=0.1.`date +%Y%m%d`
export LBPKR_RELEASE=0
export LBPKR_REVISION=`git rev-parse --short HEAD`

GOFLAGS ?= $(GOFLAGS:)

all: install test


build:
	@go build $(GOFLAGS) ./...

install:
	@go get $(GOFLAGS) ./...

test: install
	@go test -short $(GOFLAGS) ./...

all-test: install
	@go test $(GOFLAGS) -parallel 10 ./...

bench: install
	@go test -run=NONE -bench=. $(GOFLAGS) ./...

clean:
	@go clean $(GOFLAGS) -i ./...

tag:
	git tag lbpkr-${LBPKR_VERSION}-${LBPKR_RELEASE}

dist:
	git fetch --all
	git checkout master
	git pull origin master
	make all-test
	lbpkr version
	lbpkr self bdist-rpm -version=${LBPKR_VERSION} -release=${LBPKR_RELEASE}

release: dist
	lbpkr self upload-rpm lbpkr-${LBPKR_VERSION}-${LBPKR_RELEASE}.x86_64.rpm
	/bin/rm lbpkr-${LBPKR_VERSION}-${LBPKR_RELEASE}.x86_64.rpm

## EOF
