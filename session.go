package main

import "sync"

// RingBuffer is a fixed-size circular byte buffer for reconnect replay.
type RingBuffer struct {
	buf  []byte
	size int
	w    int
	full bool
	mu   sync.Mutex
}

// NewRingBuffer creates a ring buffer with the given capacity in bytes.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buf:  make([]byte, size),
		size: size,
	}
}

// Write appends data to the ring buffer, overwriting oldest data if full.
func (rb *RingBuffer) Write(p []byte) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for len(p) > 0 {
		// If write is larger than entire buffer, skip to last buffer-sized chunk
		if len(p) >= rb.size {
			copy(rb.buf, p[len(p)-rb.size:])
			rb.w = 0
			rb.full = true
			return
		}

		n := copy(rb.buf[rb.w:], p)
		p = p[n:]
		rb.w += n
		if rb.w >= rb.size {
			rb.w = 0
			rb.full = true
		}
	}
}

// Bytes returns all data in the buffer in chronological order.
func (rb *RingBuffer) Bytes() []byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.full {
		out := make([]byte, rb.w)
		copy(out, rb.buf[:rb.w])
		return out
	}

	out := make([]byte, rb.size)
	n := copy(out, rb.buf[rb.w:])
	copy(out[n:], rb.buf[:rb.w])
	return out
}
