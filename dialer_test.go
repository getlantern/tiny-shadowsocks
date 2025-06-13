package main

import (
	"bytes"
	"crypto/rand"
	"io"
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
			methods := []string{"aes-128-ctr", "aes-192-ctr", "aes-256-ctr", "rc4-md5", "chacha20-ietf", "xchacha20"}
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
			_, err := newDialer("aes-128-ctr", "")
			if err == nil {
				t.Error("Expected error for empty password, got nil")
			}
		})
	})
}

func TestDialConn_WriteRequest(t *testing.T) {
	d, _ := newDialer("rc4-md5", "testpass")
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

func TestClientConn_ReadWrite(t *testing.T) {
	d, _ := newDialer("rc4-md5", "testpass")
	mock := &mockConn{
		readBuf:  &bytes.Buffer{},
		writeBuf: &bytes.Buffer{},
	}
	cc := &clientConn{
		dialer:      d,
		Conn:        mock,
		destination: "127.0.0.1:8080",
	}

	// Manually set up streams to test Read/Write without calling writeRequest
	salt := make([]byte, d.saltLength)
	io.ReadFull(rand.Reader, salt)
	stream, _ := d.encryptConstructor(d.key, salt)
	cc.writeStream = stream
	cc.readStream, _ = d.decryptConstructor(d.key, salt)

	// Prepare encrypted data
	data := []byte("hello")
	encrypted := make([]byte, len(data))
	copy(encrypted, data)
	stream.XORKeyStream(encrypted, encrypted)
	mock.readBuf.Write(encrypted)

	// Read and verify it decrypts correctly
	buf := make([]byte, 5)
	n, err := cc.Read(buf)
	if err != nil || string(buf[:n]) != "hello" {
		t.Errorf("Expected 'hello', got '%s', err: %v", string(buf[:n]), err)
	}

	// Write and verify encrypted output
	cc.Write([]byte("test"))
	if mock.writeBuf.Len() == 0 {
		t.Error("Expected data written to mock connection")
	}
}

func TestDialer_ReadWrite_TriggersWriteRequestAndReadResponse(t *testing.T) {
	method := "rc4-md5"
	password := "testpass"
	dest := "127.0.0.1:8888"

	// Create a Dialer
	dialer, err := newDialer(method, password)
	if err != nil {
		t.Fatalf("Failed to create dialer: %v", err)
	}

	// Create mock connection buffers
	readBuf := &bytes.Buffer{}
	writeBuf := &bytes.Buffer{}
	conn := &mockConn{
		readBuf:  readBuf,
		writeBuf: writeBuf,
	}

	// Wrap with clientConn via DialEarlyConn (bypasses writeRequest)
	client := dialer.DialEarlyConn(conn, dest).(*clientConn)

	// Manually simulate server-side encrypted response
	// Generate salt and encrypted "pong" message
	salt := make([]byte, dialer.saltLength)
	io.ReadFull(rand.Reader, salt)

	// Encrypt the data using the server-side equivalent decryptConstructor
	respStream, err := dialer.encryptConstructor(dialer.key, salt)
	if err != nil {
		t.Fatalf("Failed to create encryption stream: %v", err)
	}
	plaintext := []byte("pong")
	ciphertext := make([]byte, len(plaintext))
	copy(ciphertext, plaintext)
	respStream.XORKeyStream(ciphertext, ciphertext)

	// Place salt + encrypted data into the read buffer
	readBuf.Write(salt)
	readBuf.Write(ciphertext)

	// --- Test Write (should trigger writeRequest) ---
	msg := []byte("ping")
	n, err := client.Write(msg)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(msg) {
		t.Errorf("Write returned %d, expected %d", n, len(msg))
	}
	if writeBuf.Len() == 0 {
		t.Error("Expected encrypted request to be written")
	}

	// --- Test Read (should trigger readResponse and decrypt) ---
	out := make([]byte, 4)
	n, err = client.Read(out)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(out[:n]) != "pong" {
		t.Errorf("Expected 'pong', got '%s'", string(out[:n]))
	}
}
