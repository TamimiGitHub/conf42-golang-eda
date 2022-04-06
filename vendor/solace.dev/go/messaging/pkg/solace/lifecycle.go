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

package solace

import "time"

// LifecycleControl contains lifecycle functionality common to
// various messaging  services such as publishers and receivers.
type LifecycleControl interface {
	// Start starts the messaging service synchronously.
	// Before this function is called, the messaging service is considered
	// off-duty. To operate normally, this function must be called on
	// a receiver or publisher instance. This function is idempotent.
	// Returns an error if one occurred or nil if successful.
	Start() error

	// StartAsync starts the messaging service asynchronously.
	// Before this function is called, the messaging service is considered
	// off-duty. To operate normally, this function must be called on
	// a receiver or publisher instance. This function is idempotent.
	// Returns a channel that will receive an error if one occurred or
	// nil if successful. Subsequent calls will return additional
	// channels that can await an error, or nil if already started.
	StartAsync() <-chan error

	// Terminate terminates the messaging service gracefully and synchronously.
	// This function is idempotent. The only way to resume operation
	// after this function is called is to create another instance.
	// Any attempt to call this function renders the instance
	// permanently terminated, even if this function completes.
	// A graceful shutdown is attempted within the specified grace period (gracePeriod).
	// Setting gracePeriod to 0 implies a non-graceful shutdown that ignores
	// unfinished tasks or in-flight messages.
	// This function returns an error if one occurred, or
	// nil if it successfully and gracefully terminated.
	// If gracePeriod is set to less than 0, the function waits indefinitely.
	Terminate(gracePeriod time.Duration) error

	// TerminateAsync terminates the messaging service asynchronously.
	// This function is idempotent. The only way to resume operation
	// after this function is called is to create another instance.
	// Any attempt to call this function renders the instance
	// permanently terminated, even if this function completes.
	// A graceful shutdown is attempted within the specified grace period (gracePeriod).
	// Setting gracePeriod to 0 implies a non-graceful shutdown that ignores
	// unfinished tasks or in-flight messages.
	// This function returns a channel that receives an error if one occurred, or
	// nil if it successfully and gracefully terminated.
	// If gracePeriod is set to less than 0, the function waits indefinitely.
	TerminateAsync(gracePeriod time.Duration) <-chan error

	// IsRunning checks if the process was successfully started and not yet stopped.
	// Returns true if running, otherwise false.
	IsRunning() bool

	// IsTerminates checks if the message-delivery process is terminated.
	// Returns true if terminated, otherwise false.
	IsTerminated() bool

	// IsTerminating checks if the message-delivery process termination is ongoing.
	// Returns true if the message message-delivery process is being terminated,
	// but termination is not yet complete, otherwise false.
	IsTerminating() bool

	// SetTerminationNotificationListener adds a callback to listen for
	// non-recoverable interruption events.
	SetTerminationNotificationListener(listener TerminationNotificationListener)
}

// TerminationNotificationListener represents a listener that can be
// registered with a LifecycleControl instance to listen for
// termination events.
type TerminationNotificationListener func(TerminationEvent)

// TerminationEvent represents a non-recoverable receiver or publisher
// unsolicited termination for which applications can listen.
type TerminationEvent interface {
	// GetTimestamp retrieves the timestamp of the event.
	GetTimestamp() time.Time
	// GetMessage retrieves the event message.
	GetMessage() string
	// GetCause retrieves the cause of the client exception, if any.
	// Returns the error event or nil if no cause is present.
	GetCause() error
}
