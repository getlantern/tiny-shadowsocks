# Makefile

CONFIG_FILE := config/config.go

.PHONY: all config debug release test

all: config test debug release 

config:
	@echo "Installing tinyjson..."
	go install github.com/CosmWasm/tinyjson/tinyjson@latest
	tinyjson -all $(CONFIG_FILE)

debug:
	tinygo build -o build/shadowsocks_client_debug.wasm -target=wasi -scheduler=none -gc=conservative .

release:
	@command -v wasm-opt >/dev/null 2>&1 || { echo "Error: wasm-opt (Binaryen) is required for the release build but was not found in PATH. Install Binaryen: https://github.com/WebAssembly/binaryen"; exit 1; }
	tinygo build -o build/shadowsocks_client.wasm \
	  -target=wasi \
	  -no-debug \
	  -opt=z \
	  -panic=trap \
	  -scheduler=none \
	  -gc=conservative \
	  .
	wasm-opt -Oz --strip-debug --strip-producers --strip-dwarf --vacuum --converge \
	  build/shadowsocks_client.wasm -o build/shadowsocks_client.wasm

test:
	go test ./...

integration-test: debug
	go test --tags integration -timeout=3s ./...
