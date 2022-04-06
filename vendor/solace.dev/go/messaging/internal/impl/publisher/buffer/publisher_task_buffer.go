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

// Package buffer is defined below
package buffer

import (
	"time"

	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/internal/impl/logging"
)

// PublisherTask task
type PublisherTask func(chan struct{})

// PublisherTaskBuffer interface
type PublisherTaskBuffer interface {
	// the main executor loop that should be started on a new goroutine
	Run()
	// Call to submit into the buffer, will succeed unless we are terminating
	Submit(task PublisherTask) bool
	// Call to terminate that will attempt to shutdown gracefully
	Terminate(timer *time.Timer) bool
	// Call terminate without waiting for the event loop to complete
	TerminateNow()
}

// NewChannelBasedPublisherTaskBuffer function
func NewChannelBasedPublisherTaskBuffer(capacity int, sharedTaskQueueGetter func() chan core.SendTask) PublisherTaskBuffer {
	return &channelBasedPublisherTaskBuffer{
		channel:               make(chan PublisherTask, capacity),
		sharedTaskQueueGetter: sharedTaskQueueGetter,
		terminateUngraceful:   make(chan struct{}),
		terminateGraceful:     make(chan struct{}),
		terminateComplete:     make(chan struct{}),
		terminateTask:         make(chan struct{}),
	}
}

// The task buffer can be used to queue tasks, and can be shutdown gracefully and drained.
// This task buffer is independent of the message publisher's backpressure.
// The task buffer is implemented without mutexes as it needs to be as fast as possible
type channelBasedPublisherTaskBuffer struct {
	channel               chan PublisherTask
	sharedTaskQueueGetter func() chan core.SendTask

	terminateUngraceful chan struct{}
	terminateGraceful   chan struct{}
	terminateComplete   chan struct{}
	terminateTask       chan struct{}
}

// Tasks may be submitted until the buffer is terminated. This should all but guarantee that
// we do not get into a situation where a message is published to backpressure but no task is
// created. See note in DirectMessagePublisherImpl publish function about potential race.
// Returns false if the task could not be submitted. The owning publisher must keep track of
// the space available in this buffer and not submit additional tasks.
func (buffer *channelBasedPublisherTaskBuffer) Submit(task PublisherTask) bool {
	select {
	case <-buffer.terminateComplete:
		// this will return immediately if terminateComplete is closed
		return false
	default:
		select {
		case buffer.channel <- task:
			return true
		default:
			// uh oh, the task buffer is full. This should not happen in regular use.
			logging.Default.Debug("Attempted to submit task to task buffer, but the buffer was full!")
			return false
		}
	}
}

func (buffer *channelBasedPublisherTaskBuffer) Terminate(timer *time.Timer) bool {
	close(buffer.terminateGraceful)
	if timer == nil {
		<-buffer.terminateComplete
		return true
	}
	select {
	case <-buffer.terminateComplete:
		// yay graceful termination
		return true
	case <-timer.C:
		// terminate the task if there is one
		close(buffer.terminateTask)
		// time for ungraceful termaintion
		close(buffer.terminateUngraceful)
		// this should come soon
		<-buffer.terminateComplete
		return false
	}
}

func (buffer *channelBasedPublisherTaskBuffer) TerminateNow() {
	close(buffer.terminateGraceful)
	close(buffer.terminateTask)
	close(buffer.terminateUngraceful)
}

func (buffer *channelBasedPublisherTaskBuffer) Run() {
	// loop that simply removes tasks from the backpressure and pushes tasks to the shared internal publisher
	taskCompleteNotify := make(chan bool)
loop:
	for {
		// then check if we are terminating ungracefully and should exit now
		select {
		case <-buffer.terminateUngraceful:
			break loop
		default:
			// do nothing when we don't get an ungraceful termination signal
		}
		var task PublisherTask
		select {
		case <-buffer.terminateGraceful:
			// we are terminating gracefully, drain the buffer and notify when terminated
			select {
			case task = <-buffer.channel:
			default:
				// no more tasks, time to exit!
				close(buffer.terminateComplete)
				return
			}
		default:
			select {
			// if we are not yet terminated, block on a task or block on terminate graceful
			case task = <-buffer.channel:
			case <-buffer.terminateGraceful:
				// if we get a signal to terminate gracefully, interrup the wait for a task and loop
				continue
			}
		}
		// reached if we have a task to act on
		// we want to wait for this task to complete before moving on to the next iteration of the loop
		sendTask := core.SendTask(func() {
			// first execute the task synchronously, then notify of completion
			task(buffer.terminateTask)
			taskCompleteNotify <- true
		})
		select {
		case buffer.sharedTaskQueueGetter() <- sendTask:
			// submitted task successfully, now we wait for that send task to complete
			<-taskCompleteNotify
		case <-buffer.terminateUngraceful:
			// we got a signal to terminate ungracefully, don't wait to submit the task and return immediately instead.
			break loop
		}
	}
	// notify done, we are ungraceful at this point
	close(buffer.terminateComplete)
}
