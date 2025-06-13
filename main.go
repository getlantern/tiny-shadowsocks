package main

import v1 "github.com/refraction-networking/watm/tinygo/v1"

func init() {
	v1.BuildDialerWithWrappingTransport(&ShadowsocksWrappingTransport{})
	// v1.BuildListenerWithWrappingTransport(&ShadowsocksWrappingTransport{})
	// v1.BuildRelayWithWrappingTransport(&ShadowsocksWrappingTransport{}, v0.RelayWrapRemote)
}

func main() {}
