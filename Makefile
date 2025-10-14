.DEFAULT_GOAL := help

# SHELL := /usr/bin/env bash

# NB: many of these up-front vars need to use := to ensure that we expand them once (immediately)
# rather than re-running these (marginally expensive) commands each time the var is referenced

_GO_META = go.mod go.sum
_GO_SRCS := $(shell find . -type f -name "*.go" ) $(_GO_META)
# ignore 3rd party files (CocoaPods, Node modules, etc) and vendoreds (such as: tools/vendored-homebrew-install.sh)
_SH_SRCS := $(shell find . -type f -name "*.sh" ! -iname "vendored-*.sh" | grep -v Pods | grep -v node_modules | grep -v \.venv | grep -v .terraform/modules)
_LOCAL_PLATFORM := $(shell uname | tr '[:upper:]' '[:lower:]')

# all files recursively under <uiproject>/{src, public} (recursive) and directly under <uiproject>/ are edited by us, though 'sharedui' doesn't have public (yet).
_SHAREDUI_REACT_SRCS := $(shell find sharedui/src) $(shell find sharedui -maxdepth 1 -not -type d)
_UILIB_REACT_SRCS := $(shell find ui-component-lib/src) $(shell find ui-component-lib/public -maxdepth 1 -not -type d)
_CONSOLEUI_REACT_SRCS := $(shell find console/consoleui/src) $(shell find console/consoleui/public) $(shell find console/consoleui -maxdepth 1 -not -type d)
_PLEXUI_REACT_SRCS := $(shell find plex/plexui/src) $(shell find plex/plexui/public) $(shell find plex/plexui -maxdepth 1 -not -type d)

SERVICE_BINARIES = bin/console bin/plex bin/idp bin/authz bin/checkattribute bin/logserver bin/dataprocessor bin/worker
CODEGEN_BINARIES = bin/parallelgen bin/genconstant bin/gendbjson bin/genvalidate bin/genstringconstenum bin/genorm bin/genschemas bin/genevents bin/genrouting bin/genhandler bin/genopenapi bin/genpageable
TOOL_BINARIES = bin/automatedprovisioner bin/azcli bin/cachelookup bin/cachetool bin/cleanplextokens bin/provision bin/setcompanytype bin/tenantcopy

TF_PATH = $(if $(TG_TF_PATH),$(TG_TF_PATH),"terraform")

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

KUBECTL ?= kubectl
KUSTOMIZE_VERSION ?= v5.4.2
KUSTOMIZE = $(LOCALBIN)/kustomize

LOCALDEV_CLUSTER ?= userclouds

.PHONY: help
help: ## List user-facing Make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# This is the main target for developers. It starts all the services and the UIs.
# services-dev exists so that we don't rebuild all the react stuff in a ui-dev invocation
.PHONY:
dev: console/consoleui/build plex/plexui/build ## Run all userclouds services locally
	make services-dev

services-dev: $(SERVICE_BINARIES) bin/devbox bin/devlb
	@UC_REGION=themoon AWS_ACCESS_KEY_ID="${AWS_DEV_CREDS_AWS_KEY_ID}" AWS_SECRET_ACCESS_KEY="${AWS_DEV_CREDS_AWS_KEY_SECRET}"  bin/devbox

.PHONY: dbshell-dev
dbshell-dev: bin/tenantdbshell ## Start and connect to local db
	@tools/db-shell.sh dev

.PHONY: dbshell-prod
dbshell-prod: check-deps bin/tenantdbshell ## Connect to the production databases
	@UC_UNIVERSE=prod tools/db-shell.sh prod

.PHONY: dbshell-staging
dbshell-staging: check-deps bin/tenantdbshell ## Connect to the staging databases
	@UC_UNIVERSE=staging tools/db-shell.sh staging

dbshell-debug: check-deps bin/tenantdbshell
	@UC_UNIVERSE=debug tools/db-shell.sh debug

# NB: we no longer build all the codegen binaries themselves here because they are
# now run in go routines in parallelgen to speed up package loading
.PHONY: codegen
codegen: bin/parallelgen ## Run codegen to update generated files
	go install github.com/userclouds/easyjson/...@v0.9.0-uc6
	parallelgen

