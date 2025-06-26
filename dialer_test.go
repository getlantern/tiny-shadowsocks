package main

import (
	"bytes"
	"net"
	"syscall"
	"testing"
	"time"

	v1net "github.com/refraction-networking/watm/tinygo/v1/net"
)

type mockConn struct {
	v1net.Conn
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.readBuf.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writeBuf.Write(b)
}

func (m *mockConn) Fd() int32                             { return 0 }
func (m *mockConn) Close() error                          { return nil }
func (m *mockConn) LocalAddr() net.Addr                   { return nil }
func (m *mockConn) RemoteAddr() net.Addr                  { return nil }
func (m *mockConn) SetDeadline(t time.Time) error         { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error     { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error    { return nil }
func (m *mockConn) SyscallConn() (syscall.RawConn, error) { return nil, nil }
func (m *mockConn) SetNonBlock(nonblocking bool) error    { return nil }

func TestNewDialer(t *testing.T) {
	t.Run("creating dialer with", func(t *testing.T) {
		t.Run("valid method should return dialer", func(t *testing.T) {
			methods := []string{"chacha20-ietf-poly1305", "xchacha20-ietf-poly1305"}
			for _, method := range methods {
				d, err := newDialer(method, "password123")
				if err != nil {
					t.Errorf("Expected no error for method %s, got %v", method, err)
				}
				if d == nil {
					t.Errorf("Expected dialer for method %s, got nil", method)
				}
			}
		})

		t.Run("invalid method should return error", func(t *testing.T) {
			_, err := newDialer("invalid-method", "password123")
			if err == nil {
				t.Error("Expected error for invalid method, got nil")
			}
		})

		t.Run("empty password should return error", func(t *testing.T) {
			_, err := newDialer("chacha20-ietf-poly1305", "")
			if err == nil {
				t.Error("Expected error for empty password, got nil")
			}
		})
	})
}

func TestDialConn_WriteRequest(t *testing.T) {
	d, _ := newDialer("chacha20-ietf-poly1305", "testpass")
	readBuf := &bytes.Buffer{}
	writeBuf := &bytes.Buffer{}
	conn := &mockConn{readBuf: readBuf, writeBuf: writeBuf}

	dialedConn, err := d.DialConn(conn, "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("DialConn failed: %v", err)
	}

	if dialedConn == nil {
		t.Fatal("Expected non-nil connection")
	}
	if writeBuf.Len() == 0 {
		t.Error("Expected data to be written to mock connection")
	}
}
