package nonblockingWriter

import (
	"bufio"
	"io"
)

type NonBlockingWriter chan []byte

func (b NonBlockingWriter) Write(p []byte) (int, error) {
	newSlice := make([]byte, len(p))
	copy(newSlice, p)
	b <- newSlice
	return len(p), nil
}

func (b NonBlockingWriter) Close() error {
	close(b)
	return nil
}

func NewBufWriter(writer io.Writer, cap int) NonBlockingWriter {
	w := make(NonBlockingWriter, cap)
	var bufWriter *bufio.Writer

	if _, ok := writer.(*bufio.Writer); !ok {
		bufWriter = bufio.NewWriterSize(writer, cap)
	} else {
		bufWriter = writer.(*bufio.Writer)
	}

	go func() {
		for p := range w {
			_, _ = bufWriter.Write(p)
			_ = bufWriter.Flush()
		}
	}()
	return w
}
