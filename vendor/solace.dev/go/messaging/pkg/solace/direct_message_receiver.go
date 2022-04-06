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

import (
	"time"

	"solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/resource"
)

// The DirectMessageReceiver is used to receive direct messages.
type DirectMessageReceiver interface {
	MessageReceiver // Include all functionality of MessageReceiver.

	// StartAsyncCallback starts the DirectMessageReceiver asynchronously.
	// Calls the callback when started with an error if one occurred, otherwise nil
	// if successful.
	StartAsyncCallback(callback func(DirectMessageReceiver, error))

	// TerminateAsyncCallback terminates the DirectMessageReceiver asynchronously.
	// Calls the callback when terminated with nil if successful or an error if
	// one occurred. If gracePeriod is less than 0, the function will wait indefinitely.
	TerminateAsyncCallback(gracePeriod time.Duration, callback func(error))

	// ReceiveAsync registers a callback to be called when new messages
	// are received. Returns an error if one occurred while registering the callback.
	// If a callback is already registered, it will be replaced by the specified
	// callback.
	ReceiveAsync(callback MessageHandler) error

	// ReceiveMessage receives a inbound message synchronously from the receiver.
	// Returns an error if the receiver has not started, or has already terminated.
	// ReceiveMessage waits until the specified timeout to receive a message, or will wait
	// forever if the timeout specified is a negative value. If a timeout occurs, a solace.TimeoutError
	// is returned.
	ReceiveMessage(timeout time.Duration) (received message.InboundMessage, err error)
}

// DirectMessageReceiverBuilder allows for configuration of DirectMessageReceiver instances.
type DirectMessageReceiverBuilder interface {
	// Build creates a DirectMessageReceiver with the specified properties.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	Build() (messageReceiver DirectMessageReceiver, err error)
	// BuildWithShareName creates DirectMessageReceiver with the specified ShareName.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided
	// or the specified ShareName is invalid.
	BuildWithShareName(shareName *resource.ShareName) (messageReceiver DirectMessageReceiver, err error)
	// OnBackPressureDropLatest configures the receiver with the specified buffer size. If the buffer
	// is full and a message arrives, the incoming message is discarded.
	// A buffer of the given size will be statically allocated when the receiver is built.
	// The bufferCapacity must be greater than or equal to 1.
	OnBackPressureDropLatest(bufferCapacity uint) DirectMessageReceiverBuilder
	// OnBackPressureDropOldest configures the receiver with the specified buffer size, bufferCapacity. If the buffer
	// is full and a message arrives, the oldest message in the buffer is discarded.
	// A buffer of the given size will be statically allocated when the receiver is built.
	// The value of bufferCapacity must be greater than or equal to 1.
	OnBackPressureDropOldest(bufferCapacity uint) DirectMessageReceiverBuilder
	// WithSubscriptions sets a list of TopicSubscriptions to subscribe
	// to when starting the receiver. This function also accepts *resource.TopicSubscription subscriptions.
	WithSubscriptions(topics ...resource.Subscription) DirectMessageReceiverBuilder
	// FromConfigurationProvider configures the DirectMessageReceiver with the specified properties.
	// The built-in ReceiverPropertiesConfigurationProvider implementations include:
	// - ReceiverPropertyMap - A map of ReceiverProperty keys to values.
	FromConfigurationProvider(provider config.ReceiverPropertiesConfigurationProvider) DirectMessageReceiverBuilder
}
