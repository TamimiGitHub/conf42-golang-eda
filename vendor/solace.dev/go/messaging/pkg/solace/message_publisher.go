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

// MessagePublisher represents the shared functionality between all publisher instances.
type MessagePublisher interface {
	// Extend LifecycleControl for various lifecycle management functionality.
	LifecycleControl
}

// MessagePublisherHealthCheck allows applications to check and listen for events
// that indicate when message publishers are ready to publish. This is often used
// to handle various back pressure schemes, such as reject on full, and allows
// publishing to stop until the publisher can begin accepting more messages.
type MessagePublisherHealthCheck interface {
	// IsReady checks if the publisher can publish messages. Returns true if the
	// publisher can publish messages, otherwise false if the publisher is prevented from
	// sending messages (e.g., a full buffer or I/O problems).
	IsReady() bool

	// SetPublisherReadinessListener registers a listener to be called when the
	// publisher can send messages. Typically, the listener is notified after a
	// Publisher instance raises an error indicating that the outbound message
	// buffer is full.
	SetPublisherReadinessListener(listener PublisherReadinessListener)

	// NotifyWhenReady makes a request to notify the application when the
	// publisher is ready. This function triggers a readiness notification if one
	// needs to be sent, otherwise the next readiness notification is
	// processed.
	NotifyWhenReady()
}

// PublisherReadinessListener defines a function that can be registered to
// receive notifications from a publisher instance for readiness.
type PublisherReadinessListener func()
