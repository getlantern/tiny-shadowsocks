package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"errors"
	"io"
	"net"
	"os"
	"strconv"

	v1net "github.com/refraction-networking/watm/tinygo/v1/net"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/metadata"
	"golang.org/x/crypto/chacha20"
)

type dialer struct {
	method             string
	keyLength          int
	saltLength         int
	encryptConstructor func(key []byte, salt []byte) (cipher.Stream, error)
	decryptConstructor func(key []byte, salt []byte) (cipher.Stream, error)
	key                []byte
}

func blockStream(blockCreator func(key []byte) (cipher.Block, error), streamCreator func(block cipher.Block, iv []byte) cipher.Stream) func([]byte, []byte) (cipher.Stream, error) {
	return func(key []byte, iv []byte) (cipher.Stream, error) {
		block, err := blockCreator(key)
		if err != nil {
			return nil, err
		}
		return streamCreator(block, iv), err
	}
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

func newDialer(method, password string) (*dialer, error) {
	dialer := &dialer{}
	switch method {
	case "aes-128-ctr":
		dialer.keyLength = 16
		dialer.saltLength = aes.BlockSize
		dialer.encryptConstructor = blockStream(aes.NewCipher, cipher.NewCTR)
		dialer.decryptConstructor = blockStream(aes.NewCipher, cipher.NewCTR)
	case "aes-192-ctr":
		dialer.keyLength = 24
		dialer.saltLength = aes.BlockSize
		dialer.encryptConstructor = blockStream(aes.NewCipher, cipher.NewCTR)
		dialer.decryptConstructor = blockStream(aes.NewCipher, cipher.NewCTR)
	case "aes-256-ctr":
		dialer.keyLength = 32
		dialer.saltLength = aes.BlockSize
		dialer.encryptConstructor = blockStream(aes.NewCipher, cipher.NewCTR)
		dialer.decryptConstructor = blockStream(aes.NewCipher, cipher.NewCTR)
	case "rc4-md5":
		dialer.keyLength = 16
		dialer.saltLength = 16
		dialer.encryptConstructor = func(key []byte, salt []byte) (cipher.Stream, error) {
			h := md5.New()
			h.Write(key)
			h.Write(salt)
			return rc4.NewCipher(h.Sum(nil))
		}
		dialer.decryptConstructor = func(key []byte, salt []byte) (cipher.Stream, error) {
			h := md5.New()
			h.Write(key)
			h.Write(salt)
			return rc4.NewCipher(h.Sum(nil))
		}
	case "chacha20-ietf":
		dialer.keyLength = chacha20.KeySize
		dialer.saltLength = chacha20.NonceSize
		dialer.encryptConstructor = func(key []byte, salt []byte) (cipher.Stream, error) {
			return chacha20.NewUnauthenticatedCipher(key, salt)
		}
		dialer.decryptConstructor = func(key []byte, salt []byte) (cipher.Stream, error) {
			return chacha20.NewUnauthenticatedCipher(key, salt)
		}
	case "xchacha20":
		dialer.keyLength = chacha20.KeySize
		dialer.saltLength = chacha20.NonceSizeX
		dialer.encryptConstructor = func(key []byte, salt []byte) (cipher.Stream, error) {
			return chacha20.NewUnauthenticatedCipher(key, salt)
		}
		dialer.decryptConstructor = func(key []byte, salt []byte) (cipher.Stream, error) {
			return chacha20.NewUnauthenticatedCipher(key, salt)
		}
	default:
		return nil, os.ErrInvalid
	}

	if password == "" {
		return nil, errors.New("password is required")
	}

	dialer.key = key([]byte(password), dialer.keyLength)
	return dialer, nil
}

func (d *dialer) DialConn(conn v1net.Conn, destination string) (v1net.Conn, error) {
	shadowsocksConn := &clientConn{
		dialer:      d,
		Conn:        conn,
		destination: destination,
	}
	return shadowsocksConn, shadowsocksConn.writeRequest()
}

func (d *dialer) DialEarlyConn(conn v1net.Conn, destination string) v1net.Conn {
	return &clientConn{
		dialer:      d,
		Conn:        conn,
		destination: destination,
	}
}

type clientConn struct {
	*dialer
	v1net.Conn  // embedded Conn
	destination string
	readStream  cipher.Stream
	writeStream cipher.Stream
}

func parseSocksaddrFromString(destination string) (metadata.Socksaddr, error) {
	var addr metadata.Socksaddr
	host, port, err := net.SplitHostPort(destination)
	if err != nil {
		return addr, err
	}

	ip := net.ParseIP(host)
	p, err := strconv.Atoi(port)
	if err != nil {
		return addr, err
	}
	tcpAddr := &net.TCPAddr{
		IP:   ip,
		Port: p,
	}
	return metadata.SocksaddrFromNet(tcpAddr), nil
}

func (c *clientConn) writeRequest() error {
	addr, err := parseSocksaddrFromString(c.destination)
	if err != nil {
		return err
	}

	buffer := buf.NewSize(c.saltLength + metadata.SocksaddrSerializer.AddrPortLen(addr))
	defer buffer.Release()

	salt := buffer.Extend(c.saltLength)
	_, err = io.ReadFull(rand.Reader, salt)
	if err != nil {
		return err
	}

	stream, err := c.encryptConstructor(c.key, salt)
	if err != nil {
		return err
	}

	err = metadata.SocksaddrSerializer.WriteAddrPort(buffer, addr)
	if err != nil {
		return err
	}

	stream.XORKeyStream(buffer.From(c.saltLength), buffer.From(c.saltLength))

	_, err = c.Conn.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	c.writeStream = stream
	return nil
}

func (c *clientConn) readResponse() error {
	if c.readStream != nil {
		return nil
	}

	salt := make([]byte, c.saltLength)
	_, err := io.ReadFull(c.Conn, salt)
	if err != nil {
		return err
	}
	c.readStream, err = c.decryptConstructor(c.key, salt)
	return err
}

func (c *clientConn) Read(p []byte) (n int, err error) {
	if c.readStream == nil {
		err = c.readResponse()
		if err != nil {
			return
		}
	}
	n, err = c.Conn.Read(p)
	if err != nil {
		return
	}
	c.readStream.XORKeyStream(p[:n], p[:n])
	return
}

func (c *clientConn) Write(p []byte) (n int, err error) {
	if c.writeStream == nil {
		err = c.writeRequest()
		if err != nil {
			return
		}
	}

	c.writeStream.XORKeyStream(p, p)
	return c.Conn.Write(p)
}
