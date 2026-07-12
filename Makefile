# ====================================================================================
# Setup Project
PROJECT_NAME := provider-sonarqube
PROJECT_REPO := github.com/crossplane/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

# ====================================================================================
# Setup Output

-include build/makelib/output.mk

# ====================================================================================
# Setup Go

NPROCS ?= 1
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis
GO111MODULE = on
GOLANGCILINT_VERSION = 2.12.2
-include build/makelib/golang.mk

# ====================================================================================
# Setup Kubernetes tools

# Use Helm 3. Without this the k8s_tools machinery defaults to Helm 2 and
# cluster/local/integration_tests.sh fails on `helm repo add`/`helm install`.
USE_HELM := true

# Default kindest/node tag. KIND v0.23.0 (the submodule's pinned version)
# supports kindest/node tags up to v1.30.0. Override via env if needed.
KIND_NODE_IMAGE_TAG ?= v1.30.0

-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Images

IMAGES = provider-sonarqube
-include build/makelib/imagelight.mk

# ====================================================================================
# Setup XPKG

XPKG_REG_ORGS ?= xpkg.upbound.io/crossplane ghcr.io/crossplane-contrib
# NOTE(hasheddan): skip promoting on xpkg.upbound.io as channel tags are
# inferred.
XPKG_REG_ORGS_NO_PROMOTE ?= xpkg.upbound.io/crossplane ghcr.io/crossplane-contrib
XPKGS = provider-sonarqube
-include build/makelib/xpkg.mk

# ====================================================================================
# Setup Local Development & E2E

# Pin the kind cluster name used by both controlplane.up and
# local.xpkg.deploy.provider.* so cluster/local/integration_tests.sh and
# the build submodule agree on which cluster to deploy into. The default
# in the submodule is "local-dev"; we want a build-id-tagged name so
# parallel CI runs do not collide.
KIND_CLUSTER_NAME ?= $(BUILD_REGISTRY)-inttests

# Pin the Crossplane chart version. The submodule's controlplane.up runs
# `helm install ... --version $(CROSSPLANE_VERSION)`, which fails with
# "flag needs an argument" when CROSSPLANE_VERSION is empty. Pinning here
# also keeps e2e runs reproducible.
CROSSPLANE_VERSION ?= 2.2.1

-include build/makelib/controlplane.mk
-include build/makelib/local.xpkg.mk

# NOTE(hasheddan): we force image building to happen prior to xpkg build so that
# we ensure image is present in daemon.
xpkg.build.provider-sonarqube: do.build.images

fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# integration tests
e2e.run: test-integration

# Run integration tests.
test-integration: $(KIND) $(KUBECTL) $(CROSSPLANE_CLI) $(HELM)
	@$(INFO) running integration tests using kind $(KIND_VERSION)
	@KIND_NODE_IMAGE_TAG=$(KIND_NODE_IMAGE_TAG) $(ROOT_DIR)/cluster/local/integration_tests.sh || $(FAIL)
	@$(OK) integration tests passed

# Run the Go e2e suite against an already-provisioned cluster.
# Expects SONARQUBE_URL and SONARQUBE_TOKEN to be exported by the caller;
# cluster/local/integration_tests.sh sets these up around its invocation.
E2E_TEST_TIMEOUT ?= 30m
e2e.test:
	@$(INFO) running e2e Go suite with build tag e2e
	@go test -tags=e2e -timeout=$(E2E_TEST_TIMEOUT) ./internal/test/e2e/... || $(FAIL)
	@$(OK) e2e Go suite passed

# Run the Go e2e suite against an Enterprise Edition SonarQube instance.
# The enterprise build tag pulls in the license/portfolio tests on top of
# the full suite already covered by e2e.test - both run against the
# instance pointed to by SONARQUBE_URL/SONARQUBE_TOKEN/SONARQUBE_PROVIDERCONFIG.
# SONARQUBE_LICENSE_KEY is optional: license/portfolio tests skip themselves
# when it is unset, since a valid Enterprise license is a paid artifact.
e2e.test.enterprise:
	@$(INFO) running enterprise e2e Go suite with build tags e2e,enterprise
	@go test -tags=e2e,enterprise -timeout=$(E2E_TEST_TIMEOUT) ./internal/test/e2e/... || $(FAIL)
	@$(OK) enterprise e2e Go suite passed

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# NOTE(hasheddan): the build submodule currently overrides XDG_CACHE_HOME in
# order to force the Helm 3 to use the .work/helm directory. This causes Go on
# Linux machines to use that directory as the build cache as well. We should
# adjust this behavior in the build submodule because it is also causing Linux
# users to duplicate their build cache, but for now we just make it easier to
# identify its location in CI so that we cache between builds.
go.cachedir:
	@go env GOCACHE

