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

// PersistentMessageReceiver allows for receiving persistent message (guaranteed messages).
type PersistentMessageReceiver interface {
	MessageReceiver // Include all functionality of MessageReceiver.

	// Ack acknowledges that a  message was received.
	Ack(message message.InboundMessage) error

	// StartAsyncCallback starts the PersistentMessageReceiver asynchronously.
	// Calls the callback when started with an error if one occurred, otherwise nil
	// if successful.
	StartAsyncCallback(callback func(PersistentMessageReceiver, error))

	// TerminateAsyncCallback terminates the PersistentMessageReceiver asynchronously.
	// Calls the callback when terminated with nil if successful, otherwise an error if
	// one occurred. If gracePeriod is less than 0, the function waits indefinitely.
	TerminateAsyncCallback(gracePeriod time.Duration, callback func(error))

	// ReceiveAsync registers a callback to be called when new messages
	// are received. Returns an error if one occurred while registering the callback.
	// If a callback is already registered, it is replaced by the specified
	// callback.
	ReceiveAsync(callback MessageHandler) error

	// ReceiveMessage receives a message synchronously from the receiver.
	// Returns an error if the receiver is not started or already terminated.
	// This function waits until the specified timeout to receive a message or waits
	// forever if timeout value is negative. If a timeout occurs, a solace.TimeoutError
	// is returned.
	ReceiveMessage(timeout time.Duration) (message.InboundMessage, error)

	// Pause pauses the receiver's message delivery to asynchronous message handlers.
	// Pausing an already paused receiver has no effect.
	// Returns an IllegalStateError if the receiver has not started or has already terminated.
	Pause() error

	// Resume unpause the receiver's message delivery to asynchronous message handlers.
	// Resume a receiver that is not paused has no effect.
	// Returns an IllegalStateError if the receiver has not started or has already terminated.
	Resume() error

	// ReceiverInfo returns a runtime accessor for the receiver information such as the remote
	// resource to which it connects.
	// Returns an IllegalStateError if the receiver has not started or has already terminated.
	ReceiverInfo() (info PersistentReceiverInfo, err error)
}

// PersistentMessageReceiverBuilder is used for configuration of PersistentMessageReceiver.
type PersistentMessageReceiverBuilder interface {
	// Build creates a  PersistentMessageReceiver with the specified properties.
	// Returns solace/errors.*IllegalArgumentError if the queue is nil.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	Build(queue *resource.Queue) (receiver PersistentMessageReceiver, err error)

	// WithActivationPassivationSupport sets the listener to receiver broker notifications
	// about state changes for the resulting receiver. This change can happen if there are
	// multiple instances of the same receiver for high availability and activity is exchanged.
	// This change is handled by the broker.
	WithActivationPassivationSupport(listener ReceiverStateChangeListener) PersistentMessageReceiverBuilder

	// WithMessageAutoAcknowledgement enables automatic acknowledgement on all receiver methods.
	// Auto Acknowledgement will be performed after an acknowledgement after the receive callback
	// is called when using ReceiveAsync, or after the message is passed to the application when
	// using ReceiveMessage. In cases where underlying network connectivity fails, automatic
	// acknowledgement processing is not guaranteed.
	WithMessageAutoAcknowledgement() PersistentMessageReceiverBuilder

	// WithMessageClientAcknowledgement disables automatic acknowledgement on all receiver methods
	// and instead enables support for client acknowledgement for both synchronous and asynchronous
	// message delivery functions. New persistent receiver builders default to client acknowledgement.
	WithMessageClientAcknowledgement() PersistentMessageReceiverBuilder

	// WithMessageSelector sets the message selector to the specified string.
	// If an empty string is provided, the filter is cleared.
	WithMessageSelector(filterSelectorExpression string) PersistentMessageReceiverBuilder

	// WithMissingResourcesCreationStrategy sets the missing resource creation strategy
	// defining what actions the API may take when missing resources are detected.
	WithMissingResourcesCreationStrategy(strategy config.MissingResourcesCreationStrategy) PersistentMessageReceiverBuilder

	// WithMessageReplay enables support for message replay using a specific replay
	// strategy. Once started, the receiver replays using the specified strategy.
	// Valid strategies include config.ReplayStrategyAllMessages(),
	// config.ReplayStrategyTimeBased and config.ReplicationGroupMessageIDReplayStrategy.
	WithMessageReplay(strategy config.ReplayStrategy) PersistentMessageReceiverBuilder

	// WithSubscriptions sets a list of TopicSubscriptions to subscribe
	// to when starting the receiver. Accepts *resource.TopicSubscription subscriptions.
	WithSubscriptions(topics ...resource.Subscription) PersistentMessageReceiverBuilder

	// FromConfigurationProvider configures the persistent receiver with the specified properties.
	// The built-in ReceiverPropertiesConfigurationProvider implementations include:
	//   ReceiverPropertyMap, a map of ReceiverProperty keys to values
	FromConfigurationProvider(provider config.ReceiverPropertiesConfigurationProvider) PersistentMessageReceiverBuilder
}

// ReceiverState represents the various states the receiver can be in, such as Active or Passive.
// This property is controlled by the remote broker.
type ReceiverState byte

const (
	// ReceiverActive is the state in which the receiver receives messages from a broker.
	ReceiverActive ReceiverState = iota
	// ReceiverPassive is the state in which the receiver would not receive messages from a broker.
	// Often, this is because another instance of the receiver became active, so this instance became
	// passive.
	ReceiverPassive
)

// ReceiverStateChangeListener is a listener that can be registered on a receiver to be
// notified of changes in receiver state by the remote broker.
type ReceiverStateChangeListener func(oldState ReceiverState, newState ReceiverState, timestamp time.Time)

// PersistentReceiverInfo provides information about the receiver at runtime.
type PersistentReceiverInfo interface {
	// GetResourceInfo returns the remote endpoint (resource) information for the receiver.
	GetResourceInfo() ResourceInfo
}

// ResourceInfo provides information about a resouce at runtime.
type ResourceInfo interface {
	// GetName returns the name of the resource.
	GetName() string
	// IsDurable returns true is a resource is durable, otherwise false.
	IsDurable() bool
}
