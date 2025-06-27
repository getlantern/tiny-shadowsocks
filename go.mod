module github.com/getlantern/tiny-shadowsocks

go 1.22.12

replace github.com/sagernet/sing => github.com/getlantern/sing v0.6.13-0.20250613222345-ef046611f2e9

replace github.com/tetratelabs/wazero => github.com/refraction-networking/wazero v1.7.1-w

require (
	github.com/CosmWasm/tinyjson v0.9.0
	github.com/refraction-networking/water v0.7.1-alpha
	github.com/refraction-networking/watm v0.7.0-beta
	github.com/sagernet/sing v0.6.11
	golang.org/x/crypto v0.24.0
)

require (
	github.com/blang/vfs v1.0.0 // indirect
	github.com/gaukas/wazerofs v0.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/tetratelabs/wazero v1.7.3 // indirect
	golang.org/x/sys v0.21.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