# if you need to run codegen serially, you can use this target (but it's much slower)
# you can also run an individual codegen operation by running the command after
# //go:generate in a specific file, you just need to ensure you've build that binary
# this is useful if you're debugging codegen, or if you are adding a ton (hundreds?)
# of new logging event codes and hitting conflicts because genevents runs in parallel
# across services.
codegen-serial: $(CODEGEN_BINARIES) ## Run codegen to update generated files
	go generate ./...
	make gen-openapi-spec

# TODO: move away from local tooling as a supported option
TOOL_DEPS=jq yq direnv bash curl git-lfs hub awscli n yarn postgresql@14 tmux \
	restack python3 terraform tflint terragrunt redis gh \
	helm kubernetes-cli kubeconform argocd docker \
	coreutils # for timeout (used in redis-shell.sh)
check-deps:
	@tools/check-deps.sh $(TOOL_DEPS)

.PHONY: clean ## Clean up rebuildable binaries
clean:
    # our service binaries
	-rm -f $(SERVICE_BINARIES)

    # tools
	-rm -f bin/goimports
	-rm -f bin/staticcheck
	-rm -f bin/errcheck
	-rm -f bin/shellcheck
	-rm -f bin/shfmt
	-rm -f bin/uclint
	-rm -f cover.out
	-rm -rf heap_profiler

    # generated
	-rm -f $(CODEGEN_BINARIES)

######################## test runner ######################
# Test DB should create a new DB / store per test run, so we can parallelize
#   The store itself is in memory for perf, but the dir is still useful for interacting with it
# Store the connection URL in a file so we can link it in
# Note that all the DB setup happens in the test target so we don't
# create random empty dirs during `make dev` etc :)
# We also source tools/devenv.sh in order to get AWS creds for secret manager, which is
#  a bit awkward given the way Make creates a shell environment per line and doesn't let them export
# TODO: we actually shouldn't source devenv.sh in CI, but since CI runs sh (not bash) it doesn't actually hurt
# TODO: should UI tests get pulled out to a separate target at some point? We could have `make servertest`, `make uitest`,
#  and `make test` can just depend on both (so then TESTARGS would only apply to `make servertest`).
# Use TESTARGS to run to eg. a specific test / package tests, `TESTARGS=./idp/internal/authn make test`
#  or `TESTARGS="./plex/internal -run TestCaseFoo"` make test for a single test
.PHONY: test
test: TESTARGS ?= ./...
test: TESTENV ?= test   # CI uses this to override UC_UNIVERSE
test: _TEMPDIR := $(shell mktemp -d)
test: _TEMPFILE := $(_TEMPDIR)/testdb
test: _TESTDB_STOP = docker rm -f testdb
test: ## Build project and run test suite
	@tools/setup-test-db.sh $(_TEMPFILE)
	@if [ "$(strip $(TESTENV))" == "test" ]; then\
		tools/start-redis.sh;\
	else\
		echo "skipping redis because TESTENV was specified ($(TESTENV))";\
	fi
	UC_UNIVERSE=$(TESTENV) UC_REGION=mars UC_TESTDB_POINTER_PATH=$(_TEMPFILE) go test \
	         -race \
			 -coverprofile=cover.out \
			 -vet=off \
			 $(TESTARGS) || ( $(_TESTDB_STOP) && exit 1)
	@$(_TESTDB_STOP)
	@if [ "$(strip $(TESTARGS))" == './...' ]; then\
		make consoleui-test;\
	else\
		echo "skipping UI tests because TESTARGS was specified ($(TESTARGS))";\
	fi

# Very similar target to the "test" target above that we will use in CI
# few chnages from the regular test target:
# * Runs all tests (no TESTARGS support)
# * No cleanup (stopping test DB after tests)
# * Will not try to start redis (CI runs them as services already)
# * Will only run backend (golang) test and not the UI tests
.PHONY: test-backend-ci
test-backend-ci: TESTARGS ?= ./...
test-backend-ci: _TEMPFILE := $(shell mktemp -d)/testdb
test-backend-ci: ## Build project and run test suite
	@tools/setup-test-db.sh $(_TEMPFILE)
	UC_UNIVERSE=ci UC_REGION=mars UC_TESTDB_POINTER_PATH=$(_TEMPFILE) go test \
	         -timeout 20m -parallel 4 -race -coverprofile=cover.out -vet=off $(TESTARGS)

.PHONY: test-provisioning
test-provisioning: bin/provision ## Test tenant & db provisioning
	tools/provision-test.sh

test-helm:
	./helm/test-charts.sh

test-fixme:
	tools/check-fixme.sh

