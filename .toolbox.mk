## toolbox - start
## Generated with https://github.com/bakito/toolbox

## Current working directory
TB_LOCALDIR ?= $(shell which cygpath > /dev/null 2>&1 && cygpath -m $$(pwd) || pwd)
## Location to install dependencies to
TB_LOCALBIN ?= $(TB_LOCALDIR)/bin
$(TB_LOCALBIN):
	if [ ! -e $(TB_LOCALBIN) ]; then mkdir -p $(TB_LOCALBIN); fi

## Go Version
TB_GO_VERSION ?= $(shell grep -E '^go [0-9]+\.[0-9]+' go.mod | awk '{print $$2}')

# Helper functions
STRIP_V = $(patsubst v%,%,$(1))

## Tool Binaries
TB_GOLANGCI_LINT ?= $(TB_LOCALBIN)/golangci-lint

## Tool Versions
# renovate: packageName=github.com/golangci/golangci-lint/v2
TB_GOLANGCI_LINT_VERSION ?= v2.12.2
TB_GOLANGCI_LINT_VERSION_NUM ?= $(call STRIP_V,$(TB_GOLANGCI_LINT_VERSION))

## Tool Installer
.PHONY: tb.golangci-lint
tb.golangci-lint: ## Download golangci-lint locally if necessary.
	@test -s $(TB_GOLANGCI_LINT) && $(TB_GOLANGCI_LINT) --version | grep -q $(TB_GOLANGCI_LINT_VERSION_NUM) || \
		GOBIN=$(TB_LOCALBIN) GOTOOLCHAIN=go$(TB_GO_VERSION) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(TB_GOLANGCI_LINT_VERSION)

## Reset Tools
.PHONY: tb.reset
tb.reset:
	@rm -f \
		$(TB_GOLANGCI_LINT)

## Update Tools
.PHONY: tb.update
tb.update: tb.reset
	toolbox makefile --renovate --toolchain -f $(TB_LOCALDIR)/Makefile \
		github.com/golangci/golangci-lint/v2/cmd/golangci-lint?--version
## toolbox - end
