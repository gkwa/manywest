BIN := manywest
PREFIX := github.com/gkwa/manywest/version

SRC := $(wildcard *.go **/*.go)

DATE := $(shell date +"%Y-%m-%dT%H:%M:%SZ")
GOVERSION := $(shell go version)
VERSION := $(shell git describe --tags --abbrev=8 --dirty --always --long)
SHORT_SHA := $(shell git rev-parse --short HEAD)
FULL_SHA := $(shell git rev-parse HEAD)
export GOVERSION # goreleaser wants this

LDFLAGS = -s -w
LDFLAGS += -X $(PREFIX).Version=$(VERSION)
LDFLAGS += -X '$(PREFIX).Date=$(DATE)'
LDFLAGS += -X '$(PREFIX).GoVersion=$(GOVERSION)'
LDFLAGS += -X $(PREFIX).ShortGitSHA=$(SHORT_SHA)
LDFLAGS += -X $(PREFIX).FullGitSHA=$(FULL_SHA)

.DEFAULT_GOAL := iterate

all: check $(BIN) install

.PHONY: iterate # lint and rebuild
iterate: check $(BIN)

.PHONY: check # lint and vet
check: .timestamps/.check.time

.timestamps/.check.time: tidy fmt lint vet
	@mkdir -p .timestamps
	@touch $@

.PHONY: build # build
build: $(BIN)

$(BIN): .timestamps/.build.time
	go build -ldflags "$(LDFLAGS)" -o $@ cmd/main.go

.timestamps/.build.time: $(SRC)
	@mkdir -p .timestamps
	@touch $@

.PHONY: goreleaser # run goreleaser
goreleaser: goreleaser --clean

.PHONY: tidy # go tidy
tidy: .timestamps/.tidy.time

.timestamps/.tidy.time: go.mod go.sum
	go mod tidy
	@mkdir -p .timestamps
	@touch $@

.PHONY: fmt # go fmt
fmt: .timestamps/.fmt.time

.timestamps/.fmt.time: $(SRC)
	gofumpt -w $(SRC)
	@mkdir -p .timestamps
	@touch $@

.PHONY: lint # lint
lint: .timestamps/.lint.time

.timestamps/.lint.time: $(SRC)
	golangci-lint run
	@mkdir -p .timestamps
	@touch $@

.PHONY: vet # go vet
vet: .timestamps/.vet.time

.timestamps/.vet.time: $(SRC)
	go vet ./...
	@mkdir -p .timestamps
	@touch $@

.PHONY: install # go install
install:
	go install -ldflags "$(LDFLAGS)"

.PHONY: help # show makefile rules
help:
	@grep '^.PHONY: .* #' Makefile | sed 's/\.PHONY: \(.*\) # \(.*\)/\1 \2/' | expand -t20

.PHONY: clean # clean bin
clean:
	$(RM) -r $(BIN) .timestamps
