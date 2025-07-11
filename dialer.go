package main

import (
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"

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
	reader      *Reader
	writer      *Writer
}

func (c *ClientConn) writeRequest(payload []byte) error {
	salt := buf.NewSize(c.keySaltLength)
	defer salt.Release()
	salt.WriteRandom(c.keySaltLength)

	key := buf.NewSize(c.keySaltLength)

	Kdf(c.key, salt.Bytes(), key)
	writeCipher, err := c.constructor(key.Bytes())
	key.Release()
	if err != nil {
		fmt.Println("error creating cipher:", err)
		return err
	}
	writer := NewWriter(c.Conn, writeCipher, MaxPacketSize)
	header := writer.Buffer()
	common.Must1(header.Write(salt.Bytes()))
	bufferedWriter := writer.BufferedWriter(header.Len())

	if len(payload) > 0 {
		err = metadata.SocksaddrSerializer.WriteAddrPort(bufferedWriter, c.destination)
		if err != nil {
			fmt.Println("error writing address and port:", err)
			return err
		}

		_, err = bufferedWriter.Write(payload)
		if err != nil {
			fmt.Println("error writing payload:", err)
			return err
		}
	} else {
		err = metadata.SocksaddrSerializer.WriteAddrPort(bufferedWriter, c.destination)
		if err != nil {
			fmt.Println("error writing address and port:", err)
			return err
		}
	}

	err = bufferedWriter.Flush()
	if err != nil {
		fmt.Println("error flushing buffered writer:", err)
		return err
	}

	c.writer = writer
	return nil
}

func (c *ClientConn) readResponse() error {
	salt := buf.NewSize(c.keySaltLength)
	defer salt.Release()
	_, err := salt.ReadFullFrom(c.Conn, c.keySaltLength)
	if err != nil {
		fmt.Println("error reading salt:", err)
		return err
	}
	key := buf.NewSize(c.keySaltLength)
	defer key.Release()
	Kdf(c.key, salt.Bytes(), key)
	readCipher, err := c.constructor(key.Bytes())
	if err != nil {
		fmt.Println("error creating read cipher:", err)
		return err
	}
	c.reader = NewReader(
		c.Conn,
		readCipher,
		MaxPacketSize,
	)
	return nil
}

func (c *ClientConn) Read(p []byte) (n int, err error) {
	if c.reader == nil {
		if err = c.readResponse(); err != nil {
			fmt.Println("error reading response:", err)
			return
		}
	}
	return c.reader.Read(p)
}

func (c *ClientConn) WriteTo(w io.Writer) (n int64, err error) {
	if c.reader == nil {
		if err = c.readResponse(); err != nil {
			fmt.Println("error reading response for WriteTo:", err)
			return
		}
	}
	return c.reader.WriteTo(w)
}

func (c *ClientConn) Write(p []byte) (n int, err error) {
	if c.writer == nil {
		err = c.writeRequest(p)
		if err != nil {
			fmt.Println("error writing request:", err)
			return
		}
		return len(p), nil
	}
	return c.writer.Write(p)
}
