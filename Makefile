PWD := $(shell pwd)
PKG := github.com/conseweb/poe

VERSION := $(shell cat VERSION.txt)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

APP := poe
IMAGE := conseweb/poe:$(GIT_BRANCH)
INNER_GOPATH := /opt/gopath
DEV_IMAGE := ckeyer/obc:dev

test: 
	docker run --rm \
	 --name $(APP)-testing \
	 -v $(PWD):$(INNER_GOPATH)/src/$(PKG) \
	 -w $(INNER_GOPATH)/src/$(PKG) \
	 $(DEV_IMAGE) make unit-test

unit-test: 
	go test $$(go list ./... |grep -v "vendor") 

integration-test:
	echo "todo..."

testInner: 
	go test $$(go list ./... |grep -v "vendor"|grep -v "integration-tests")

image: build build-image

build: 
	docker run --rm \
	 --name $(APP)-building \
	 -v $(PWD):$(INNER_GOPATH)/src/$(PKG) \
	 -w $(INNER_GOPATH)/src/$(PKG) \
	 -e CGO_ENABLED=0 \
	 $(DEV_IMAGE) make local

build-image:
	docker build -t $(IMAGE) -f Dockerfile.run .

local:
	go build -o bundles/$(APP) .

dev:
	docker run --rm \
	 --name $(APP)-dev \
	 -v $(PWD):$(INNER_GOPATH)/src/$(PKG) \
	 -w $(INNER_GOPATH)/src/$(PKG) \
	 -it $(DEV_IMAGE) bash
