/*
 * MIT License
 *
 * Copyright (c) 2022-2023 Tochemey
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package actors

import (
	"go.uber.org/atomic"
)

// Mailbox defines the actor mailbox.
// Any implementation should be a thread-safe FIFO
type Mailbox interface {
	// Buffer returns a blocking channel
	Buffer() chan ReceiveContext
	// Push pushes a message into the mailbox. This returns an error
	// when the mailbox is full
	Push(msg ReceiveContext) error
	// Pop fetches a message from the mailbox. This returns an error
	// when the mailbox is empty
	Pop() (msg ReceiveContext, err error)
	// IsEmpty returns true when the mailbox is empty
	IsEmpty() bool
	// IsFull returns true when the mailbox is full
	IsFull() bool
	// Size returns the size of the buffer atomically
	Size() uint64
	// Reset resets the mailbox
	Reset()
	// Clone clones the current mailbox and returns a new Mailbox with reset settings
	Clone() Mailbox
	// Capacity returns the mailbox capacity atomically
	Capacity() uint64
}

// receiveContextBuffer is the actor default inbox
type receiveContextBuffer struct {
	// specifies capacity of the buffer
	capacity *atomic.Uint64
	buffer   chan ReceiveContext
}

// newReceiveContextBuffer creates a Mailbox with a fixed capacity
func newReceiveContextBuffer(capacity uint64) Mailbox {
	return &receiveContextBuffer{
		capacity: atomic.NewUint64(capacity),
		buffer:   make(chan ReceiveContext, capacity),
	}
}

// enforce compilation error
var _ Mailbox = &receiveContextBuffer{}

// Buffer returns a blocking channel
func (x *receiveContextBuffer) Buffer() chan ReceiveContext {
	return x.buffer
}

// Push pushes a message into the mailbox. This returns an error
// when the box is full
func (x *receiveContextBuffer) Push(msg ReceiveContext) error {
	select {
	case x.buffer <- msg:
		return nil
	default:
		return ErrFullMailbox
	}
}

// Pop fetches a message from the mailbox. This returns an error
// when the mailbox is empty
func (x *receiveContextBuffer) Pop() (msg ReceiveContext, err error) {
	select {
	case msg := <-x.buffer:
		return msg, nil
	default:
		return nil, ErrEmptyMailbox
	}
}

// IsEmpty returns true when the buffer is empty
func (x *receiveContextBuffer) IsEmpty() bool {
	return len(x.buffer) == 0
}

// Size returns the size of the buffer
func (x *receiveContextBuffer) Size() uint64 {
	return uint64(len(x.buffer))
}

// Clone clones the current mailbox and returns a new Mailbox with reset settings
func (x *receiveContextBuffer) Clone() Mailbox {
	return &receiveContextBuffer{
		capacity: x.capacity,
		buffer:   make(chan ReceiveContext, x.capacity.Load()),
	}
}

// Reset resets the mailbox
func (x *receiveContextBuffer) Reset() {
	x.buffer = make(chan ReceiveContext, x.capacity.Load())
}

// IsFull returns true when the mailbox is full
func (x *receiveContextBuffer) IsFull() bool {
	return x.Size() >= x.capacity.Load()
}

// Capacity implements Mailbox.
func (x *receiveContextBuffer) Capacity() uint64 {
	return x.capacity.Load()
}
