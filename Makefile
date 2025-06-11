# Makefile

CONFIG_FILE := config/config.go

.PHONY: all config debug release test

all: config test debug release 

config:
	@echo "Installing tinyjson..."
	go install github.com/CosmWasm/tinyjson/tinyjson@latest
	tinyjson -all $(CONFIG_FILE)

debug:
	tinygo build -o build/shadowsocks.wasm -target=wasi -scheduler=asyncify -gc=conservative -tags=purego .

release:
	tinygo build -o build/shadowsocks.wasm -target=wasi -no-debug -scheduler=asyncify -gc=conservative -tags=purego .

test:
	go test ./...
