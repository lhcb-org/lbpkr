## simple makefile to log workflow
.PHONY: all test clean build

export LBPKR_VERSION=0.1.`date +%Y%m%d`
export LBPKR_RELEASE=0
export LBPKR_REVISION=`git rev-parse --short HEAD`

#GOFLAGS := $(GOFLAGS:-race -v)
GOFLAGS := $(GOFLAGS:-v)

all: build test
	@echo "## bye."

build: clean
	@go get $(GOFLAGS) ./...

test: build
	@go test -v $(GOFLAGS) ./...

clean:
	@go clean $(GOFLAGS) -i ./...

dist:
	git fetch --all
	git checkout master
	git pull origin master
	make all
	lbpkr version
	lbpkr self bdist-rpm -version=${LBPKR_VERSION} -release=${LBPKR_RELEASE}

release: dist
	lbpkr self upload-rpm lbpkr-${LBPKR_VERSION}-${LBPKR_RELEASE}.x86_64.rpm

## EOF
