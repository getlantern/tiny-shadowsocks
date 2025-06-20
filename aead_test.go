package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"testing"

	"github.com/sagernet/sing/common/buf"
)

func newTestAEAD() cipher.AEAD {
	key := make([]byte, 16)
	rand.Read(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	return aead
}

func TestWriterAndReader(t *testing.T) {
	aead := newTestAEAD()

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "basic",
			plaintext: []byte("hello world, this is a test message"),
		},
		{
			name:      "empty",
			plaintext: []byte{},
		},
		{
			name:      "large",
			plaintext: func() []byte { b := make([]byte, MaxPacketSize*2+100); rand.Read(b); return b }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := NewWriter(&buf, aead, MaxPacketSize)

			n, err := writer.Write(tt.plaintext)
			if err != nil {
				t.Fatalf("writer.Write error: %v", err)
			}
			if n != len(tt.plaintext) {
				t.Fatalf("writer.Write n = %d, want %d", n, len(tt.plaintext))
			}

			data := buf.Bytes()
			reader := NewReader(bytes.NewReader(data), aead, MaxPacketSize)
			readBuf := make([]byte, len(tt.plaintext))
			total := 0
			for total < len(tt.plaintext) {
				n, err := reader.Read(readBuf[total:])
				if err != nil && err != io.EOF {
					t.Fatalf("reader.Read error: %v", err)
				}
				total += n
				if err == io.EOF {
					break
				}
			}
			if !bytes.Equal(readBuf, tt.plaintext) {
				t.Fatalf("decrypted data mismatch: got %q, want %q", readBuf, tt.plaintext)
			}
		})
	}
}

func TestReader_WriteTo(t *testing.T) {
	aead := newTestAEAD()
	var srcBuf bytes.Buffer
	plaintext := []byte("write to test data")
	writer := NewWriter(&srcBuf, aead, MaxPacketSize)
	_, err := writer.Write(plaintext)
	if err != nil {
		t.Fatalf("writer.Write error: %v", err)
	}
	data := srcBuf.Bytes()
	reader := NewReader(bytes.NewReader(data), aead, MaxPacketSize)
	var dstBuf bytes.Buffer
	n, err := reader.WriteTo(&dstBuf)
	if err != nil && err != io.EOF {
		t.Fatalf("WriteTo error: %v", err)
	}
	if n != int64(len(plaintext)) {
		t.Fatalf("WriteTo n = %d, want %d", n, len(plaintext))
	}
	if !bytes.Equal(dstBuf.Bytes(), plaintext) {
		t.Fatalf("WriteTo data mismatch")
	}
}

func TestReader_ReadByte(t *testing.T) {
	aead := newTestAEAD()
	var srcBuf bytes.Buffer
	plaintext := []byte("byte test")
	writer := NewWriter(&srcBuf, aead, MaxPacketSize)
	_, err := writer.Write(plaintext)
	if err != nil {
		t.Fatalf("writer.Write error: %v", err)
	}
	data := srcBuf.Bytes()
	reader := NewReader(bytes.NewReader(data), aead, MaxPacketSize)
	for i := 0; i < len(plaintext); i++ {
		b, err := reader.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte error: %v", err)
		}
		if b != plaintext[i] {
			t.Fatalf("ReadByte got %v, want %v", b, plaintext[i])
		}
	}
}

func TestReader_Discard(t *testing.T) {
	aead := newTestAEAD()
	var srcBuf bytes.Buffer
	plaintext := []byte("discard test data")
	writer := NewWriter(&srcBuf, aead, MaxPacketSize)
	_, err := writer.Write(plaintext)
	if err != nil {
		t.Fatalf("writer.Write error: %v", err)
	}
	data := srcBuf.Bytes()
	reader := NewReader(bytes.NewReader(data), aead, MaxPacketSize)
	err = reader.Discard(8)
	if err != nil {
		t.Fatalf("Discard error: %v", err)
	}
	buf := make([]byte, len(plaintext)-8)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read after Discard error: %v", err)
	}
	if !bytes.Equal(buf[:n], plaintext[8:]) {
		t.Fatalf("Discard did not skip bytes correctly")
	}
}

