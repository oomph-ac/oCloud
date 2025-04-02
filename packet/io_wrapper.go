package packet

import (
	"io"
)

type ByteWriterWrapper[T interface {
	io.Writer
	io.Closer
}] struct {
	w T
}

func NewByteWriterWrapper[T interface {
	io.Writer
	io.Closer
}](w T) *ByteWriterWrapper[T] {
	return &ByteWriterWrapper[T]{w: w}
}

func (w *ByteWriterWrapper[T]) Interface() T {
	return w.w
}

func (w *ByteWriterWrapper[T]) Write(p []byte) (n int, err error) {
	return w.w.Write(p)
}

func (w *ByteWriterWrapper[T]) WriteByte(b byte) error {
	_, err := w.w.Write([]byte{b})
	return err
}

func (w *ByteWriterWrapper[T]) Close() error {
	return w.w.Close()
}
