package main

import (
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha1"
	"errors"
	"fmt"
	"os"

	"github.com/getlantern/tiny-shadowsocks/bufio"
	"github.com/getlantern/tiny-shadowsocks/internal/shadowio"
	v1net "github.com/refraction-networking/watm/tinygo/v1/net"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/metadata"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

type Dialer struct {
	method        string
	keySaltLength int
	constructor   func(key []byte) (cipher.AEAD, error)
	key           []byte
}

func key(password []byte, keySize int) []byte {
	var b, prev []byte
	h := md5.New()
	for len(b) < keySize {
		h.Write(prev)
		h.Write(password)
		b = h.Sum(b)
		prev = b[len(b)-h.Size():]
		h.Reset()
	}
	return b[:keySize]
}

func newDialer(method, password string) (*Dialer, error) {
	dialer := &Dialer{method: method}
	switch method {
	case "chacha20-ietf-poly1305":
		dialer.keySaltLength = 32
		dialer.constructor = chacha20poly1305.New
	case "xchacha20-ietf-poly1305":
		dialer.keySaltLength = 32
		dialer.constructor = chacha20poly1305.NewX
	default:
		return nil, os.ErrInvalid
	}

	if password == "" {
		return nil, errors.New("password is required")
	}

	dialer.key = key([]byte(password), dialer.keySaltLength)
	return dialer, nil
}

func Kdf(key, iv []byte, buffer *buf.Buffer) {
	kdf := hkdf.New(sha1.New, key, iv, []byte("ss-subkey"))
	common.Must1(buffer.ReadFullFrom(kdf, buffer.FreeLen()))
}

func (d *Dialer) DialConn(conn v1net.Conn, destination metadata.Socksaddr) (v1net.Conn, error) {
	shadowsocksConn := &ClientConn{
		Dialer:      d,
		Conn:        conn,
		destination: destination,
	}
	return shadowsocksConn, shadowsocksConn.writeRequest(nil)
}

func (d *Dialer) DialEarlyConn(conn v1net.Conn, destination metadata.Socksaddr) v1net.Conn {
	return &ClientConn{
		Dialer:      d,
		Conn:        conn,
		destination: destination,
	}
}

type ClientConn struct {
	*Dialer
	v1net.Conn  // embedded Conn
	destination metadata.Socksaddr
	reader      *shadowio.Reader
	writer      *shadowio.Writer
}

func (c *ClientConn) writeRequest(payload []byte) error {
	requestBuffer := buf.New()
	requestBuffer.WriteRandom(c.keySaltLength)
	key := buf.NewSize(c.keySaltLength)
	Kdf(c.key, requestBuffer.Bytes(), key)
	writeCipher, err := c.constructor(key.Bytes())
	if err != nil {
		return err
	}
	bufferedRequestWriter := bufio.NewBufferedWriter(c.Conn, requestBuffer)
	requestContentWriter := shadowio.NewWriter(bufferedRequestWriter, writeCipher, nil, MaxPacketSize)
	bufferedRequestContentWriter := bufio.NewBufferedWriter(requestContentWriter, buf.New())
	if err = metadata.SocksaddrSerializer.WriteAddrPort(bufferedRequestContentWriter, c.destination); err != nil {
		return err
	}

	if _, err = bufferedRequestContentWriter.Write(payload); err != nil {
		return err
	}

	if err = bufferedRequestContentWriter.Fallthrough(); err != nil {
		return err
	}

	if err = bufferedRequestWriter.Fallthrough(); err != nil {
		return err
	}
	c.writer = shadowio.NewWriter(c.Conn, writeCipher, requestContentWriter.TakeNonce(), MaxPacketSize)
	return nil
}

func (c *ClientConn) readResponse() error {
	buffer := buf.NewSize(c.keySaltLength)
	defer buffer.Release()

	if _, err := buffer.ReadFullFrom(c.Conn, c.keySaltLength); err != nil {
		fmt.Println("failed to read salt", err)
		return err
	}
	key := buf.NewSize(c.keySaltLength)
	defer key.Release()
	Kdf(c.key, buffer.Bytes(), key)
	readCipher, err := c.constructor(key.Bytes())
	if err != nil {
		fmt.Println("failed to build cipher decrypt: ", err)
		return err
	}
	reader := shadowio.NewReader(c.Conn, readCipher)
	c.reader = reader
	return nil
}

func (c *ClientConn) Read(p []byte) (n int, err error) {
	if c.reader == nil {
		err = c.readResponse()
		if err != nil {
			return
		}
	}
	return c.reader.Read(p)
}

func (c *ClientConn) Write(p []byte) (n int, err error) {
	if c.writer == nil {
		err = c.writeRequest(p)
		if err == nil {
			n = len(p)
		}
		return
	}
	return c.writer.Write(p)
}
