SHELL 	   := $(shell which bash)

## BOF define block

BINARIES   := pluralize
BINARY     = $(word 1, $@)

PLATFORMS  := windows linux darwin
PLATFORM   = $(word 1, $@)

ROOT_DIR   := $(shell git rev-parse --show-toplevel)
BIN_DIR    := $(ROOT_DIR)/bin
REL_DIR    := $(ROOT_DIR)/release
SRC_DIR    := $(ROOT_DIR)/cmd
INC_DIR    := $(ROOT_DIR)/include
TMP_DIR    := $(ROOT_DIR)/tmp

VERSION    :=`git describe --tags 2>/dev/null`
COMMIT     :=`git rev-parse --short HEAD 2>/dev/null`
DATE       :=`date "+%FT%T%z"`

LDBASE     := github.com/gertd/go-pluralize/pkg/version
LDFLAGS    := -ldflags "-w -s -X $(LDBASE).ver=${VERSION} -X $(LDBASE).date=${DATE} -X $(LDBASE).commit=${COMMIT}"

GOARCH     ?= amd64
GOOS       ?= $(shell go env GOOS)

LINTER     := $(BIN_DIR)/golangci-lint
LINTVERSION:= v1.27.0

TESTRUNNER := $(BIN_DIR)/gotestsum
TESTVERSION:= v0.5.0

PROTOC     := $(BIN_DIR)/protoc
PROTOCVER  := 3.12.3

NO_COLOR   :=\033[0m
OK_COLOR   :=\033[32;01m
ERR_COLOR  :=\033[31;01m
WARN_COLOR :=\033[36;01m
ATTN_COLOR :=\033[33;01m

## EOF define block

.PHONY: all
all: deps gen build test lint

deps:
	@echo -e "$(ATTN_COLOR)==> $@ $(NO_COLOR)"
	@GO111MODULE=on go mod download

.PHONY: gen
gen: deps $(BIN_DIR)
	@echo -e "$(ATTN_COLOR)==> $@ $(NO_COLOR)"
	@go generate ./...

.PHONY: dobuild
dobuild:
	@echo -e "$(ATTN_COLOR)==> $@ $(B) GOOS=$(P) GOARCH=$(GOARCH) VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) $(NO_COLOR)"
	@GOOS=$(P) GOARCH=$(GOARCH) GO111MODULE=on go build $(LDFLAGS) -o $(T)/$(P)-$(GOARCH)/$(B)$(if $(findstring $(P),windows),".exe","") $(SRC_DIR)/$(B)
ifneq ($(P),windows)
	@chmod +x $(T)/$(P)-$(GOARCH)/$(B)
endif

.PHONY: build 
build: $(BIN_DIR) deps
	@echo -e "$(ATTN_COLOR)==> $@ $(NO_COLOR)"
	@for b in ${BINARIES}; 									\
	do 														\
		$(MAKE) dobuild B=$${b} P=${GOOS} T=${BIN_DIR}; 	\
	done 													

.PHONY: doinstall
doinstall:
	@echo -e "$(ATTN_COLOR)==> $@ $(B) GOOS=$(P) GOARCH=$(GOARCH) VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) $(NO_COLOR)"
	@GOOS=$(P) GOARCH=$(GOARCH) GO111MODULE=on go install $(LDFLAGS) $(SRC_DIR)/$(B)

.PHONY: install
install: 
	@echo -e "$(ATTN_COLOR)==> $@ $(NO_COLOR)"
	@for b in ${BINARIES}; 									\
	do 														\
		$(MAKE) doinstall B=$${b} P=${GOOS}; 			 	\
	done 													

.PHONY: dorelease
dorelease:
	@echo -e "$(ATTN_COLOR)==> $@ build GOOS=$(P) GOARCH=$(GOARCH) VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) $(NO_COLOR)"
	@GOOS=$(P) GOARCH=$(GOARCH) GO111MODULE=on go build $(LDFLAGS) -o $(T)/$(P)-$(GOARCH)/$(B)$(if $(findstring $(P),windows),".exe","") $(SRC_DIR)/$(B)
ifneq ($(P),windows)
	@chmod +x $(T)/$(P)-$(GOARCH)/$(B)
endif
	@echo -e "$(ATTN_COLOR)==> $@ zip $(B)-$(P)-$(GOARCH).zip $(NO_COLOR)"
	@zip -j $(T)/$(P)-$(GOARCH)/$(B)-$(P)-$(GOARCH).zip $(T)/$(P)-$(GOARCH)/$(B)$(if $(findstring $(P),windows),".exe","") >/dev/null

.PHONY: release
release: $(REL_DIR)
	@echo -e "$(ATTN_COLOR)==> $@ $(NO_COLOR)"
	@for b in ${BINARIES}; 									\
	do 														\
		for p in ${PLATFORMS};								\
		do 													\
			$(MAKE) dorelease B=$${b} P=$${p} T=${REL_DIR}; 	\
		done;												\
	done 													\

$(TESTRUNNER):
	@echo -e "$(ATTN_COLOR)==> get $@  $(NO_COLOR)"
	@GOBIN=$(BIN_DIR) go get -u gotest.tools/gotestsum

.PHONY: test 
test: $(TESTRUNNER)
	@echo -e "$(ATTN_COLOR)==> $@ $(NO_COLOR)"
	@CGO_ENABLED=0 $(BIN_DIR)/gotestsum --format short-verbose -- -count=1 -v $(ROOT_DIR)/...

$(LINTER):
	@echo -e "$(ATTN_COLOR)==> get $@  $(NO_COLOR)"
	@curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s $(LINTVERSION)
 
.PHONY: lint
lint: $(LINTER)
	@echo -e "$(ATTN_COLOR)==> $@ $(NO_COLOR)"
	@CGO_ENABLED=0 $(LINTER) run --enable-all
	@echo -e "$(NO_COLOR)\c"

.PHONY: doclean
doclean:
	@echo -e "$(ATTN_COLOR)==> $@ $(B) GOOS=$(P) $(NO_COLOR)"
	@if [ -a $(GOPATH)/bin/$(B)$(if $(findstring $(P),windows),".exe","") ];\
	then																	\
		rm $(GOPATH)/bin/$(B)$(if $(findstring $(P),windows),".exe","");	\
	fi

.PHONY: clean
clean:
	@echo -e "$(ATTN_COLOR)==> $@ $(NO_COLOR)"
	@rm -rf $(BIN_DIR)
	@rm -rf $(REL_DIR)
	@go clean
	@for b in ${BINARIES}; 									\
	do 														\
		$(MAKE) doclean B=$${b} P=${GOOS};	 				\
	done 													

$(REL_DIR):
	@echo -e "$(ATTN_COLOR)==> create REL_DIR $(REL_DIR) $(NO_COLOR)"
	@mkdir -p $(REL_DIR)

$(BIN_DIR):
	@echo -e "$(ATTN_COLOR)==> create BIN_DIR $(BIN_DIR) $(NO_COLOR)"
	@mkdir -p $(BIN_DIR)
