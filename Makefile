.PHONY: all

BIN := openair-station
PKG := github.com/openairtech/station
ARCH := amd64 arm arm64

PUB_SERVER := openair.city
PUB_DIR := /var/www/get.openair.city/station

BINDIR = bin

VERSION_VAR := main.Version
TIMESTAMP_VAR := main.Timestamp

VERSION ?= $(shell git describe --always --dirty --tags)
TIMESTAMP := $(shell date '+%Y-%m-%d_%T%Z')

GOBUILD_LDFLAGS := -ldflags "-linkmode external -extldflags \"-static\" -s -w -X $(VERSION_VAR)=$(VERSION) -X $(TIMESTAMP_VAR)=$(TIMESTAMP)"

default: all

all: build

build:
	go build -x $(GOBUILD_LDFLAGS) -v -o $(BINDIR)/$(BIN)

build-static: $(addprefix build-static-, $(ARCH))

build-static-amd64:
	env CGO_ENABLED=1 CC=musl-gcc GOOS=linux GOARCH=amd64 go build -a -installsuffix "static" $(GOBUILD_LDFLAGS) -o $(BINDIR)/$(BIN).amd64

build-static-arm:
	env CGO_ENABLED=1 CC=arm-linux-musleabi-gcc GOOS=linux GOARCH=arm go build -a -installsuffix "static" $(GOBUILD_LDFLAGS) -o $(BINDIR)/$(BIN).arm

build-static-arm64:
	env CGO_ENABLED=1 CC=aarch64-linux-musleabi-gcc GOOS=linux GOARCH=arm64 go build -a -installsuffix "static" $(GOBUILD_LDFLAGS) -o $(BINDIR)/$(BIN).arm64

shasum:
	cd $(BINDIR) && for file in $(ARCH) ; do sha256sum ./$(BIN).$${file} > ./$(BIN).$${file}.sha256.txt; done

clean:
	rm -dRf $(BINDIR)

dist: clean build-static shasum
	cp contrib/scripts/* $(BINDIR)

publish: dist
	rsync -az $(BINDIR)/ $(PUB_SERVER):$(PUB_DIR)

fmt:
	go fmt ./...

# https://golangci.com/
# curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $GOPATH/bin v1.10.2
lint:
	golangci-lint run

test:
	go test ./...
