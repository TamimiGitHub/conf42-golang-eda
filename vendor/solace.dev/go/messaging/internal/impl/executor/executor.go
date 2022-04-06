// pubsubplus-go-client
//
// Copyright 2021-2022 Solace Corporation. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package executor is defined below
package executor

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// defaultSize is the default size fo the channel
const defaultSize = 1024

// scalingFactor is the factor to scale by when the channel is full
const scalingFactor = 2

// Task represents a task that can be submitted to an executor
type Task func()

// Executor interface
type Executor interface {
	Run()
	Submit(task Task) bool
	// Terminate will gracefully shut down the executor, stop allowing more tasks but will continue to process submitted tasks
	Terminate()
	// AwaitTermination will terminate the executor gracefully and wait for all outstanding tasks to be completed
	// AwaitTeramintion may be called after Terminate or TerminateNow.
	AwaitTermination()
}

// Exectuor implements an unbonuded executor service serializing the events
// on a single goroutine. It can be started with executor.Run()
type executor struct {
	channel           unsafe.Pointer
	channelLock       sync.RWMutex
	stateLock         sync.RWMutex
	closed            int32
	scale             int64
	terminateComplete chan struct{}
}

// Run runs the main executor. This may be run in its own goroutine, may be locked
// to a single thread, etc.
func (executor *executor) Run() {
loop:
	for {
		// read lock and wait for a new task
		executor.channelLock.RLock()
		// if the channel is closed, ok will be false and this call will return immediately
		task, ok := <-executor.getChannel()
		executor.channelLock.RUnlock()
		if !ok {
			// the channel has been closed, we can terminate
			break loop
		}
		task()
	}
	close(executor.terminateComplete)
}

func (executor *executor) getChannel() chan Task {
	return *(*chan Task)(atomic.LoadPointer(&executor.channel))
}

// Terminate is a blocking call to terminate the publisher and wait for the
// terminate signal. This will close the inbound channel meaning no more tasks
// can be submitted, but the existing tasks will be executed until the grace period
// expires at which point no more tasks will be executed and the executor will
// shut down.
func (executor *executor) AwaitTermination() {
	executor.Terminate()
	<-executor.terminateComplete
}

func (executor *executor) Terminate() {
	// start off by closing the channel meaning no more tasks will be submitted
	// acquire a hard lock to make sure that no tasks are submitted racing with the closed
	executor.stateLock.Lock()
	defer executor.stateLock.Unlock()
	if atomic.CompareAndSwapInt32(&executor.closed, 0, 1) {
		close(executor.getChannel())
	}
}

// Submit will submit a new task to the executor. If the channel is full, a new one
// will be allocated and all tasks will be transferred. Will return true if the task
// can be submitted, or false if the executor service is terminating/terminated.
func (executor *executor) Submit(task Task) (submitted bool) {
	// the state lock should only be in contention with a call to terminate
	executor.stateLock.RLock()
	defer executor.stateLock.RUnlock()
	if atomic.LoadInt32(&executor.closed) != 0 {
		return false
	}
	// acquire the channel read lock, should only be in contention with other resizes
	// this will not be in contention with Run above as multiple threads can acquire the read lock
	executor.channelLock.RLock()
	select {
	case executor.getChannel() <- task:
		// we successfully submitted, no need to resize!
		executor.channelLock.RUnlock()
	default:
		executor.channelLock.RUnlock()
		// relock with the write lock, this requires all read locks to be released. Run will not hold the
		// read lock for long when we are going to resize.
		executor.channelLock.Lock()
		defer executor.channelLock.Unlock()
		// first try to submit the task again since its possible that this races with another resize
		select {
		case executor.getChannel() <- task:
			// submitted successfully, no resize required
		default:
			// resize the channel
			executor.resize()
			executor.getChannel() <- task
		}
	}
	return true
}

// resize will increase the size of the channel by channelSize * scalingFactor
// this function is not thread safe and should be wrapped in locks
func (executor *executor) resize() {
	// create a new channel, close the old channel and drain it, moving all items to the new channel
	// during this time, Run will not be able to get any more tasks as this function will be surrounded by locks
	newScale := scalingFactor * atomic.LoadInt64(&executor.scale)
	atomic.StoreInt64(&executor.scale, newScale)
	newChannel := make(chan Task, newScale)
	oldChannel := executor.getChannel()
	close(oldChannel)
	for t := range executor.getChannel() {
		newChannel <- t
	}
	atomic.StorePointer(&executor.channel, unsafe.Pointer(&newChannel))
}

// NewExecutor allocates a new executor with default parameters.
func NewExecutor() Executor {
	executor := &executor{}
	channel := make(chan Task, defaultSize)
	// don't have to be threadsafe here, no need for atomic
	executor.channel = unsafe.Pointer(&channel)
	executor.channelLock = sync.RWMutex{}
	executor.stateLock = sync.RWMutex{}
	executor.scale = defaultSize
	executor.closed = 0
	executor.terminateComplete = make(chan struct{})
	return executor
}