test-codegen:
	UC_CONFIG_DIR=./config tools/check-codegen.sh

check-go-modules:
	tools/check-go-modules.sh

######################### linters ##########################

.PHONY: lint
lint: ## Lint code and config
	@tools/lint.sh

lint-golang: bin/goimports bin/staticcheck bin/errcheck bin/revive bin/uclint bin/modernize
	@tools/lint-golang.sh

lint-frontend: sharedui/build ui-lib/build # TODO: this is a required dep because the build generates *.d.ts files needed to lint downstream modules. We may want to check these in to git in the future?
	@tools/lint-frontend.sh

lint-shell: bin/shfmt bin/shellcheck
	@tools/lint-shell.sh "$(_SH_SRCS)"

lint-python:
	@tools/lint-python.sh

lint-sql:
	@tools/lint-sql.sh

.PHONY: lintfix
lintfix: bin/goimports bin/shfmt ## Automatically fix some lint problems
lintfix: sharedui/build ui-lib/build # TODO: this is a required dep because the build generates *.d.ts files needed to lint downstream modules. We may want to check these in to git in the future?
	@tools/lintfix.sh "$(_SH_SRCS)"

######################### logging ##########################
.PHONY: log-prod
log-prod: bin/uclog
	UC_UNIVERSE=prod tools/ensure-aws-auth.sh
	bin/uclog --time 5 --streamname prod --live --verbose --outputpref sh --interactive --summary --ignorehttpcode 401,409 listlog

.PHONY: log-staging
log-staging: bin/uclog
	UC_UNIVERSE=staging tools/ensure-aws-auth.sh
	bin/uclog --time 5 --streamname staging --live  --verbose --outputpref sh --interactive --summary listlog

.PHONY: log-debug
log-debug: bin/uclog
	UC_UNIVERSE=debug tools/ensure-aws-auth.sh
	bin/uclog --time 5 --streamname debug --live --verbose --outputpref sh --interactive --summary listlog

######################### image binaries ##########################
.PHONY: image-binaries
image-binaries: bin/userclouds bin/consoleuiinitdata $(TOOL_BINARIES)

######################### service binaries ##########################
bin/userclouds: $(_GO_SRCS)
	go build --trimpath -o $@ \
    		-ldflags \
    			"-s -w -X userclouds.com/infra/service.buildHash=$(shell git rev-parse HEAD) \
    			 -X userclouds.com/infra/service.buildTime=$(shell TZ=UTC git show -s --format=%cd --date=iso-strict-local HEAD)" \
    		./userclouds/cmd

$(SERVICE_BINARIES): $(_GO_SRCS)
	go build --trimpath -o $@ \
		-ldflags \
			"-s -w -X userclouds.com/infra/service.buildHash=$(shell git rev-parse HEAD) \
			 -X userclouds.com/infra/service.buildTime=$(shell TZ=UTC git show -s --format=%cd --date=iso-strict-local HEAD)" \
		./$(notdir $@)/cmd

######################### tool binaries ##########################
$(TOOL_BINARIES): $(_GO_SRCS)
	go build --trimpath -o $@ -ldflags "-s -w" ./cmd/$(notdir $@)

######################### code gen binaries #########################
$(CODEGEN_BINARIES): $(_GO_SRCS)
	@echo "building $@"
	go build -o $@ ./cmd/$(notdir $@)

######################### react ui stuff #########################

# Install/update dependencies (node modules) in development environments. This creates/updates the `node_modules`
# directories wherever there are `package.json` files in our tree, and creates/updates the `yarn.lock` file
# which tracks metadata for all installed modules. `package.json` is the source of truth for which dependencies/modules
# to fetch and what versions to use, but re-running this may alter the `yarn.lock` file as dependencies change as
# we don't always pin specific versions.
.PHONY: ui-yarn-install
ui-yarn-install: ## Install/update dependencies for our React UI projects (needed if adding new deps)
	uv run yarn install
	@tools/install-playwright.sh

# Install dependencies and reqs needed to build UI bundles and run UI tests (playwright) in CI.
.PHONY: ui-yarn-ci
ui-yarn-ci:
	time yarn install --immutable
	time tools/install-playwright.sh

