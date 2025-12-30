package main

import (
	"bytes"
	"io"
)

// bodyCapturer wraps an io.ReadCloser, capturing read data up to a limit.
type bodyCapturer struct {
	rc     io.ReadCloser
	buf    bytes.Buffer
	limit  int64
	read   int64
}

func newBodyCapturer(rc io.ReadCloser, limit int64) *bodyCapturer {
	return &bodyCapturer{
		rc:    rc,
		limit: limit,
	}
}

func (b *bodyCapturer) Read(p []byte) (n int, err error) {
	n, err = b.rc.Read(p)
	if n > 0 {
		// Calculate how much space is left in the buffer relative to limit
		remaining := b.limit - b.read
		if remaining > 0 {
			toCapture := int64(n)
			if toCapture > remaining {
				toCapture = remaining
			}
			b.buf.Write(p[:toCapture])
			b.read += toCapture
		}
	}
	return n, err
}

func (b *bodyCapturer) Close() error {
	return b.rc.Close()
}

// GetBody returns the captured body as a byte slice.
func (b *bodyCapturer) GetBody() []byte {
	return b.buf.Bytes()
}
