// Copyright 2013, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sync2

import (
	"sync"
	"sync/atomic"
	"time"
)

// AtomicInt32 is a wrapper with a simpler interface around atomic.(Add|Store|Load|CompareAndSwap)Int32 functions.
type AtomicInt32 struct {
	int32
}

// NewAtomicInt32 initializes a new AtomicInt32 with a given value.
func NewAtomicInt32(n int32) AtomicInt32 {
	return AtomicInt32{n}
}

// Add atomically adds n to the value.
func (i *AtomicInt32) Add(n int32) int32 {
	return atomic.AddInt32(&i.int32, n)
}

// Set atomically sets n as new value.
func (i *AtomicInt32) Set(n int32) {
	atomic.StoreInt32(&i.int32, n)
}

// Get atomically returns the current value.
func (i *AtomicInt32) Get() int32 {
	return atomic.LoadInt32(&i.int32)
}

// CompareAndSwap atomatically swaps the old with the new value.
func (i *AtomicInt32) CompareAndSwap(oldval, newval int32) (swapped bool) {
	return atomic.CompareAndSwapInt32(&i.int32, oldval, newval)
}

// AtomicUint32 is a wrapper with a simpler interface around atomic.(Add|Store|Load|CompareAndSwap)Uint32 functions.
type AtomicUint32 struct {
	uint32
}

// NewAtomicUint32 initializes a new AtomicUint32 with a given value.
func NewAtomicUint32(n uint32) AtomicUint32 {
	return AtomicUint32{n}
}

// Add atomically adds n to the value.
func (i *AtomicUint32) Add(n uint32) uint32 {
	return atomic.AddUint32(&i.uint32, n)
}

// Set atomically sets n as new value.
func (i *AtomicUint32) Set(n uint32) {
	atomic.StoreUint32(&i.uint32, n)
}

// Get atomically returns the current value.
func (i *AtomicUint32) Get() uint32 {
	return atomic.LoadUint32(&i.uint32)
}

// CompareAndSwap atomatically swaps the old with the new value.
func (i *AtomicUint32) CompareAndSwap(oldval, newval uint32) (swapped bool) {
	return atomic.CompareAndSwapUint32(&i.uint32, oldval, newval)
}

// AtomicInt64 is a wrapper with a simpler interface around atomic.(Add|Store|Load|CompareAndSwap)Int64 functions.
type AtomicInt64 struct {
	int64
}

// NewAtomicInt64 initializes a new AtomicInt64 with a given value.
func NewAtomicInt64(n int64) AtomicInt64 {
	return AtomicInt64{n}
}

// Add atomically adds n to the value.
func (i *AtomicInt64) Add(n int64) int64 {
	return atomic.AddInt64(&i.int64, n)
}

// Set atomically sets n as new value.
func (i *AtomicInt64) Set(n int64) {
	atomic.StoreInt64(&i.int64, n)
}

// Get atomically returns the current value.
func (i *AtomicInt64) Get() int64 {
	return atomic.LoadInt64(&i.int64)
}

// CompareAndSwap atomatically swaps the old with the new value.
func (i *AtomicInt64) CompareAndSwap(oldval, newval int64) (swapped bool) {
	return atomic.CompareAndSwapInt64(&i.int64, oldval, newval)
}

type AtomicDuration int64

func (d *AtomicDuration) Add(duration time.Duration) time.Duration {
	return time.Duration(atomic.AddInt64((*int64)(d), int64(duration)))
}

func (d *AtomicDuration) Set(duration time.Duration) {
	atomic.StoreInt64((*int64)(d), int64(duration))
}

func (d *AtomicDuration) Get() time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(d)))
}

func (d *AtomicDuration) CompareAndSwap(oldval, newval time.Duration) (swapped bool) {
	return atomic.CompareAndSwapInt64((*int64)(d), int64(oldval), int64(newval))
}

// AtomicString gives you atomic-style APIs for string, but
// it's only a convenience wrapper that uses a mutex. So, it's
// not as efficient as the rest of the atomic types.
type AtomicString struct {
	mu  sync.Mutex
	str string
}

func (s *AtomicString) Set(str string) {
	s.mu.Lock()
	s.str = str
	s.mu.Unlock()
}

func (s *AtomicString) Get() string {
	s.mu.Lock()
	str := s.str
	s.mu.Unlock()
	return str
}

func (s *AtomicString) CompareAndSwap(oldval, newval string) (swqpped bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.str == oldval {
		s.str = newval
		return true
	}
	return false
}
