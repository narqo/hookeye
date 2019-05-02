PKG := $(shell head -1 go.mod | cut -d' ' -f 2)

GO        ?= go
LDFLAGS   :=
GOFLAGS   :=
TESTFLAGS :=

BUILDTIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GITSHA := $(shell git rev-parse --short HEAD 2>/dev/null)

ifndef VERSION
	VERSION := git-$(GITSHA)
endif

LDFLAGS += -X $(PKG)/version.Version=$(VERSION)
LDFLAGS += -X $(PKG)/version.Commit=$(GITSHA)
LDFLAGS += -X $(PKG)/version.BuildTime=$(BUILDTIME)

BUILDDIR := BUILD

# Required for globs to work correctly
SHELL := /bin/bash

BUILD.go = $(GO) build $(GOFLAGS)
TEST.go  = $(GO) test $(TESTFLAGS)

.PHONY: all build test

all: build

build:
	$(BUILD.go) -ldflags "$(LDFLAGS)" -o $(BUILDDIR)/hookeye $(PKG)

test:
	$(TEST.go) -ldflags "$(LDFLAGS)" ./...
