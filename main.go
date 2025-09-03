package main

import (
	v1 "github.com/refraction-networking/watm/tinygo/v1"
)

func init() {
	v1.WorkerFairness(false)
	v1.SetReadBufferSize(MaxPacketSize)
	v1.BuildDialerWithWrappingTransport(&ShadowsocksWrappingTransport{})
	// v1.BuildListenerWithWrappingTransport(&ShadowsocksWrappingTransport{})
	// v1.BuildRelayWithWrappingTransport(&ShadowsocksWrappingTransport{}, v1.RelayWrapRemote)
	v1.BuildFixedDialerWithFixedDialingTransport(&ShadowsocksFixedDialingTransport{})
}

func main() {}