# Install/update dependencies (node modules) in CI / Build pipelines for UI apps & libraries.
# It is very similar to `ui-yarn-install` except it treats the `package.json` and `yarn.lock` files as read-only
# and ensures that all modules are reinstalled from scratch. Hence its suitability for CI/Build jobs but why it
# isn't used in normal development.
# https://stackoverflow.com/questions/52499617/what-is-the-difference-between-npm-install-and-npm-ci
# and https://stackoverflow.com/questions/58482655/what-is-the-closest-to-npm-ci-in-yarn
ui-yarn-build-only-ci:
	time yarn install --immutable

sharedui/build: $(_SHAREDUI_REACT_SRCS)
	@rm -rf sharedui/build
	yarn sharedui:build

ui-lib/build: $(_UILIB_REACT_SRCS)
	yarn ui-lib:build # this does the rm -rf part itself

console/consoleui/build: $(_CONSOLEUI_REACT_SRCS) sharedui/build ui-lib/build
	@rm -rf console/consoleui/build
	yarn consoleui:build
	UC_CONFIG_DIR=config/ go run cmd/consoleuiinitdata/main.go

plex/plexui/build: $(_PLEXUI_REACT_SRCS) sharedui/build ui-lib/build
	@rm -rf plex/plexui/build
	yarn plexui:build

.PHONY: ui-build
ui-build: sharedui/build ui-lib/build console/consoleui/build plex/plexui/build ## Build static asset bundles for all React UI projects

.PHONY: ui-clean
ui-clean: ## Clean the output (build) dirs of our React UI projects
	@rm -rf sharedui/build ui-component-lib/build console/consoleui/build plex/plexui/build

.PHONY: ui-yarn-clean
ui-yarn-clean: ui-clean ## Clean yarn generated/downloaded files. Must re-run `make ui-yarn-install` after
	@rm -rf node_modules sharedui/node_modules ui-component-lib/node_modules console/consoleui/node_modules plex/plexui/node_modules
	@rm -rf yarn.lock sharedui/yarn.lock ui-component-lib/yarn.lock console/consoleui/yarn.lock plex/plexui/yarn.lock

.PHONY: sharedui-dev
sharedui-dev: ## Run the React rollup server in watch mode for 'sharedui'
	yarn sharedui:dev

.PHONY: ui-lib-watch
ui-lib-watch: ## Run the React rollup server in watch mode for 'ui-component-lib'
	yarn ui-lib:watch

.PHONY: ui-lib-dev
ui-lib-dev: ## Run the React rollup server in watch mode for 'ui-component-lib'
	yarn ui-lib:dev

.PHONY: consoleui-dev
consoleui-dev: sharedui/build ## Run the React development server for 'consoleui'
	yarn consoleui:dev

.PHONY: plexui-dev
plexui-dev: sharedui/build ## Run the React development server for 'plexui'
	BROWSER=none yarn plexui:dev

.PHONY: consoleui-test
consoleui-test: ## Run the tests for 'consoleui'
	make sharedui/build ui-lib/build
	CI=1 yarn consoleui:test

.PHONY: ui-dev
ui-dev: ## Run the dev backend + react dev server for plex & console
	tmux new-session "tmux source-file tools/tmux-uidev.cmd"

bin/containerrunner: $(_GO_SRCS)
	go build -o $@ ./cmd/containerrunner

tools: bin/goimports
bin/goimports: go.mod
	go install -mod=readonly golang.org/x/tools/cmd/goimports

tools: bin/modernize
bin/modernize: go.mod
	go install -mod=readonly golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@v0.18.0

tools: bin/shfmt
# When upgrading shfmt, make sure to update the version the cache key in .github/workflows/lint-shell.yml
bin/shfmt: go.mod
	go install -mod=readonly mvdan.cc/sh/v3/cmd/shfmt@v3.11.0

# Note that in CI, we untar shellcheck directly into bin/ so we don't polute our git porcelain status :)
tools: bin/shellcheck
bin/shellcheck:
ifeq ($(_LOCAL_PLATFORM), darwin)
	shellcheck --version || brew install shellcheck
	ln -s $$(which shellcheck) bin/shellcheck
else ifeq ($(_LOCAL_PLATFORM), linux)
	tools/install-shellcheck-linux.sh
else
	$(error "Don't know how to download shellcheck on $(_LOCAL_PLATFORM)")
endif

tools: bin/staticcheck
bin/staticcheck: go.mod
	go install -mod=readonly honnef.co/go/tools/cmd/staticcheck@2025.1.1

tools: bin/revive
bin/revive: go.mod
	go install -mod=readonly github.com/mgechev/revive

tools: bin/errcheck
bin/errcheck: go.mod
	go install -mod=readonly github.com/kisielk/errcheck

