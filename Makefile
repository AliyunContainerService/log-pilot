TAG ?= latest
REGISTRY ?= acs

DOCKER ?= docker
SED_I ?= sed -i

GOHOSTOS ?= $(shell go env GOHOSTOS)
ifeq ($(GOHOSTOS),darwin)
  SED_I=sed -i ''
endif

REPO_INFO=$(shell git config --get remote.origin.url)

ifndef GIT_COMMIT
  GIT_COMMIT := git-$(shell git rev-parse --short HEAD)
endif

PKG = github.com/AliyunContainerService/log-pilot

ARCH ?= $(shell go env GOARCH)
GOARCH = ${ARCH}
DUMB_ARCH = ${ARCH}
GOBUILD_FLAGS :=
ALL_ARCH = amd64
BUSTED_ARGS =-v --pattern=_test

IMGNAME = log-pilot
IMAGE = $(REGISTRY)/$(IMGNAME)
#MULTI_ARCH_IMG = $(IMAGE)-$(ARCH)
MULTI_ARCH_IMG = $(IMAGE)

TEMP_DIR := $(shell mktemp -d)
DEF_VARS:=ARCH=$(ARCH)           \
	TAG=$(TAG)               \
	PKG=$(PKG)               \
	GOARCH=$(GOARCH)         \
	GIT_COMMIT=$(GIT_COMMIT) \
	REPO_INFO=$(REPO_INFO)   \
	PWD=$(PWD)

.PHONY: container
container: clean-container .container-$(ARCH)

.PHONY: .container-$(ARCH)
.container-$(ARCH):
	@echo "+ Copying artifact to temporary directory"
	mkdir -p $(TEMP_DIR)/
	cp -RP ./* $(TEMP_DIR)/

	@echo "+ Building container image $(MULTI_ARCH_IMG)-filebeat:$(TAG)"
	$(DOCKER) build --no-cache --pull -t $(MULTI_ARCH_IMG)-filebeat:$(TAG) -f Dockerfile.filebeat $(TEMP_DIR)/
ifeq ($(ARCH), amd64)
	$(DOCKER) tag $(MULTI_ARCH_IMG)-filebeat:$(TAG) $(IMAGE)-filebeat:$(TAG)
	$(DOCKER) tag $(MULTI_ARCH_IMG)-filebeat:$(TAG) $(IMGNAME):$(TAG)
endif

	@echo "+ Building container image $(MULTI_ARCH_IMG)-fluentd:$(TAG)"
	$(DOCKER) build --no-cache --pull -t $(MULTI_ARCH_IMG)-fluentd:$(TAG) -f Dockerfile.fluentd $(TEMP_DIR)/
ifeq ($(ARCH), amd64)
	$(DOCKER) tag $(MULTI_ARCH_IMG)-fluentd:$(TAG) $(IMAGE)-fluentd:$(TAG)
endif

.PHONY: clean-container
clean-container:
	@echo "+ Deleting container image $(MULTI_ARCH_IMG):$(TAG)"
	$(DOCKER) rmi -f $(MULTI_ARCH_IMG)-filebeat:$(TAG) || true
	$(DOCKER) rmi -f $(MULTI_ARCH_IMG)-fluentd:$(TAG) || true

.PHONY: static-check
static-check:
	@$(DEF_VARS) \
	build/static-check.sh

.PHONY: test
test:
	@$(DEF_VARS)                 \
	build/test.sh

.PHONY: vet
vet:
	@go vet $(shell go list ${PKG}/... | grep -v vendor)

.PHONY: check_dead_links
check_dead_links:
	docker run -t \
	  -v $$PWD:/tmp aledbf/awesome_bot:0.1 \
	  --allow-dupe \
	  --allow-redirect $(shell find $$PWD -mindepth 1 -name "*.md" -printf '%P\n' | grep -v vendor | grep -v Changelog.md)

.PHONY: dep-ensure
dep-ensure:
	dep version || curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	dep ensure -v
	dep prune -v
	find vendor -name '*_test.go' -delete

.PHONY: misspell
misspell:
	@go get github.com/client9/misspell/cmd/misspell
	misspell \
		-locale US \
		-error \
		main.go assets/* build/* docs/* examples/* hack/* pilot/* quickstart/* README.md