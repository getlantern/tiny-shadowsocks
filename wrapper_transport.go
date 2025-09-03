package main

import (
	"fmt"
	"log/slog"

	"github.com/CosmWasm/tinyjson"
	"github.com/getlantern/tiny-shadowsocks/config"
	v1 "github.com/refraction-networking/watm/tinygo/v1"
	v1net "github.com/refraction-networking/watm/tinygo/v1/net"
	"github.com/sagernet/sing/common/metadata"
)

type ShadowsocksWrappingTransport struct {
	destination metadata.Socksaddr
	dialer      *Dialer
}

var _ v1.WrappingTransport = (*ShadowsocksWrappingTransport)(nil)
var _ v1.ConfigurableTransport = (*ShadowsocksWrappingTransport)(nil)

func (t *ShadowsocksWrappingTransport) Wrap(conn v1net.Conn) (v1net.Conn, error) {
	if t.dialer == nil {
		return nil, fmt.Errorf("dialer is not configured")
	}
	return t.dialer.DialEarlyConn(conn, t.destination), conn.SetNonBlock(true)
}

func (t *ShadowsocksWrappingTransport) Configure(cfg []byte) error {
	var parsedConfig config.Config
	if err := tinyjson.Unmarshal(cfg, &parsedConfig); err != nil {
		slog.Error("failed to unmarshal config", slog.Any("error", err))
		return err
	}

	if parsedConfig.InternalBufferSize != 0 {
		v1.SetReadBufferSize(parsedConfig.InternalBufferSize)
	}

	dialer, err := newDialer(parsedConfig.Method, parsedConfig.Password)
	if err != nil {
		return fmt.Errorf("failed to create dialer: %w", err)
	}
	t.dialer = dialer
	t.destination = metadata.ParseSocksaddrHostPortStr(parsedConfig.RemoteAddr, parsedConfig.RemotePort)
	return nil
}
