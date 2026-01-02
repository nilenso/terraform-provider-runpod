.PHONY: build test testacc install clean fmt lint

BINARY=terraform-provider-runpod
VERSION?=0.1.0

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Linux)
    OS=linux
endif
ifeq ($(UNAME_S),Darwin)
    OS=darwin
endif
ifeq ($(UNAME_M),x86_64)
    ARCH=amd64
endif
ifeq ($(UNAME_M),arm64)
    ARCH=arm64
endif
ifeq ($(UNAME_M),aarch64)
    ARCH=arm64
endif

INSTALL_PATH=~/.terraform.d/plugins/registry.terraform.io/nilenso/runpod/$(VERSION)/$(OS)_$(ARCH)

build:
	go build -o $(BINARY) -v

test:
	go test -v ./... -short

testacc:
	TF_ACC=1 go test -v ./internal/provider -timeout 30m -p 1 -parallel 1

testacc-one:
	TF_ACC=1 go test -v ./internal/provider -timeout 30m -run $(TEST)

install: build
	mkdir -p $(INSTALL_PATH)
	cp $(BINARY) $(INSTALL_PATH)/

clean:
	rm -f $(BINARY)

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

deps:
	go mod tidy
	go mod download
