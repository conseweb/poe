PWD := $(shell pwd)
PKG := github.com/conseweb/poe

VERSION := $(shell cat VERSION.txt)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

APP := poe
IMAGE := conseweb/poe:$(GIT_BRANCH)
INNER_GOPATH := /opt/gopath
DEV_IMAGE := ckeyer/obc:dev

test: unit-test integration-test

unit-test: 
	echo "todo..."

integration-test:
	echo "todo..."

testInner: 
	go test $$(go list ./... |grep -v "vendor"|grep -v "integration-tests")

build: 
	docker run --rm \
	 --name $(BUILD_CONTAINER) \
	 -v $(PWD):$(INNER_GOPATH)/src/$(PKG) \
	 -w $(INNER_GOPATH)/src/$(PKG) \
	 $(DEV_IMAGE) go build -o bundles/$(APP) .

build-local:
	go build -o bundles/$(APP) .

dev:
	docker run --rm \
	 --name $(APP)-dev \
	 -v $(GOPATH):$(INNER_GOPATH) \
	 -w $(INNER_GOPATH)/src/$(PKG) \
	 -it $(DEV_IMAGE) bash
	 