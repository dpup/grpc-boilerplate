.DEFAULT_GOAL := help
.EXPORT_ALL_VARIABLES:

ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
PATH := $(ROOT_DIR)/go/bin:$(PATH)

GOPATH := $(ROOT_DIR)/go
GO_SRC := $(ROOT_DIR)/go/src
GO_BIN := $(ROOT_DIR)/go/bin
GO_MOD_ROOT := $(ROOT_DIR)/go/src/boiler.plate
GO111MODULE := on
GODEBUG=x509ignoreCN=0

PROTO_SRC := $(ROOT_DIR)/protos
CERT_SRC := $(ROOT_DIR)/certs

run: build
	@$(GO_BIN)/boiler-plate

build: gen-proto
	@cd $(GO_MOD_ROOT) && go build -o $(GO_BIN)/boiler-plate cmd/boiler-plate.go
	@echo "👷🏻 Go binary built"

gen-proto:
	@cd $(PROTO_SRC) && protoc -I$(PROTO_SRC) -I$(GO_SRC) \
		--go_out=$(GO_SRC) \
		--grpc_out=$(GO_SRC) \
		--grpc-gateway_out $(GO_SRC) \
		--grpc-gateway_opt logtostderr=true \
    --grpc-gateway_opt generate_unbound_methods=true \
		$(shell ls $(ROOT_DIR)/protos/*.proto)
	@echo "👷🏽‍♀️ Protos built"

gen-certs:
	@cd $(CERT_SRC) && ./gen-certs.sh
	@echo "🔐 Certificate created"
	@echo "ℹ️  Now Open Keychain Access"
	@echo "ℹ️  File > Import Items and select certs/rootCA.crt and server.crt"
	@echo "ℹ️  Double both the imported certificate and select 'Always Trust'"
	@echo "ℹ️  Close the certs and save your changes"
	@echo "ℹ️  You are ready to run the server now 'make run'"

install-deps:
	@cd $(GO_MOD_ROOT) && go install \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
		google.golang.org/protobuf/cmd/protoc-gen-go \
		google.golang.org/grpc/cmd/protoc-gen-go-grpc
	@cd $(GO_BIN) && cp protoc-gen-go-grpc protoc-gen-grpc # Hack to avoid "protoc-gen-grpc: program not found or is not executable" error
	@echo "🛠 Deps installed"

tidy:
	@cd $(GO_MOD_ROOT) && go mod tidy
	@echo "🧹 Mod file tidied"

env:
	go env

help:
	@echo "go-grpc-boiler-plate"
	@echo "Quickstart for building GRPC based Go services with a HTTP Gateway."
	@echo ""
	@echo "Available commands:"
	@echo "  make install-deps - Installs necessary commands in go/bin/"
	@echo "  make gen-certs    - Generates self-signed certs for dev"
	@echo "  make build        - Build the Go binary"
	@echo "  make run          - Runs the sample GRPC server"
	@echo "  make gen-proto    - Generate Go code from protos"
	@echo "  make tidy         - Updates go.mod following any changes"
