//go:build integration
// +build integration

package main

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/refraction-networking/water"
	_ "github.com/refraction-networking/water/transport/v1"
)

func TestWASM(t *testing.T) {
	// verify if build/shadowsocks.wasm exists
	filename := "build/shadowsocks_client_debug.wasm"
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Skipf("Skipping test: %q does not exist. Please run 'make debug' or 'make release' to generate the WASM file.", filename)
	}

	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("failed to open %q: %v", filename, err)
	}

	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read %q: %v", filename, err)
	}

	config := `
	{
		"remote_addr": "34.160.111.145",
		"remote_port": "80",
		"password": "8JCsPssfgS8tiRwiMlhARg==",
		"method": "chacha20-ietf-poly1305"
	}
	`
	cfg := &water.Config{
		TransportModuleBin:    b,
		TransportModuleConfig: water.TransportModuleConfigFromBytes([]byte(config)),
		NetworkDialerFunc: func(network, address string) (net.Conn, error) {
			return net.Dial(network, "127.0.0.1:8388")
		},
	}

	dialer, err := water.NewDialerWithContext(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create dialer: %v", err)
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	// Send a simple HTTP GET request
	request, err := http.NewRequest("GET", "http://ifconfig.me/ip", nil)
	if err != nil {
		t.Fatalf("failed to create HTTP request: %v", err)
	}
	request.Header.Set("Host", "ifconfig.me")
	request.Header.Set("Accept", "text/plain")

	if err = request.Write(conn); err != nil {
		t.Fatalf("failed to write HTTP request: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), request)
	if err != nil {
		t.Fatalf("failed to read HTTP response: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	t.Logf("response: %s", body)
}

func TestDialFixed(t *testing.T) {
	// verify if build/shadowsocks.wasm exists
	filename := "build/shadowsocks_client_debug.wasm"
	// filename := "/home/wendelhime/github.com/watm/tinygo/v1/examples/plain/plain.wasm"
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Skipf("Skipping test: %q does not exist. Please run 'make debug' or 'make release' to generate the WASM file.", filename)
	}

	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("failed to open %q: %v", filename, err)
	}

	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read %q: %v", filename, err)
	}

	config := `
	{
		"remote_addr": "34.160.111.145",
		"remote_port": "80",
		"password": "8JCsPssfgS8tiRwiMlhARg==",
		"method": "chacha20-ietf-poly1305"
	}
	`
	cfg := &water.Config{
		TransportModuleBin:    b,
		TransportModuleConfig: water.TransportModuleConfigFromBytes([]byte(config)),
		NetworkDialerFunc: func(network, address string) (net.Conn, error) {
			return net.Dial(network, "127.0.0.1:8388")
		},
		DialedAddressValidator: func(network, address string) error {
			if network != "tcp" {
				return net.InvalidAddrError("network must be tcp")
			}
			if address == "" {
				return net.InvalidAddrError("address cannot be empty")
			}
			return nil
		},
	}

	dialer, err := water.NewFixedDialerWithContext(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create dialer: %v", err)
	}

	conn, err := dialer.DialFixedContext(context.Background())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	// Send a simple HTTP GET request
	request, err := http.NewRequest("GET", "http://ifconfig.me/ip", nil)
	if err != nil {
		t.Fatalf("failed to create HTTP request: %v", err)
	}
	request.Header.Set("Host", "ifconfig.me")
	request.Header.Set("Accept", "text/plain")

	if err = request.Write(conn); err != nil {
		t.Fatalf("failed to write HTTP request: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), request)
	if err != nil {
		t.Fatalf("failed to read HTTP response: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	t.Logf("response: %s", body)
}
