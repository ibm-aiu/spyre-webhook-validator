# Copyright (c) 2025, 2026 IBM Corp.
# SPDX-License-Identifier: Apache-2.0

# Enable automatic Go toolchain management
export GOTOOLCHAIN = auto

GOLANG_VERSION		?= $(shell cd $(REPO_ROOT) && go list -f {{.GoVersion}} -m)
BUILDER_IMAGE		?= registry.access.redhat.com/ubi9/go-toolset:$(GOLANG_VERSION)
GOTOOLCHAIN			?= go$(GOLANG_VERSION)
MAKEFILE_PATH		:= $(abspath $(lastword $(MAKEFILE_LIST)))
REPO_ROOT 			:= $(abspath $(patsubst %/,%,$(dir $(MAKEFILE_PATH))))
CURRENT_DIR			:= $(shell pwd)
VERSION				?= $(shell cat $(REPO_ROOT)/VERSION)
REGISTRY			?= docker.io/spyre-operator
DOCKER				?= $(shell command -v podman 2> /dev/null || echo docker)
DOCKERFILE			= $(REPO_ROOT)/Dockerfile
DOCKER_BUILD_OPTS	?= --progress=plain

IMAGE_NAME 			:= $(REGISTRY)/spyre-webhook-validator
IMAGE_TAG 			?= $(VERSION)
IMAGE 				?= $(IMAGE_NAME):$(IMAGE_TAG)
TEST_IMG			?= $(IMAGE_NAME):dev
CODECOV_PERCENT		?= 67
COVERAGE_FILE		:= coverage-report.out


KUBECTL              ?= $(shell command -v oc 2> /dev/null || echo kubectl)

# Operating system
OS					?= $(shell go env GOOS)
ARCH				?= $(shell go env GOARCH)


# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
ENVTEST			?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT	?= $(LOCALBIN)/golangci-lint
GOVULCHECK		?= $(LOCALBIN)/govulncheck
GINKGO			?= $(LOCALBIN)/ginkgo
YQ				?= $(LOCALBIN)/yq
YAMLFMT			?= $(LOCALBIN)/yamlfmt

## Tool Versions
ENVTEST_K8S_VERSION			?= 1.31
GINKGO_VERSION				?= v2.25.1
GOLANGCI_LINT_VERSION		?= 2.11.4
GOLANGCI_LINT_INSTALL_SCRIPT ?= https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh
YQ_VERSION					?= v4.29.2
YAMLFMT_VERSION				?= v0.17.0
PYTHON                      ?= python3
PIP                         ?= pip3

# detect-secrets
DETECT_SECRETS_GIT ?= "https://github.com/ibm/detect-secrets.git@master\#egg=detect-secrets"

DOCKER_GO_BUILD_FLAGS ?=
ADDITIONAL_IMAGE_TAG ?= latest

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Development tools

