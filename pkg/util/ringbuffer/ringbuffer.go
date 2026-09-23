package ringbuffer

import (
	"io"
)

// Buffer is a simple implementation of a ring/circular buffer.
type Buffer struct {
	buf []byte
	// Picture [read, write] as a sliding window. These values grow without
	// bounds, read not being allowed to exceed write, but are handled
	// modularly during |buf| i/o considerations.
	read  int
	write int
}

// NewBuffer returns a ring buffer of a given size.
func NewBuffer(size uint) *Buffer {
	return &Buffer{
		buf: make([]byte, size),
	}
}

// Read reads from the buffer, returning io.EOF if it has read up until where
// it has written.
func (b *Buffer) Read(p []byte) (int, error) {
	maxBytes := min(len(p), b.Len())
	b.copyToBuffer(p[:maxBytes], b.read)
	b.read += maxBytes
	if maxBytes == 0 {
		return 0, io.EOF
	}
	return maxBytes, nil
}

// Write writes to the buffer, circularly overwriting data if p exceeds the
// size of the buffer.
func (b *Buffer) Write(p []byte) (int, error) {
	total := len(p)
	for len(p) != 0 {
		// We don't want b.write to get more then len(b.buf) ahead of b.read; we
		// read as much as possible taking that into account.
		maxBytes := min(len(p), len(b.buf)-(b.write-b.read))
		// If b.write and b.read are maximally far apart, we can overwrite
		// len(p) or len(b.buf) many bytes.
		if maxBytes == 0 {
			maxBytes = min(len(p), len(b.buf))
			b.read += maxBytes
		}
		b.copyFromBuffer(p[:maxBytes], b.write)
		b.write += maxBytes
		p = p[maxBytes:]
	}
	return total, nil
}

func (b *Buffer) Close() error { return nil }

func (b *Buffer) Len() int {
	return b.write - b.read
}

func (b *Buffer) copyToBuffer(p []byte, start int) {
	N := len(b.buf)
	P := len(p)
	// Assume P <= N.
	if P > N {
		panic("copyToBuffer: expects len(p) <= size of Buffer")
	}
	start = start % N
	if start+P <= N {
		copy(p, b.buf[start:P+start])
	} else {
		copy(p[:N-start], b.buf[start:])
		copy(p[N-start:], b.buf[:P-(N-start)])
	}
}

func (b *Buffer) copyFromBuffer(p []byte, start int) {
	N := len(b.buf)
	P := len(p)
	// Assume P <= N.
	if P > N {
		panic("copyFromBuffer: expects len(p) <= size of Buffer")
	}
	start = start % N
	if start+P <= N {
		copy(b.buf[start:start+P], p)
	} else {
		copy(b.buf[start:], p[:N-start])
		copy(b.buf[:P-(N-start)], p[N-start:])
	}
}
