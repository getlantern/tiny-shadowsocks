package shadowio

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"io"
	"syscall"

	"github.com/sagernet/sing/common/buf"
	N "github.com/sagernet/sing/common/network"
)

const PacketLengthBufferSize = 2

const (
	// Overhead
	// crypto/cipher.gcmTagSize
	// golang.org/x/crypto/chacha20poly1305.Overhead
	Overhead = 16
)

var (
	_ N.ExtendedReader = (*Reader)(nil)
	_ N.ReadWaiter     = (*Reader)(nil)
)

type Reader struct {
	reader          io.Reader
	cipher          cipher.AEAD
	nonce           []byte
	cache           *buf.Buffer
	readWaitOptions N.ReadWaitOptions

	// State for resuming partial AEAD frame reads across EAGAIN boundaries.
	// When reading from a non-blocking socket, io.ReadFull may return EAGAIN
	// after a partial read if an encrypted frame spans multiple TCP segments.
	// These fields preserve the in-progress buffer so the next call resumes.
	pending      *buf.Buffer
	pendingPhase int // 0 = reading length header, 1 = reading content
	contentLen   int
}

func NewReader(upstream io.Reader, cipher cipher.AEAD) *Reader {
	return &Reader{
		reader: upstream,
		cipher: cipher,
		nonce:  make([]byte, cipher.NonceSize()),
	}
}

// Close releases any pooled buffers still held by the reader. It is safe to
// call multiple times. It does not close the underlying reader.
func (r *Reader) Close() error {
	if r.cache != nil {
		r.cache.Release()
		r.cache = nil
	}
	if r.pending != nil {
		r.pending.Release()
		r.pending = nil
	}
	// Reset frame state so a reused Reader restarts at the length header
	// instead of dereferencing a nil r.pending in phase 1.
	r.pendingPhase = 0
	r.contentLen = 0
	return nil
}

func (r *Reader) ReadFixedBuffer(pLen int) (*buf.Buffer, error) {
	buffer := buf.NewSize(pLen + Overhead)
	_, err := buffer.ReadFullFrom(r.reader, buffer.FreeLen())
	if err != nil {
		buffer.Release()
		return nil, err
	}
	err = r.Decrypt(buffer.Index(0), buffer.Bytes())
	if err != nil {
		buffer.Release()
		return nil, err
	}
	buffer.Truncate(pLen)
	r.cache = buffer
	return buffer, nil
}

func (r *Reader) Decrypt(destination []byte, source []byte) error {
	_, err := r.cipher.Open(destination[:0], r.nonce, source, nil)
	if err != nil {
		return err
	}
	increaseNonce(r.nonce)
	return nil
}

func (r *Reader) Read(p []byte) (n int, err error) {
	for {
		if r.cache != nil {
			if r.cache.IsEmpty() {
				r.cache.Release()
				r.cache = nil
			} else {
				n = copy(p, r.cache.Bytes())
				if n > 0 {
					r.cache.Advance(n)
					return
				}
			}
		}
		r.cache, err = r.readBuffer()
		if err != nil {
			return
		}
	}
}

func (r *Reader) ReadBuffer(buffer *buf.Buffer) error {
	var err error
	for {
		if r.cache != nil {
			if r.cache.IsEmpty() {
				r.cache.Release()
				r.cache = nil
			} else {
				n := copy(buffer.FreeBytes(), r.cache.Bytes())
				if n > 0 {
					buffer.Truncate(n)
					r.cache.Advance(n)
					return nil
				}
			}
		}
		r.cache, err = r.readBuffer()
		if err != nil {
			return err
		}
	}
}

func (r *Reader) InitializeReadWaiter(options N.ReadWaitOptions) (needCopy bool) {
	r.readWaitOptions = options
	return options.NeedHeadroom()
}

func (r *Reader) WaitReadBuffer() (buffer *buf.Buffer, err error) {
	if r.readWaitOptions.NeedHeadroom() {
		for {
			if r.cache != nil {
				if r.cache.IsEmpty() {
					r.cache.Release()
					r.cache = nil
				} else {
					buffer = r.readWaitOptions.NewBuffer()
					var n int
					n, err = buffer.Write(r.cache.Bytes())
					if err != nil {
						buffer.Release()
						return
					}
					buffer.Truncate(n)
					r.cache.Advance(n)
					r.readWaitOptions.PostReturn(buffer)
					return
				}
			}
			r.cache, err = r.readBuffer()
			if err != nil {
				return
			}
		}
	} else {
		cache := r.cache
		if cache != nil {
			r.cache = nil
			return cache, nil
		}
		return r.readBuffer()
	}
}

func (r *Reader) readBuffer() (*buf.Buffer, error) {
	// Phase 0: Read and decrypt the 2-byte length header + AEAD overhead.
	if r.pendingPhase == 0 {
		if r.pending == nil {
			r.pending = buf.NewSize(PacketLengthBufferSize + Overhead)
		}
		_, err := r.pending.ReadFullFrom(r.reader, r.pending.FreeLen())
		if err != nil {
			if errors.Is(err, syscall.EAGAIN) {
				// Partial read — preserve buffer state for next call.
				return nil, err
			}
			r.pending.Release()
			r.pending = nil
			return nil, err
		}
		_, err = r.cipher.Open(r.pending.Index(0), r.nonce, r.pending.Bytes(), nil)
		if err != nil {
			r.pending.Release()
			r.pending = nil
			return nil, err
		}
		r.contentLen = int(binary.BigEndian.Uint16(r.pending.To(PacketLengthBufferSize)))
		r.pending.Release()
		r.pending = buf.NewSize(r.contentLen + Overhead)
		r.pendingPhase = 1
	}

	// Phase 1: Read and decrypt the content payload.
	_, err := r.pending.ReadFullFrom(r.reader, r.pending.FreeLen())
	if err != nil {
		if errors.Is(err, syscall.EAGAIN) {
			// Partial read — preserve buffer state for next call.
			return nil, err
		}
		r.pending.Release()
		r.pending = nil
		r.pendingPhase = 0
		return nil, err
	}

	increaseNonce(r.nonce)
	buffer := r.pending
	r.pending = nil
	r.pendingPhase = 0

	_, err = r.cipher.Open(buffer.Index(0), r.nonce, buffer.Bytes(), nil)
	if err != nil {
		buffer.Release()
		return nil, err
	}
	increaseNonce(r.nonce)
	buffer.Truncate(r.contentLen)
	return buffer, nil
}
