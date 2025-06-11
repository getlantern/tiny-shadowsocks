package main

import (
	"fmt"

	"github.com/CosmWasm/tinyjson"
	"github.com/getlantern/tiny-shadowsocks/config"
	v1 "github.com/refraction-networking/watm/tinygo/v1"
	v1net "github.com/refraction-networking/watm/tinygo/v1/net"
	"github.com/sagernet/sing/common/metadata"
)

type ShadowsocksWrappingTransport struct {
	destination metadata.Socksaddr
	dialer      *dialer
}

// type guard: ReverseWrappingTransport must implement [v1.WrappingTransport].
var _ v1.WrappingTransport = (*ShadowsocksWrappingTransport)(nil)

func (t *ShadowsocksWrappingTransport) Wrap(conn v1net.Conn) (v1net.Conn, error) {
	if t.dialer == nil {
		return nil, fmt.Errorf("dialer is not configured")
	}

	shadowsocksConn, err := t.dialer.DialConn(conn, t.destination.String())
	if err != nil {
		return nil, fmt.Errorf("failed to dial destination: %w", err)
	}

	return shadowsocksConn, nil
}

var _ v1.ConfigurableTransport = (*ShadowsocksWrappingTransport)(nil)

func (t *ShadowsocksWrappingTransport) Configure(cfg []byte) error {
	var parsedConfig config.Config
	if err := tinyjson.Unmarshal(cfg, &parsedConfig); err != nil {
		return err
	}

	dialer, err := newDialer(parsedConfig.Method, parsedConfig.Password)
	if err != nil {
		return fmt.Errorf("failed to create dialer: %w", err)
	}
	t.dialer = dialer
	t.destination = metadata.ParseSocksaddr(fmt.Sprintf("%s:%s", parsedConfig.RemoteAddr, parsedConfig.RemotePort))
	return nil
}
