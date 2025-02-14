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
	"sync"

	"go.uber.org/atomic"
)

// Mailbox defines the actor mailbox.
// Any implementation should be a thread-safe FIFO
type Mailbox interface {
	// Push pushes a message into the mailbox. This returns an error
	// when the box is full
	Push(msg ReceiveContext) error
	// Pop fetches a message from the mailbox
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
	// specifies the number of messages to stash
	capacity *atomic.Uint64
	counter  *atomic.Uint64
	buffer   chan ReceiveContext
	mu       sync.Mutex
}

// newReceiveContextBuffer creates a Mailbox with a fixed capacity
func newReceiveContextBuffer(capacity uint64) Mailbox {
	return &receiveContextBuffer{
		capacity: atomic.NewUint64(capacity),
		buffer:   make(chan ReceiveContext, capacity),
		counter:  atomic.NewUint64(0),
		mu:       sync.Mutex{},
	}
}

// enforce compilation error
var _ Mailbox = &receiveContextBuffer{}

// Push pushes a message into the mailbox. This returns an error
// when the box is full
func (x *receiveContextBuffer) Push(msg ReceiveContext) error {
	// check whether the buffer is full
	if x.Size() < x.capacity.Load() {
		x.mu.Lock()
		x.buffer <- msg
		x.mu.Unlock()
		x.counter.Inc()
		return nil
	}
	return ErrFullMailbox
}

// Pop fetches a message from the mailbox
func (x *receiveContextBuffer) Pop() (msg ReceiveContext, err error) {
	// check whether the buffer is empty
	if x.IsEmpty() {
		return nil, ErrEmptyMailbox
	}
	// grab the message
	x.mu.Lock()
	msg = <-x.buffer
	x.mu.Unlock()
	x.counter.Dec()
	return
}

// IsEmpty returns true when the buffer is empty
func (x *receiveContextBuffer) IsEmpty() bool {
	return x.counter.Load() == 0
}

// Size returns the size of the buffer
func (x *receiveContextBuffer) Size() uint64 {
	return x.counter.Load()
}

// Clone clones the current mailbox and returns a new Mailbox with reset settings
func (x *receiveContextBuffer) Clone() Mailbox {
	return &receiveContextBuffer{
		capacity: x.capacity,
		counter:  atomic.NewUint64(0),
		buffer:   make(chan ReceiveContext, x.capacity.Load()),
		mu:       sync.Mutex{},
	}
}

// Reset resets the mailbox
func (x *receiveContextBuffer) Reset() {
	x.counter.Store(0)
	x.mu.Lock()
	x.buffer = make(chan ReceiveContext, x.capacity.Load())
	x.mu.Unlock()
}

// IsFull returns true when the mailbox is full
func (x *receiveContextBuffer) IsFull() bool {
	return x.Size() >= x.capacity.Load()
}

// Capacity implements Mailbox.
func (x *receiveContextBuffer) Capacity() uint64 {
	return x.capacity.Load()
}
