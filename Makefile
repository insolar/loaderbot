TEST_COUNT ?= 1
TEST_ARGS ?=
COVERPROFILE ?= coverage.out
BIN_DIR = bin
export GOPATH ?= $(shell go env GOPATH)
export GO111MODULE ?= on

.PHONY: lint
lint: ## run linter
	${BIN_DIR}/golangci-lint --color=always run ./... -v --timeout 5m

golangci: ## install golangci-linter
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${BIN_DIR} v1.27.0

go-acc: ## install coverage tool
	go get github.com/ory/go-acc@v0.2.3

install-deps: golangci go-acc ## install necessary dependencies

protoc_gen: ## generate protobuf
	protoc service.proto --go_out=plugins=grpc:. -I=.

.PHONY: test
test:  ## run all tests
	go test -v -run TestCommon ./... -race -count $(TEST_COUNT) $(TEST_ARGS)

.PHONY: clean
clean:  ## remove all report and debug data
	rm -rf results_csv results_html
