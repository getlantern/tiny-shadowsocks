// Package bufio implement some functions from github.com/sagernet/sing/common/bufio package
package bufio

import (
	"io"

	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/buf"
	N "github.com/sagernet/sing/common/network"
)

type ExtendedWriterWrapper struct {
	io.Writer
}

func (w *ExtendedWriterWrapper) WriteBuffer(buffer *buf.Buffer) error {
	defer buffer.Release()
	return common.Error(w.Write(buffer.Bytes()))
}

func (w *ExtendedWriterWrapper) Upstream() any {
	return w.Writer
}

func (w *ExtendedWriterWrapper) WriterReplaceable() bool {
	return true
}

func NewExtendedWriter(writer io.Writer) N.ExtendedWriter {
	return &ExtendedWriterWrapper{writer}
}
