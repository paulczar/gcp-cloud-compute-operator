# Set the shell
SHELL := /bin/bash

# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)

# Set the build dir, where built cross-compiled binaries will be output
BUILDDIR := ${PREFIX}/cross

# Populate version variables
# Add to compile time flags
VERSION := $(shell cat VERSION.txt)
GITCOMMIT := $(shell git rev-parse --short HEAD)
GITUNTRACKEDCHANGES := $(shell git status --porcelain --untracked-files=no)
ifneq ($(GITUNTRACKEDCHANGES),)
	GITCOMMIT := $(GITCOMMIT)-dirty
endif
ifeq ($(GITCOMMIT),)
    GITCOMMIT := ${GITHUB_SHA}
endif
CTIMEVAR=-X $(PKG)/version.GITCOMMIT=$(GITCOMMIT) -X $(PKG)/version.VERSION=$(VERSION)
GO_LDFLAGS=-ldflags "-w $(CTIMEVAR)"
GO_LDFLAGS_STATIC=-ldflags "-w $(CTIMEVAR) -extldflags -static"

# Set our default go compiler
GO := go

# List the GOOS and GOARCH to build
GOOSARCHES = $(shell cat .goosarch)

# Set the graph driver as the current graphdriver if not set.
DOCKER_GRAPHDRIVER := $(if $(DOCKER_GRAPHDRIVER),$(DOCKER_GRAPHDRIVER),$(shell docker info 2>&1 | grep "Storage Driver" | sed 's/.*: //'))
export DOCKER_GRAPHDRIVER

# If this session isn't interactive, then we don't want to allocate a
# TTY, which would fail, but if it is interactive, we do want to attach
# so that the user can send e.g. ^C through.
INTERACTIVE := $(shell [ -t 0 ] && echo 1 || echo 0)
ifeq ($(INTERACTIVE), 1)
	DOCKER_FLAGS += -t
endif

.PHONY: lint
lint: ## Verifies `golint` passes.
	@echo "+ $@"
	@if [[ ! -z "$(shell golint ./cmd/... ./pkg/... | grep -v -E "struct field \w+ should be" | tee /dev/stderr)" ]]; then \
		exit 1; \
	fi

.PHONY: fmt
fmt: ## Verifies all files have been `gofmt`ed.
	@echo "+ $@"
	@if [[ ! -z "$(shell gofmt -s -l ./pkg | tee /dev/stderr)" ]]; then \
		exit 1; \
	@if [[ ! -z "$(shell gofmt -s -l ./cmd | tee /dev/stderr)" ]]; then \
		exit 1; \
	fi

.PHONY: docker
docker: ## builds docker image using operator-sdk
	operator-sdk build paulczar/$(NAME)

.PHONY: build
build: ## Builds a dynamic executable or package.
	$(GO) build -tags "$(BUILDTAGS)" ${GO_LDFLAGS} -o build/_output/bin/$(NAME) cmd/manager/main.go
