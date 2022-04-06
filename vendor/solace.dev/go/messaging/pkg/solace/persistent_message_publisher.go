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

// PersistentMessagePublisher allows for the publishing of persistent messages (guaranteed messages).
type PersistentMessagePublisher interface {
	MessagePublisher
	MessagePublisherHealthCheck

	// StartAsyncCallback starts the PersistentMessagePublisher asynchronously.
	// Calls the callback when started with an error if one occurred, otherwise nil
	// when successful.
	StartAsyncCallback(callback func(PersistentMessagePublisher, error))

	// SetPublishReceiptListener sets the listener to receive delivery receipts.
	// PublishReceipt events are triggered once the API receives an acknowledgement
	// when a message is received from the event broker.
	// This should be set before the publisher is started to avoid dropping acknowledgements.
	// The listener does not receive events from PublishAwaitAcknowledgement calls.
	SetMessagePublishReceiptListener(listener MessagePublishReceiptListener)

	// TerminateAsyncCallback terminates the PersistentMessagePublisher asynchronously.
	// Calls the callback when terminated with nil if successful, otherwise an error if
	// one occurred. When gracePeriod is a value less than 0, the function waits indefinitely.
	TerminateAsyncCallback(gracePeriod time.Duration, callback func(error))

	// PublishBytes sends a message of type byte array to the specified destination.
	// Returns an error if one occurred while attempting to publish or if the publisher
	// is not started/terminated. Possible errors include:
	// - solace/errors.*PubSubPlusClientError - If the message could not be sent and all retry attempts failed.
	// - solace/errors.*PublisherOverflowError - If messages are published faster than publisher's I/O
	//   capabilities allow. When publishing can be resumed, the registered PublisherReadinessListeners
	//   are called.
	PublishBytes(message []byte, destination *resource.Topic) error

	// PublishString sends a message of type string to the specified destination.
	// Possible errors include:
	// - solace/errors.*PubSubPlusClientError - If the message could not be sent and all retry attempts failed.
	// - solace/errors.*PublisherOverflowError - If messages are published faster than publisher's I/O
	//   capabilities allow. When publishing can be resumed, the registered PublisherReadinessListeners
	//   are called.
	PublishString(message string, destination *resource.Topic) error

	// Publish sends the specified  message of type OutboundMessage built by a
	// OutboundMessageBuilder to the specified destination.
	// Optionally, you can provide properties in the form of OutboundMessageProperties to override
	// any properties set on OutboundMessage. The properties argument can be nil to
	// not set any properties.
	// Optionally, provide a context that is available in the PublishReceiptListener
	// registered with SetMessagePublishReceiptListener as GetUserContext.
	// The context argument can be nil to not set a context. Possible errors include:
	// - solace/errors.*PubSubPlusClientError - If the message could not be sent and all retry attempts failed.
	// - solace/errors.*PublisherOverflowError - If messages are published faster than publisher's I/O
	//   capabilities allow. When publishing can be resumed, the registered PublisherReadinessListeners
	//   are called.
	Publish(message message.OutboundMessage, destination *resource.Topic, properties config.MessagePropertiesConfigurationProvider, context interface{}) error

	// PublishAwaitAcknowledgement sends the specified message of type OutboundMessage
	// and awaits a publish acknowledgement.
	// Optionally, you can provide properties in the form of OutboundMessageProperties to override
	// any properties set on OutboundMessage. The properties argument can be nil to
	// not set any properties.
	// If the specified timeout argument is less than 0, the function waits indefinitely.
	// Possible errors include:
	// - solace/errors.*PubSubPlusClientError - If the message could not be sent and all retry attempts failed.
	// - solace/errors.*PublisherOverflowError - If messages are published faster than publisher's I/O
	//   capabilities allow. When publishing can be resumed, the registered PublisherReadinessListeners
	//   are called.
	PublishAwaitAcknowledgement(message message.OutboundMessage, destination *resource.Topic, timeout time.Duration, properties config.MessagePropertiesConfigurationProvider) error
}

// MessagePublishReceiptListener is a listener that can be registered for the delivery receipt events.
type MessagePublishReceiptListener func(PublishReceipt)

// PublishReceipt is the receipt for delivery of a persistent message.
type PublishReceipt interface {
	// GetUserContext retrieves the context associated with the publish, if provided.
	// Returns nil if no user context is set.
	GetUserContext() interface{}

	// GetTimeStamp retrieves the time. The time indicates when the event occurred, specifically the time when the
	// acknowledgement was received by the API from the event  broker, or if present,  when the GetError error
	// occurred.
	GetTimeStamp() time.Time

	// GetMessage returns an OutboundMessage that was successfully published.
	GetMessage() message.OutboundMessage

	// GetError retrieves an error if one occurred, which usually indicates a publish attempt failed.
	// GetError returns nil on a successful publish, otherwise an error if a failure occurred
	// while delivering the message.
	GetError() error

	// IsPersisted returns true if the event broker confirmed that the message was successfully received and persisted,
	// otherwise false.
	IsPersisted() bool
}

// PersistentMessagePublisherBuilder allows for configuration of persistent message publisher instances.
type PersistentMessagePublisherBuilder interface {
	// Build returns a new PersistentMessagePublisher based on the configured properties.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	Build() (messagePublisher PersistentMessagePublisher, err error)
	// OnBackPressureReject sets the publisher back pressure strategy to reject
	// where the publish attempts are rejected once the bufferSize (the number of messages), is reached.
	// If bufferSize is 0, an error is thrown when the transport is full when attempting to publish.
	// A buffer of the given size will be statically allocated when the publisher is built.
	// Valid bufferSize is greater than or equal to  0.
	OnBackPressureReject(bufferSize uint) PersistentMessagePublisherBuilder
	// OnBackPressureWait sets the publisher back pressure strategy to wait where publish
	// attempts block until there is space in the buffer of size bufferSize (the number of messages).
	// A buffer of the given size will be statically allocated when the publisher is built.
	// Valid bufferSize is greater than or equal to 1.
	OnBackPressureWait(bufferSize uint) PersistentMessagePublisherBuilder
	// FromConfigurationProvider configures the persistent publisher with the given properties.
	// Built in PublisherPropertiesConfigurationProvider implementations include:
	// - PublisherPropertyMap - A  map of PublisherProperty keys to values.
	FromConfigurationProvider(provider config.PublisherPropertiesConfigurationProvider) PersistentMessagePublisherBuilder
}
