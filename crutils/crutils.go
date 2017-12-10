//// file: crutils.go

// Package crutils ...
// Is a utility package for crawler
package crutils

import (
	"sync/atomic"
)

//// public:

// AtomicInt ...
// Is an atomic 32 bit integer
type AtomicInt int32

// Increment ...
// Increments and returns atomic integer
func (c *AtomicInt) Increment() int32 {
	return atomic.AddInt32((*int32)(c), 1)
}

// Decrement ...
// Decrements and returns atomic integer
func (c *AtomicInt) Decrement() int32 {
	return atomic.AddInt32((*int32)(c), -1)
}

// Get ...
// Returns atomic integer
func (c *AtomicInt) Get() int32 {
	return atomic.LoadInt32((*int32)(c))
}
