# Include toolbox tasks
include ./.toolbox.mk

lint: tb.golangci-lint
	$(TB_GOLANGCI_LINT) run --fix

# Run go mod tidy
tidy:
	go mod tidy

# Run tests
test: tidy lint test-ci

test-update:
	go test -v ./inventory/ -update-inventory-tests

test-ci:
	go test ./... -v -coverprofile=coverage.out
	go tool cover -func coverage.out

generate:
	cd types &&	go generate
