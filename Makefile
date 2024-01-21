# Copyright 2024 Richard Kosegi
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

REGISTRY ?= ghcr.io/rkosegi
TAG_PREFIX = v
DOCKER ?= docker
VERSION = $(shell cat VERSION)
VER_PARTS   := $(subst ., ,$(VERSION))
VER_MAJOR	:= $(word 1,$(VER_PARTS))
VER_MINOR   := $(word 2,$(VER_PARTS))
VER_PATCH   := $(word 3,$(VER_PARTS))
VER_NEXT_PATCH    := $(VER_MAJOR).$(VER_MINOR).$(shell echo $$(($(VER_PATCH)+1)))
TAG ?= $(TAG_PREFIX)$(VERSION)
BRANCH = $(strip $(shell git rev-parse --abbrev-ref HEAD))
ARCH ?= $(shell go env GOARCH)
BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
PKG = github.com/prometheus/common
GO_VERSION = 1.21.1
OS ?= $(shell uname -s | tr A-Z a-z)
IMAGE = $(REGISTRY)/k8s-footprint-exporter
USER ?= $(shell id -u -n)
HOST ?= $(shell hostname)

bump-patch-version:
	@echo Current: $(VER_CURRENT)
	@echo Next: $(VER_NEXT_PATCH)
	@echo "$(VER_NEXT_PATCH)" > VERSION
	sed -i 's/^appVersion: .*/appVersion: $(VER_NEXT_PATCH)/g' chart/Chart.yaml
	sed -i 's/^version: .*/version: $(VER_NEXT_PATCH)/g' chart/Chart.yaml
	git add -- Makefile chart/Chart.yaml
	git commit -sm "Bump version to $(VER_NEXT_PATCH)"

git-tag:
	git tag -am "Release $(VERSION)" $(VERSION)

update-go-deps:
	@for m in $$(go list -mod=readonly -m -f '{{ if and (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all); do \
		go get -d $$m; \
	done
	go mod tidy

gen-docs:
	cd chart && frigate gen . > README.md

lint:
	pre-commit run --all-files

test:
	go test -v ./...

build-local:
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build \
		-ldflags "-s -w -X ${PKG}/version.Version=${TAG} -X ${PKG}/version.Revision=${GIT_COMMIT} -X ${PKG}/version.Branch=${BRANCH} -X ${PKG}/version.BuildUser=${USER}@${HOST} -X ${PKG}/version.BuildDate=${BUILD_DATE}" \
		-o exporter

build-docker:
	$(DOCKER) build -t $(IMAGE):$(VERSION) \
		--build-arg GOVERSION=$(GO_VERSION) \
		--build-arg GOARCH=$(ARCH) \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		.

.PHONY: build-local build-docker clean lint test gen-docs update-go-deps bump-patch-version
