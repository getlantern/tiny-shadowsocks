package main

import (
	"fmt"

	"github.com/CosmWasm/tinyjson"
	"github.com/getlantern/tiny-shadowsocks/config"
	v1 "github.com/refraction-networking/watm/tinygo/v1"
	v1net "github.com/refraction-networking/watm/tinygo/v1/net"
	"github.com/sagernet/sing/common/metadata"
)

type ShadowsocksFixedDialingTransport struct {
	dialer            func(network, address string) (v1net.Conn, error)
	shadowsocksDialer *Dialer
	destination       metadata.Socksaddr
}

var _ v1.FixedDialingTransport = (*ShadowsocksFixedDialingTransport)(nil)
var _ v1.ConfigurableTransport = (*ShadowsocksFixedDialingTransport)(nil)

func (fdt *ShadowsocksFixedDialingTransport) SetDialer(dialer func(network, address string) (v1net.Conn, error)) {
	fdt.dialer = dialer
}

func (fdt *ShadowsocksFixedDialingTransport) DialFixed() (v1net.Conn, error) {
	conn, err := fdt.dialer("tcp", "127.0.0.1:7777") // TODO: hardcoded address, any better idea?
	if err != nil {
		fmt.Println("failed to dial with dialer: ", err.Error())
		return nil, err
	}

	return fdt.shadowsocksDialer.DialEarlyConn(conn, fdt.destination), conn.SetNonBlock(true) // must set non-block, otherwise will block on read and lose fairness
}

func (fdt *ShadowsocksFixedDialingTransport) Configure(cfg []byte) error {
	var parsedConfig config.Config
	if err := tinyjson.Unmarshal(cfg, &parsedConfig); err != nil {
		fmt.Printf("failed to unmarshal config: %+v\n", err)
		return err
	}

	dialer, err := newDialer(parsedConfig.Method, parsedConfig.Password)
	if err != nil {
		fmt.Printf("failed to create dialer: %+v\n", err)
		return fmt.Errorf("failed to create dialer: %w", err)
	}
	fdt.shadowsocksDialer = dialer
	fdt.destination = metadata.ParseSocksaddrHostPortStr(parsedConfig.RemoteAddr, parsedConfig.RemotePort)
	return nil
}
