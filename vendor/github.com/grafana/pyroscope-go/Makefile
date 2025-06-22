GO_VERSION_PRE20 := $(shell go version | awk '{print $$3}' | awk -F '.' '{print ($$1 == "go1" && int($$2) < 20)}')
TEST_PACKAGES := ./... ./godeltaprof/compat/... ./godeltaprof/...

.PHONY: test
test:
	go test -race $(shell go list $(TEST_PACKAGES) | grep -v /example)

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
	cd godeltaprof/compat/ && gotip get -d -v golang.org/x/tools@v0.25.0
	git --no-pager diff
	! git diff | grep toolchain
