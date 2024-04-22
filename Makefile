.DEFAULT_GOAL := help
.EXPORT_ALL_VARIABLES:

ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
PATH := $(ROOT_DIR)/go/bin:$(ROOT_DIR)/ts/src/node_modules/.bin:$(PATH)

GOPATH := $(ROOT_DIR)/go
GO_SRC := $(ROOT_DIR)/go/src
GO_BIN := $(ROOT_DIR)/go/bin
GO_MOD_ROOT := $(ROOT_DIR)/go/src/boiler.plate
GO111MODULE := on
GODEBUG=x509ignoreCN=0

PROTO_SRC := $(ROOT_DIR)/protos
TYPESCRIPT_SRC := $(ROOT_DIR)/ts/src
TYPESCRIPT_PUB := $(ROOT_DIR)/ts/public
TYPESCRIPT_SERVICES := $(ROOT_DIR)/ts/src/services
GEN_OUT := $(ROOT_DIR)/gen
CERT_SRC := $(GEN_OUT)/certs

run: build-backend build-static
	@echo "🚀 Running server on port 5050"
	@$(GO_BIN)/boiler-plate

build-backend: gen-proto
	@cd $(GO_MOD_ROOT) && go build -o $(GO_BIN)/boiler-plate cmd/boiler-plate.go
	@echo "👷🏻 Go binary built"

build-static: gen-proto
	@mkdir -p $(GEN_OUT)/www/
	@cp -r $(TYPESCRIPT_PUB)/* $(GEN_OUT)/www/
	@cd $(TYPESCRIPT_SRC) && tsc
	@# Hack to allow imports to work in the browser. Typescript emits imports for
	@# "greeter.pb" but it needs to be "greeter.pb.js".
	@grep -rl '.pb"' $(GEN_OUT)/www | xargs sed -i '' s/\.pb"/\.pb\.js"/g
	@echo "👷🏽‍♀️ typescript transpilation complete"

gen-proto:
	@mkdir -p $(GEN_OUT)/openapiv2
	@mkdir -p $(TYPESCRIPT_SERVICES)
	@cd $(PROTO_SRC) && protoc -I$(PROTO_SRC) -I$(GO_SRC) \
		--go_out=$(GO_SRC) \
		--grpc_out=$(GO_SRC) \
		--grpc-gateway_out $(GO_SRC) \
		--grpc-gateway_opt logtostderr=true \
    --grpc-gateway_opt generate_unbound_methods=true \
		--grpc-gateway-ts_out=ts_import_roots=$(TYPESCRIPT_SRC),ts_import_root_aliases=base:$(TYPESCRIPT_SERVICES) \
    --openapiv2_out $(GEN_OUT)/openapiv2 \
    --openapiv2_opt logtostderr=true \
    --openapiv2_opt use_go_templates=true \
    --openapiv2_opt simple_operation_ids=true \
    --openapiv2_opt openapi_naming_strategy=fqn \
    --openapiv2_opt disable_default_errors=true \
		$(shell find $(PROTO_SRC) -name "*.proto")
	@echo "👷🏽‍♀️ Proto definitions generated"

gen-certs:
	@mkdir -p $(CERT_SRC)
	@cd $(CERT_SRC) && openssl genrsa -out rootCA.key 2048
	@cd $(CERT_SRC) && openssl req -new -x509 -days 365 -nodes -sha256 -key rootCA.key -subj "/C=CN/ST=GD/L=SZ/O=GRPC Boilerplate/CN=Boilerplate CA" -out rootCA.crt
	@cd $(CERT_SRC) && openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/C=CN/ST=GD/L=SZ/O=GRPC Boilerplate/CN=localhost" -out server.csr
	@cd $(CERT_SRC) && echo 'subjectAltName=DNS:localhost,DNS:localhost' > extfile.cnf
	@cd $(CERT_SRC) && openssl x509 -req -extfile extfile.cnf -days 365 -in server.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out server.crt
	@echo "🔐 Certificate created"
	@echo "ℹ️  Now Open Keychain Access"
	@echo "ℹ️  File > Import Items and select $(CERT_SRC)/rootCA.crt and server.crt"
	@echo "ℹ️  Double both the imported certificate and select 'Always Trust'"
	@echo "ℹ️  Close the certs and save your changes"
	@echo "ℹ️  You are ready to run the server now 'make run'"

install-deps:
	@# Add go tools to go/src/tools/tools.go and use `make tidy` to fetch.
	@cd $(GO_MOD_ROOT) && go install \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
		github.com/dpup/protoc-gen-grpc-gateway-ts \
		google.golang.org/protobuf/cmd/protoc-gen-go \
		google.golang.org/grpc/cmd/protoc-gen-go-grpc
	@# Hack to avoid "protoc-gen-grpc: program not found or is not executable" error
	@cd $(GO_BIN) && cp protoc-gen-go-grpc protoc-gen-grpc
	@cd $(TYPESCRIPT_SRC) && yarn install
	@echo "🛠  Deps installed"

tidy:
	@cd $(GO_MOD_ROOT) && go mod tidy
	@echo "🧹 Mod file tidied"

clean:
	@rm -r $(GEN_OUT)/openapiv2
	@rm -r $(GEN_OUT)/www
	@find $(GO_SRC) | grep .gw.go | xargs rm
	@find $(GO_SRC) | grep .pb.go | xargs rm
	@find $(TYPESCRIPT_SERVICES) | grep .pb.ts | xargs rm
	@echo "🧹 Generated files removed"

run-docs: gen-proto
	@echo "📝 Running Swagger server on port 5051"
	@docker run -p 5051:8080 \
    -e SWAGGER_JSON=/openapiv2/greeter.swagger.json \
    -v $(PWD)/gen/openapiv2/:/openapiv2 \
    swaggerapi/swagger-ui

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
	@echo "  make run          - Runs the sample GRPC server on port 5050"
	@echo "  make run-docs     - Uses docker to render Swagger API docs on port 5051"
	@echo "  make gen-proto    - Generate GRPC definitions from protos"
	@echo "  make tidy         - Updates go.mod following any changes"
	@echo "  make clean        - Removes generated files (except certs)"