go.mod.cachedir:
	@go env GOMODCACHE

# NOTE(hasheddan): we must ensure up is installed in tool cache prior to build
# as including the k8s_tools machinery prior to the xpkg machinery sets UP to
# point to tool cache.
build.init: $(CROSSPLANE_CLI)

# This is for running out-of-cluster locally, and is for convenience. Running
# this make target will print out the command which was used. For more control,
# try running the binary directly with different arguments.
run: go.build
	@$(INFO) Running Crossplane locally out-of-cluster . . .
	@# To see other arguments that can be provided, run the command with --help instead
	$(GO_OUT_DIR)/provider --debug

dev: $(KIND) $(KUBECTL)
	@$(INFO) Creating kind cluster
	@$(KIND) create cluster --name=$(PROJECT_NAME)-dev
	@$(KUBECTL) cluster-info --context kind-$(PROJECT_NAME)-dev
	@$(INFO) Installing Provider SonarQube CRDs
	@$(KUBECTL) apply -R -f package/crds
	@$(INFO) Starting Provider SonarQube controllers
	@$(GO) run cmd/provider/main.go --debug

dev-clean: $(KIND) $(KUBECTL)
	@$(INFO) Deleting kind cluster
	@$(KIND) delete cluster --name=$(PROJECT_NAME)-dev

.PHONY: submodules fallthrough test-integration e2e.run e2e.test e2e.test.enterprise run dev dev-clean

# ====================================================================================
# Special Targets

# Install gomplate
GOMPLATE_VERSION := 3.10.0
GOMPLATE := $(TOOLS_HOST_DIR)/gomplate-$(GOMPLATE_VERSION)

$(GOMPLATE):
	@$(INFO) installing gomplate $(SAFEHOSTPLATFORM)
	@mkdir -p $(TOOLS_HOST_DIR)
	@curl -fsSLo $(GOMPLATE) https://github.com/hairyhenderson/gomplate/releases/download/v$(GOMPLATE_VERSION)/gomplate_$(SAFEHOSTPLATFORM) || $(FAIL)
	@chmod +x $(GOMPLATE)
	@$(OK) installing gomplate $(SAFEHOSTPLATFORM)

export GOMPLATE

# This target prepares repo for your provider by replacing all "sonarqube"
# occurrences with your provider name.
# This target can only be run once, if you want to rerun for some reason,
# consider stashing/resetting your git state.
# Arguments:
#   provider: Camel case name of your provider, e.g. GitHub, PlanetScale
provider.prepare:
	@[ "${provider}" ] || ( echo "argument \"provider\" is not set"; exit 1 )
	@PROVIDER=$(provider) ./hack/helpers/prepare.sh

# This target adds a new api type and its controller.
# You would still need to register new api in "apis/<provider>.go" and
# controller in "internal/controller/<provider>.go".
# Arguments:
#   provider: Camel case name of your provider, e.g. GitHub, PlanetScale
#   group: API group for the type you want to add.
#   kind: Kind of the type you want to add
#	apiversion: API version of the type you want to add. Optional and defaults to "v1alpha1"
provider.addtype: $(GOMPLATE)
	@[ "${provider}" ] || ( echo "argument \"provider\" is not set"; exit 1 )
	@[ "${group}" ] || ( echo "argument \"group\" is not set"; exit 1 )
	@[ "${kind}" ] || ( echo "argument \"kind\" is not set"; exit 1 )
	@PROVIDER=$(provider) GROUP=$(group) KIND=$(kind) APIVERSION=$(apiversion) PROJECT_REPO=$(PROJECT_REPO) ./hack/helpers/addtype.sh

define CROSSPLANE_MAKE_HELP
Crossplane Targets:
    submodules            Update the submodules, such as the common build scripts.
    run                   Run crossplane locally, out-of-cluster. Useful for development.

endef
# The reason CROSSPLANE_MAKE_HELP is used instead of CROSSPLANE_HELP is because the crossplane
# binary will try to use CROSSPLANE_HELP if it is set, and this is for something different.
export CROSSPLANE_MAKE_HELP

crossplane.help:
	@echo "$$CROSSPLANE_MAKE_HELP"

help-special: crossplane.help

.PHONY: crossplane.help help-special
