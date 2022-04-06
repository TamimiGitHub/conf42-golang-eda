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

// The DirectMessagePublisher interface is used to publish direct messages.
type DirectMessagePublisher interface {
	MessagePublisher
	MessagePublisherHealthCheck

	// StartAsyncCallback starts the DirectMessagePublisher asynchronously.
	// Calls the specified callback when started with an error if one occurred, otherwise nil
	// if successful.
	StartAsyncCallback(callback func(DirectMessagePublisher, error))

	// SetPublishFailureListener sets the listener to call if the publishing of
	// a direct message fails.
	SetPublishFailureListener(listener PublishFailureListener)

	// TerminateAsyncCallback terminates the DirectMessagePublisher asynchronously.
	// Calls the callback when terminated with nil if successful, or an error if
	// one occurred. If gracePeriod is less than 0, this function waits indefinitely.
	TerminateAsyncCallback(gracePeriod time.Duration, callback func(error))

	// PublishBytes publishes a message of type byte array to the specified destination.
	// Returns an error if one occurred while attempting to publish, or if the publisher
	// is not started/terminated. Possible errors include:
	// - solace/errors.*PubSubPlusClientError - If the message could not be sent and all retry attempts failed.
	// - solace/errors.*PublisherOverflowError - If messages are published faster than the publisher's I/O
	//   capabilities allow. When publishing can be resumed, the registered
	//   PublisherReadinessListeners are called.
	PublishBytes(message []byte, destination *resource.Topic) error

	// PublishString publishes a message of type string to the specified destination.
	// Returns an error if one occurred. Possible errors include:
	// - solace/errors.*PubSubPlusClientError - If the message could not be sent and all retry attempts failed.
	// - solace/errors.*PublisherOverflowError - If messages are published faster than the publisher's I/O
	//   capabilities allow. When publishing can be resumed, the registered PublisherReadinessListeners are called.
	PublishString(message string, destination *resource.Topic) error

	// Publish publishes the specified message of type OutboundMessage built by a
	// OutboundMessageBuilder to the specified destination. Possible errors include:
	// - solace/errors.*PubSubPlusClientError - If the message could not be sent and all retry attempts failed.
	// - solace/errors.*PublisherOverflowError - If messages are published faster than the publisher's I/O
	//   capabilities allow. When publishing can be resumed, the registered PublisherReadinessListeners are called.
	Publish(message message.OutboundMessage, destination *resource.Topic) error

	// PublishWithProperties publishes the specified message of type OutboundMessage
	// with the specified properties. These properties override the properties on
	// the OutboundMessage instance if it is present. Possible errors include:
	// - solace/errors.*PubSubPlusClientError - If the message could not be sent and all retry attempts failed.
	// - solace/errors.*PublisherOverflowError - If messages are published faster than the publisher's I/O
	//   capabilities allow. When publishing can be resumed, the registered  PublisherReadinessListeners are called.
	PublishWithProperties(message message.OutboundMessage, destination *resource.Topic, properties config.MessagePropertiesConfigurationProvider) error
}

// PublishFailureListener is a listener that can be registered for publish failure events.
type PublishFailureListener func(FailedPublishEvent)

// FailedPublishEvent represents an event thrown when publishing a direct message fails.
type FailedPublishEvent interface {
	// GetMessage retrieves the message that was not delivered
	GetMessage() message.OutboundMessage
	// GetDestination retrieves the destination that the message was published to.
	GetDestination() resource.Destination
	// GetTimeStamp retrieves the timestamp of the error.
	GetTimeStamp() time.Time
	// GetError retrieves the error that failed the publish attempt.
	GetError() error
}

// DirectMessagePublisherBuilder allows for configuration of direct message publisher instances.
type DirectMessagePublisherBuilder interface {
	
    // Build creates a new DirectMessagePublisher instance based on the configured properties.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	Build() (messagePublisher DirectMessagePublisher, err error)
	
	// OnBackPressureReject sets the publisher back pressure strategy to reject
	// where publish attempts will be rejected once the bufferSize, in number of messages, is reached.
	// If bufferSize is 0, an error will be thrown when the transport is full when publishing.
	// A buffer of the given size will be statically allocated when the publisher is built.
	// Valid bufferSize is >= 0.
	OnBackPressureReject(bufferSize uint) DirectMessagePublisherBuilder
	
	// OnBackPressureWait sets the publisher back pressure strategy to wait where publish
	// attempts may block until there is space in the buffer of size bufferSize in number of messages.
	// A buffer of the given size will be statically allocated when the publisher is built.
	// Valid bufferSize is >= 1.
	OnBackPressureWait(bufferSize uint) DirectMessagePublisherBuilder
	
	// FromConfigurationProvider configures the direct publisher with the specified properties.
	// The built-in PublisherPropertiesConfigurationProvider implementations include:
	// - PublisherPropertyMap - A map of PublisherProperty keys to values.
	FromConfigurationProvider(provider config.PublisherPropertiesConfigurationProvider) DirectMessagePublisherBuilder
}
