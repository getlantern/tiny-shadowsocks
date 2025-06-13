module github.com/getlantern/tiny-shadowsocks

go 1.23.0

toolchain go1.23.10

replace github.com/sagernet/sing => github.com/getlantern/sing v0.6.12

require (
	github.com/CosmWasm/tinyjson v0.9.0
	github.com/refraction-networking/watm v0.7.0-beta
	github.com/sagernet/sing v0.6.11
	golang.org/x/crypto v0.39.0
)

require (
	github.com/josharian/intern v1.0.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)