func TestReader_Buffer_Cached_CachedSlice(t *testing.T) {
	aead := newTestAEAD()
	var srcBuf bytes.Buffer
	plaintext := []byte("buffer test data")
	writer := NewWriter(&srcBuf, aead, MaxPacketSize)
	_, err := writer.Write(plaintext)
	if err != nil {
		t.Fatalf("writer.Write error: %v", err)
	}
	data := srcBuf.Bytes()
	reader := NewReader(bytes.NewReader(data), aead, MaxPacketSize)
	// Read some bytes to fill cache
	buf := make([]byte, 5)
	_, err = reader.Read(buf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	b := reader.Buffer()
	if b == nil {
		t.Fatalf("Buffer() returned nil")
	}
	if reader.Cached() != len(plaintext)-5 {
		t.Fatalf("Cached() = %d, want %d", reader.Cached(), len(plaintext)-5)
	}
	cachedSlice := reader.CachedSlice()
	if !bytes.Equal(cachedSlice, plaintext[5:]) {
		t.Fatalf("CachedSlice mismatch")
	}
}

func TestReader_ReadWithLengthChunk(t *testing.T) {
	aead := newTestAEAD()
	var srcBuf bytes.Buffer
	plaintext := []byte("length chunk test")
	writer := NewWriter(&srcBuf, aead, MaxPacketSize)
	_, err := writer.Write(plaintext)
	if err != nil {
		t.Fatalf("writer.Write error: %v", err)
	}
	data := srcBuf.Bytes()
	// Simulate reading the length chunk
	lengthChunk := data[:PacketLengthBufferSize+Overhead]
	reader := NewReader(bytes.NewReader(data[PacketLengthBufferSize+Overhead:]), aead, MaxPacketSize)
	err = reader.ReadWithLengthChunk(lengthChunk)
	if err != nil {
		t.Fatalf("ReadWithLengthChunk error: %v", err)
	}
	buf := make([]byte, len(plaintext))
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}
	if !bytes.Equal(buf[:n], plaintext) {
		t.Fatalf("ReadWithLengthChunk data mismatch")
	}
}

func TestReader_ReadWithLength(t *testing.T) {
	aead := newTestAEAD()
	var srcBuf bytes.Buffer
	plaintext := []byte("length test")
	writer := NewWriter(&srcBuf, aead, MaxPacketSize)
	_, err := writer.Write(plaintext)
	if err != nil {
		t.Fatalf("writer.Write error: %v", err)
	}
	data := srcBuf.Bytes()
	offset := PacketLengthBufferSize + Overhead
	reader := NewReader(bytes.NewReader(data[offset:]), aead, MaxPacketSize)
	// Increment nonce to match the state after reading the length chunk
	increaseNonce(reader.nonce)
	err = reader.ReadWithLength(uint16(len(plaintext)))
	if err != nil {
		t.Fatalf("ReadWithLength error: %v", err)
	}
	buf := make([]byte, len(plaintext))
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read error: %v", err)
	}
	if !bytes.Equal(buf[:n], plaintext) {
		t.Fatalf("ReadWithLength data mismatch")
	}
}

func TestNewRawWriter(t *testing.T) {
	aead := newTestAEAD()
	var buf bytes.Buffer
	buffer := make([]byte, MaxPacketSize+PacketLengthBufferSize+Overhead*2)
	nonce := make([]byte, aead.NonceSize())
	writer := NewRawWriter(&buf, aead, MaxPacketSize, buffer, nonce)
	if writer == nil {
		t.Fatalf("NewRawWriter returned nil")
	}
	if !bytes.Equal(writer.buffer, buffer) {
		t.Fatalf("NewRawWriter buffer mismatch")
	}
	if !bytes.Equal(writer.nonce, nonce) {
		t.Fatalf("NewRawWriter nonce mismatch")
	}
}

func TestReader_ReadExternalChunk(t *testing.T) {
	aead := newTestAEAD()
	nonce := make([]byte, aead.NonceSize())
	plaintext := []byte("external chunk test")
	chunk := aead.Seal(nil, nonce, plaintext, nil)
	reader := NewReader(nil, aead, MaxPacketSize)
	copy(reader.nonce, nonce)
	err := reader.ReadExternalChunk(chunk)
	if err != nil {
		t.Fatalf("ReadExternalChunk error: %v", err)
	}
	if reader.cached != len(plaintext) {
		t.Fatalf("ReadExternalChunk cached = %d, want %d", reader.cached, len(plaintext))
	}
	got := reader.CachedSlice()
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("ReadExternalChunk got %q, want %q", got, plaintext)
	}
}

func TestReader_ReadChunk(t *testing.T) {
	aead := newTestAEAD()
	nonce := make([]byte, aead.NonceSize())
	plaintext := []byte("read chunk test")
	chunk := aead.Seal(nil, nonce, plaintext, nil)
	reader := NewReader(nil, aead, MaxPacketSize)
	copy(reader.nonce, nonce)
	buffer := buf.New()
	err := reader.ReadChunk(buffer, chunk)
	if err != nil {
		t.Fatalf("ReadChunk error: %v", err)
	}
	if !bytes.Equal(buffer.Bytes(), plaintext) {
		t.Fatalf("ReadChunk got %q, want %q", buffer.Bytes(), plaintext)
	}
}

func TestWriter_WriteChunk(t *testing.T) {
	aead := newTestAEAD()
	writer := NewWriter(io.Discard, aead, MaxPacketSize)
	buffer := buf.New()
	chunk := []byte("chunk data")
	writer.WriteChunk(buffer, chunk)
	if buffer.Len() == 0 {
		t.Fatalf("WriteChunk did not write data")
	}
}
