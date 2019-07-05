.PHONY: all

BIN := openair-station-esp
PKG := github.com/openairtech/station-esp
ARCH := amd64 arm

PUB_SERVER := openair.city
PUB_DIR := /var/www/get.openair.city/station-esp

BINDIR = bin

VERSION_VAR := main.Version
TIMESTAMP_VAR := main.Timestamp

VERSION ?= $(shell git describe --always --dirty --tags)
TIMESTAMP := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')

GOBUILD_LDFLAGS := -ldflags "-s -w -X $(VERSION_VAR)=$(VERSION) -X $(TIMESTAMP_VAR)=$(TIMESTAMP)"

default: all

all: build

build:
	go build -x $(GOBUILD_LDFLAGS) -v -o $(BINDIR)/$(BIN)

build-static: $(ARCH)

$(ARCH):
	env CGO_ENABLED=0 GOOS=linux GOARCH=$@ go build -a -installsuffix "static" $(GOBUILD_LDFLAGS) -o $(BINDIR)/$(BIN).$@

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