.PHONY: venv
venv: ## Setup and activate venv
	$(PYTHON) -m venv venv

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download and install ginkgo
$(GINKGO):$(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download and install setup-envtest
$(ENVTEST):$(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20240624150636-162a113134de

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ### Download golangci-lint locally if necessary.
$(GOLANGCI_LINT):$(LOCALBIN)
	test -s $(GOLANGCI_LINT) || { curl -sSfL $(GOLANGCI_LINT_INSTALL_SCRIPT) | sh -s -- -b $(LOCALBIN)  v$(GOLANGCI_LINT_VERSION); }

.PHONY: yq
yq: $(YQ) ## Download yq locally if necessary.
$(YQ): $(LOCALBIN)
	test -s $(YQ) || GOBIN=$(LOCALBIN) go install github.com/mikefarah/yq/v4@$(YQ_VERSION)

.PHONY: yamlfmt
yamlfmt: $(YAMLFMT) ## Download yamlfmt locally if necessary
$(YAMLFMT):$(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/google/yamlfmt/cmd/yamlfmt@$(YAMLFMT_VERSION)

.PHONY: govulncheck
govulncheck: $(GOVULCHECK) ## Download govulncheck tool if necessary
$(GOVULCHECK): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install golang.org/x/vuln/cmd/govulncheck@latest

##@ Test targets

.PHONY: test
test: ginkgo envtest fmt vet ## Run unit tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" $(LOCALBIN)/ginkgo run --cover --coverprofile=$(COVERAGE_FILE) -race -v ./...
	go tool cover -func $(COVERAGE_FILE)
	go tool cover -html $(COVERAGE_FILE) -o coverage-report.html
	@percentage=$$(go tool cover -func=$(COVERAGE_FILE) | grep ^total | awk '{print $$3}' | tr -d '%'); \
		if (( $$(echo "$$percentage < $(CODECOV_PERCENT)" | bc -l) )); then \
			echo "----------"; \
			echo "Total test coverage ($${percentage}%) is less than the coverage threshold ($(CODECOV_PERCENT)%)."; \
			exit 1; \
		fi


##@ Development Targets

.PHONY: fmt
fmt: ## Run the formatter
	go fmt ./...

.PHONY: vet
vet: vendor ## Run the vet command
	go vet -mod vendor ./...

.PHONY: vendor
vendor: ## Run vendor
	go mod vendor

.PHONY: build
build: vendor ## Build local binary
	go build -mod vendor -race -a -o $(LOCALBIN)/spyre-webhook-validator main.go

.PHONY: lint
lint: golangci-lint vendor ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run --config $(REPO_ROOT)/.golangci.yaml

.PHONY: lint-fix
lint-fix: golangci-lint vendor ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run --fix --config $(REPO_ROOT)/.golangci.yaml

.PHONY: vulcheck
vulcheck: govulncheck ## Scan for golang vulnerabilities
	$(GOVULCHECK) -show verbose	 ./...

.PHONY: clean
clean: ## Clean-up intermediate artifacts
	-rm -rf vendor
	-rm -rf $(LOCALBIN)

.PHONY: pr
pr: vendor fmt vet lint test docker-build docker-push ## Execute a pull request build

.PHONY: development
development: vendor fmt vet lint test docker-buildx docker-pushx ## Execute a development build

##@ Image operations

.PHONY: docker-build
docker-build: vendor ## Build sypre webhook validator image for the build host architecture
	$(DOCKER) build $(DOCKER_BUILD_OPTS) --pull \
	--tag $(IMAGE) \
	--tag $(IMAGE_NAME):$(ADDITIONAL_IMAGE_TAG) \
	--build-arg VERSION="$(VERSION)" \
	--build-arg BUILDER_IMAGE="$(BUILDER_IMAGE)" \
	--build-arg BUILD_FLAGS="$(DOCKER_GO_BUILD_FLAGS)" \
	--file $(DOCKERFILE) $(CURDIR)

.PHONY: docker-push
docker-push: ## Push spyre webhook validator image image for the build host architecture
	$(DOCKER) push $(IMAGE)

.PHONY: docker-build-push
docker-build-push: docker-build docker-push ## Build and push the spyre webhook validator image for the build host

.PHONY: docker-build-amd64
docker-build-amd64: vendor ## Build the spyre webhook validator image for linux/amd64
ifeq ($(DOCKER), docker)
	docker buildx build --platform linux/amd64 \
		$(DOCKER_BUILD_OPTS) \
		--push --pull  --load --no-cache \
		--provenance false --sbom false \
		--tag $(IMAGE)-amd64 \
		--tag $(IMAGE_NAME):$(ADDITIONAL_IMAGE_TAG)-amd64 \
		--build-arg VERSION="$(VERSION)" \
		--build-arg BUILDER_IMAGE="$(BUILDER_IMAGE)" \
		--build-arg BUILD_FLAGS="$(DOCKER_GO_BUILD_FLAGS)" \
		--file $(DOCKERFILE) $(CURDIR)
else
	podman build --platform linux/amd64 \
		--format docker \
		$(DOCKER_BUILD_OPTS) \
		--build-arg VERSION="$(VERSION)" \
		--build-arg BUILD_FLAGS="$(DOCKER_GO_BUILD_FLAGS)" \
		--build-arg BUILDER_IMAGE="$(BUILDER_IMAGE)" \
		--tag $(IMAGE)-amd64 \
		--tag $(IMAGE_NAME):$(ADDITIONAL_IMAGE_TAG)-amd64 \
		--file $(DOCKERFILE) $(CURDIR)
endif


.PHONY: docker-push-amd64
docker-push-amd64: ## Push webhook validator for amd64 only
ifeq ($(DOCKER), docker)
	echo "Images already pushed by Docker"
else
	$(DOCKER) push $(IMAGE)-amd64
endif


.PHONY: docker-build-manifest
docker-build-manifest: ## Build spyre webhook validator manifest for all architectures
ifeq ($(DOCKER), docker)
	docker manifest annotate $(IMAGE) $(IMAGE)-amd64   --os linux --arch amd64
else
	podman manifest create $(IMAGE)
	podman manifest add $(IMAGE) $(IMAGE)-amd64
endif

.PHONY: docker-push-manifest
docker-push-manifest: ## Push webhook validator image multi architecture manifest
	$(DOCKER) manifest push $(IMAGE)

.PHONY: docker-buildx
docker-buildx: docker-build-amd64  ## Build webhook validator image for all architectures

.PHONY: docker-pushx ## Push spyre webhook validator image image for all architectures
docker-pushx: docker-push-amd64 docker-build-manifest docker-push-manifest

.PHONY: docker-build-pushx
docker-build-pushx: docker-buildx docker-pushx ## Build and push the multi architecture docker image

.PHONY: docker-remove-images
docker-remove-images: ## Remove images from build host
	$(DOCKER) manifest rm $(IMAGE) || true
	$(DOCKER) rmi -f $(IMAGE)-amd64 || true

##@ Deployment

.PHONY: deploy
deploy: ## Deploy webhook
	$(KUBECTL) apply -f test/manifest/deploy/validator.yaml
	$(KUBECTL) apply -f test/manifest/deploy/validator-service.yaml
	$(KUBECTL) apply -f test/manifest/webhook-config/validatingwebhookconfig.yaml

.PHONY: undeploy
undeploy: ## Undeploy webhook
	$(KUBECTL) delete -f test/manifest/webhook-config/validatingwebhookconfig.yaml
	$(KUBECTL) delete -f test/manifest/deploy/validator-service.yaml
	$(KUBECTL) delete -f test/manifest/deploy/validator.yaml

.PHONY: detect-secrets-install
detect-secrets-install: venv ## Install detect-secret tool
	. venv/bin/activate; $(PIP) install "git+$(DETECT_SECRETS_GIT)"

.PHONY: secrets-scan
secrets-scan: venv detect-secrets-install ## Scan secrets and create secret-baseline for repo
	. venv/bin/activate; detect-secrets scan --no-ghe-scan --exclude-files go.sum --update .secrets.baseline

.PHONY: secrets-audit
secrets-audit: venv detect-secrets-install ## Audit secrets
	. venv/bin/activate; detect-secrets audit .secrets.baseline

# helper target for viewing the value of makefile variables.
print-%  : ;@echo $* = $($*)
