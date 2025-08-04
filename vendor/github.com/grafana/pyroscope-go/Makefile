TEST_PACKAGES := ./... ./godeltaprof/compat/... ./godeltaprof/...
GO ?= go
GOTIP ?= gotip

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