tools: bin/uclint
bin/uclint: $(_GO_SRCS)
	go build -o $@ userclouds.com/cmd/uclint

tools:bin/migrate
bin/migrate: $(_GO_SRCS)
	go build -o bin/migrate ./cmd/migrate

tools: bin/tenantdbshell
bin/tenantdbshell: $(_GO_SRCS)
	go build -o bin/tenantdbshell ./cmd/tenantdbshell

tools: bin/queryrunner
bin/queryrunner: $(_GO_SRCS)
	go build -o bin/queryrunner ./cmd/queryrunner

tools: bin/dataimport
bin/dataimport: $(_GO_SRCS)
	go build -o bin/dataimport ./cmd/dataimport

tools: bin/cleanupusercolumns
bin/cleanupusercolumns: $(_GO_SRCS)
	go build -o bin/cleanupusercolumns ./cmd/cleanupusercolumns

tools: bin/uclog
bin/uclog: $(_GO_SRCS)
	go build -o bin/uclog ./cmd/uclog

tools: bin/testdevcert
bin/testdevcert: $(_GO_SRCS)
	go build -o bin/testdevcert ./cmd/testdevcert

tools: bin/consoleuiinitdata
bin/consoleuiinitdata: $(_GO_SRCS)
	go build -o bin/consoleuiinitdata ./cmd/consoleuiinitdata

tools: bin/runaccessors
bin/runaccessors: $(_GO_SRCS)
	go build -o bin/runaccessors ./cmd/runaccessors

tools: bin/envtestecs
bin/envtestecs: $(_GO_SRCS)
	go build -o bin/envtestecs ./cmd/envtestecs

bin/auditlogview: $(_GO_SRCS)
	go build -o bin/auditlogview ./cmd/auditlogview

bin/remoteuserregionconfig: $(_GO_SRCS)
	go build -o bin/remoteuserregionconfig ./cmd/remoteuserregionconfig

.PHONY:
build-deploy-binaries: $(SERVICE_BINARIES) bin/consoleuiinitdata
	echo "Built binaries for deployment"

.PHONY:
gen-openapi-spec: ## Generate the consolidated OpenAPI spec for our APIs
	go run cmd/genopenapi/main.go

.PHONY: tf-provider-build
tf-provider-build: ## Build the Terraform provider
	$(MAKE) -C ./public-repos/terraform-provider-userclouds build

######################### helm ##########################
helm-dep-update:
	helm dep update helm/charts/userclouds

######################### local development ##########################
.PHONY: localdev
localdev: deps localdev-cluster localdev-shared

.PHONY: localdev-cluster
localdev-cluster:
	@if k3d cluster get $(LOCALDEV_CLUSTER) --no-headers >/dev/null 2>&1;  \
		then echo "Cluster exists, skipping creation"; \
		else k3d cluster create --config config/localdev/k3d/config.yaml --volume $(PWD):/app; \
		fi

.PHONY: localdev-shared
localdev-shared:
	@$(KUSTOMIZE) build config/localdev/userclouds/kubegres | envsubst | $(KUBECTL) apply --server-side -f -
	@$(KUSTOMIZE) build config/localdev/userclouds/cert-manager | envsubst | $(KUBECTL) apply --server-side -f -
	@$(KUBECTL) wait --for=condition=available --timeout=120s deploy -l app.kubernetes.io/group=cert-manager -n cert-manager
	@$(KUSTOMIZE) build config/localdev/userclouds/localstack | envsubst | $(KUBECTL) apply --server-side -f -
	@$(KUSTOMIZE) build config/localdev/userclouds/postgres | envsubst | $(KUBECTL) apply --server-side -f -
	@$(KUBECTL) wait --for=condition=available --timeout=120s deploy/localstack -n userclouds

.PHONY: localdev-clean
localdev-clean:
	@k3d cluster delete userclouds

.PHONY: localdev-helm
localdev-helm: helm-dep-update
	helm upgrade --install userclouds helm/charts/userclouds \
		--kube-context k3d-$(LOCALDEV_CLUSTER) \
		--namespace userclouds \
		--create-namespace \
		--values helm/charts/userclouds/values.yaml \
		--values helm/charts/userclouds/values-localdev.yaml

######################### deps ##########################
deps: $(LOCALBIN) $(KUSTOMIZE)

$(KUSTOMIZE):
	@test -s $(KUSTOMIZE) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)
