TEST_PACKAGES := ./... ./godeltaprof/compat/... ./godeltaprof/...
GO ?= go
GOTIP ?= gotip

GOLANGCI_LINT_VERSION ?= v2.2.2
TOOLS_DIR := $(CURDIR)/.tools
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint

.PHONY: test
test:
	$(GO) test -race $(shell $(GO) list $(TEST_PACKAGES) | grep -v /example)

.PHONY: go/mod
go/mod:
	GO111MODULE=on go mod download
	go work sync
	GO111MODULE=on go mod tidy
	cd godeltaprof/compat/ && GO111MODULE=on go mod download
	cd godeltaprof/compat/ && GO111MODULE=on go mod tidy
	cd godeltaprof/ && GO111MODULE=on go mod download
	cd godeltaprof/ && GO111MODULE=on go mod tidy

# https://github.com/grafana/pyroscope-go/issues/129
.PHONY: gotip/fix
gotip/fix:
	cd godeltaprof/compat/ && $(GOTIP) get -v golang.org/x/tools@v0.34.0
	git --no-pager diff
	! git diff | grep toolchain

.PHONY: install-lint
install-lint:
	@ mkdir -p $(TOOLS_DIR)
	@ GOBIN=$(TOOLS_DIR) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: lint
lint: install-lint
	$(GOLANGCI_LINT) run
	cd godeltaprof && $(GOLANGCI_LINT) run
	cd godeltaprof/compat && $(GOLANGCI_LINT) run

.PHONY: examples
examples:
	 go build example/http/main.go
	 go build example/simple/main.go
	 go build example/timing/timing.go
