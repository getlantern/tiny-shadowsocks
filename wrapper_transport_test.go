package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/getlantern/tiny-shadowsocks/config"
	"github.com/sagernet/sing/common/metadata"
)

func TestShadowsocksWrappingTransport_Wrap_NoDialer(t *testing.T) {
	tp := &ShadowsocksWrappingTransport{}
	conn := &mockConn{readBuf: &bytes.Buffer{}, writeBuf: &bytes.Buffer{}}

	_, err := tp.Wrap(conn)
	if err == nil || err.Error() != "dialer is not configured" {
		t.Errorf("Expected dialer not configured error, got: %v", err)
	}
}

func TestShadowsocksWrappingTransport_Wrap_Success(t *testing.T) {
	d, err := newDialer("chacha20-ietf-poly1305", "testpass")
	if err != nil {
		t.Fatalf("Failed to create dialer: %v", err)
	}

	tp := &ShadowsocksWrappingTransport{
		dialer:      d,
		destination: metadata.ParseSocksaddr("127.0.0.1:8080"),
	}
	conn := &mockConn{readBuf: &bytes.Buffer{}, writeBuf: &bytes.Buffer{}}

	wrapped, err := tp.Wrap(conn)
	if err != nil {
		t.Errorf("Expected wrap to succeed, got error: %v", err)
	}
	if wrapped == nil {
		t.Error("Expected wrapped connection, got nil")
	}
}

func TestShadowsocksWrappingTransport_Configure_Success(t *testing.T) {
	cfg := config.Config{
		Method:     "chacha20-ietf-poly1305",
		Password:   "abc123",
		RemoteAddr: "example.com",
		RemotePort: "1234",
	}
	jsonCfg := fmt.Appendf([]byte{}, `{"method":"%s","password":"%s","remote_addr":"%s","remote_port":"%s"}`, cfg.Method, cfg.Password, cfg.RemoteAddr, cfg.RemotePort)

	tp := &ShadowsocksWrappingTransport{}
	err := tp.Configure(jsonCfg)
	if err != nil {
		t.Errorf("Expected configuration success, got error: %v", err)
	}
	if tp.dialer == nil {
		t.Error("Expected dialer to be configured")
	}
	if tp.destination.String() != "example.com:1234" {
		t.Errorf("Expected destination 'example.com:1234', got '%s'", tp.destination.String())
	}
}

func TestShadowsocksWrappingTransport_Configure_InvalidJSON(t *testing.T) {
	tp := &ShadowsocksWrappingTransport{}
	err := tp.Configure([]byte(`{invalid json}`))
	if err == nil {
		t.Error("Expected JSON unmarshal error, got nil")
	}
}

func TestShadowsocksWrappingTransport_Configure_InvalidMethod(t *testing.T) {
	jsonCfg := []byte(`{"method":"bad-method","password":"abc","remote_addr":"host","remote_port":"1234"}`)
	tp := &ShadowsocksWrappingTransport{}
	err := tp.Configure(jsonCfg)
	if err == nil {
		t.Error("Expected error due to invalid encryption method")
	}
}
