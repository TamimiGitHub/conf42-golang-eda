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
	"solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/resource"
)

// MessageReceiver represents the shared functionality between all MessageReceivers
type MessageReceiver interface {
	// Extend LifecycleControl for various lifecycle management functionality
	LifecycleControl

	// AddSubscription will subscribe to another message source on a PubSub+ Broker to receive messages from.
	// Will block until subscription is added. Accepts *resource.TopicSubscription instances as the subscription.
	// Returns a solace/errors.*IllegalStateError if the service is not running.
	// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
	// Returns nil if successful.
	AddSubscription(subscription resource.Subscription) error

	// RemoveSubscription will unsubscribe from a previously subscribed message source on a broker
	// such that no more messages will be received from it.
	// Will block until subscription is removed.
	// Accepts *resource.TopicSubscription instances as the subscription.
	// Returns an solace/errors.*IllegalStateError if the service is not running.
	// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
	// Returns nil if successful.
	RemoveSubscription(subscription resource.Subscription) error

	// AddSubscriptionAsync will subscribe to another message source on a PubSub+ Broker to receive messages from.
	// Will block until subscription is added. Accepts *resource.TopicSubscription instances as the subscription.
	// Returns a solace/errors.*IllegalStateError if the service is not running.
	// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
	// Returns nil if successful.
	AddSubscriptionAsync(subscription resource.Subscription, listener SubscriptionChangeListener) error

	// RemoveSubscriptionAsymc will unsubscribe from a previously subscribed message source on a broker
	// such that no more messages will be received from it. Will block until subscription is removed.
	// Accepts *resource.TopicSubscription instances as the subscription.
	// Returns an solace/errors.*IllegalStateError if the service is not running.
	// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
	// Returns nil if successful.
	RemoveSubscriptionAsync(subscription resource.Subscription, listener SubscriptionChangeListener) error
}

// SubscriptionOperation represents the operation that triggered a SubscriptionChangeListener callback
type SubscriptionOperation byte

const (
	// SubscriptionAdded is the resulting subscription operation from AddSubscriptionAsync
	SubscriptionAdded SubscriptionOperation = iota
	// SubscriptionRemoved is the resulting subscription operation from RemoveSubscription
	SubscriptionRemoved
)

// SubscriptionChangeListener is a callback that can be set on async subscription operations
// that allows for handling of success or failure. The callback will be passed the subscription
// in question, the operation (either SubscriptionAdded or SubscriptionRemoved), and the error
// or nil if no error was thrown while adding the subscription.
type SubscriptionChangeListener func(subscription resource.Subscription, operation SubscriptionOperation, errOrNil error)

// MessageHandler is a callback that can be registered to receive messages asynchronously.
type MessageHandler func(inboundMessage message.InboundMessage)
