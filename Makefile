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
	tinygo build -o build/shadowsocks_client.wasm -target=wasi -no-debug -scheduler=none -gc=conservative .

test:
	go test ./...

integration-test: debug
	go test --tags integration -timeout=3s ./...
