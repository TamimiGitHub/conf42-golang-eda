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
	"solace.dev/go/messaging/pkg/solace/metrics"
)

// MessagingService represents a broker that provides a messaging service.
type MessagingService interface {

	// Connect connects the messaging service.
	// This function blocks until the connection attempt is completed.
	// Returns nil if successful, otherwise an error containing failure details, which may be the following:
	// - solace/errors.*PubSubPlusClientError - If a connection error occurs.
	// - solace/errors.*IllegalStateError - If MessagingService has already been terminated.
	Connect() error

	// ConnectAsync connects the messaging service asynchronously.
	// Returns a channel that receives an event when completed.
	// Channel (chan) receives nil if successful, otherwise an error containing failure details.
	// For more information, see MessagingService.Connect.
	ConnectAsync() <-chan error

	// ConnectAsyncWithCallback connects the messaging service asynchonously.
	// When complete, the specified callback is called with nil if successful,
	// otherwise an error if not successful. In both cases, the messaging service
	// is passed as well.
	ConnectAsyncWithCallback(callback func(MessagingService, error))

	// CreateDirectMessagePublisherBuilder creates a DirectMessagePublisherBuilder
	// that can be used to configure direct message publisher instances.
	CreateDirectMessagePublisherBuilder() DirectMessagePublisherBuilder

	// CreateDirectMessageReceiverBuilder creates a DirectMessageReceiverBuilder
	// that can be used to configure direct message receiver instances.
	CreateDirectMessageReceiverBuilder() DirectMessageReceiverBuilder

	// CreatePersistentMessagePublisherBuilder creates a PersistentMessagePublisherBuilder
	// that can be used to configure persistent message publisher instances.
	CreatePersistentMessagePublisherBuilder() PersistentMessagePublisherBuilder

	// CreatePersistentMessageReceiverBuilder creates a PersistentMessageReceiverBuilder
	// that can be used to configure persistent message receiver instances.
	CreatePersistentMessageReceiverBuilder() PersistentMessageReceiverBuilder

	// MessageBuilder creates an OutboundMessageBuilder that can be
	// used to build messages to send via a message publisher.
	MessageBuilder() OutboundMessageBuilder

	// Disconnect disconnects the messaging service.
	// The messaging service must be connected to disconnect.
	// This function blocks until the disconnection attempt is completed.
	// Returns nil if successful, otherwise an error containing failure details.
	// A disconnected messaging service may not be reconnected.
	// Returns solace/errors.*IllegalStateError if it is not yet connected.
	Disconnect() error

	// DisconnectAsync disconnects the messaging service asynchronously.
	// Returns a channel (chan) that receives an event when completed.
	// The channel receives nil if successful, otherwise an error containing the failure details
	// For more information, see MessagingService.Disconnect.
	DisconnectAsync() <-chan error

	// DisconnectAsyncWithCallback disconnects the messaging service asynchronously.
	// When complete, the specified callback is called with nil if successful, otherwise
	// an error if not successful.
	DisconnectAsyncWithCallback(callback func(error))

	// IsConnected determines if the messaging service is operational and if Connect was previously
	// called successfully.
	// Returns true if the messaging service is connected to a remote destination, otherwise false.
	IsConnected() bool

	// AddReconnectionListener adds a new reconnection listener to the messaging service.
	// The reconnection listener is called when reconnection events occur.
	// Returns an identifier that can be used to remove the listener using RemoveReconnectionListener.
	AddReconnectionListener(listener ReconnectionListener) uint64

	// AddReconnectionAttemptListener adds a listener to receive reconnection-attempt notifications.
	// The reconnection listener is called when reconnection-attempt events occur.
	// Returns an identifier that can be used to remove the listener using RemoveReconnectionAttemptListener.
	AddReconnectionAttemptListener(listener ReconnectionAttemptListener) uint64

	// RemoveReconnectionListener removes a listener from the messaging service with the specified identifier.
	RemoveReconnectionListener(listenerID uint64)

	// RemoveReconnectionAttemptListener removes a listener from the messaging service with the specified identifier.
	RemoveReconnectionAttemptListener(listenerID uint64)

	// AddServiceInterruptionListener adds a listener to receive non-recoverable, service-interruption events.
	// Returns an identifier othat can be used to remove the listener using RemoveServiceInterruptionListener.
	AddServiceInterruptionListener(listener ServiceInterruptionListener) uint64

	// RemoveServiceInterruptionListener removes a service listener to receive non-recoverable,
	// service-interruption events with the specified identifier.
	RemoveServiceInterruptionListener(listenerID uint64)

	// GetApplicationID retrieves the application identifier.
	GetApplicationID() string

	// Metrics returns the metrics for this MessagingService instance.
	Metrics() metrics.APIMetrics

	// Info returns the API Info for this MessagingService instance.
	Info() metrics.APIInfo
}

// MessagingServiceBuilder is used to configure and build MessagingService instances.
type MessagingServiceBuilder interface {

	// Build creates MessagingService based on the provided configuration.
	// Returns the built MessagingService instance, otherwise nil if an error occurred.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	Build() (messagingService MessagingService, err error)

	// BuildWithApplicationID creates MessagingService based on the provided configuration
	// using the specified  application identifier as the applicationID.
	// Returns the created MessagingService instance, otherwise nil if an error occurred.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	BuildWithApplicationID(applicationID string) (messagingService MessagingService, err error)

	// FromConfigurationProvider sets the configuration based on the specified configuration provider.
	// The following are built in configuration providers:
	// - ServicePropertyMap - This can be used to set a ServiceProperty to a value programatically.
	//
	// The ServicePropertiesConfigurationProvider interface can also be implemented by a type
	// to have it act as a configuration factory by implementing the following:
	//
	//   func (type MyType) GetConfiguration() ServicePropertyMap {...}
	//
	// Any properties provided by the configuration provider are layered over top of any
	// previously set properties, including those set by specifying various strategies.
	FromConfigurationProvider(provider config.ServicePropertiesConfigurationProvider) MessagingServiceBuilder

	// WithAuthenticationStrategy configures the resulting messaging service
	// with the specified authentication configuration
	WithAuthenticationStrategy(authenticationStrategy config.AuthenticationStrategy) MessagingServiceBuilder

	// WithRetryStrategy configures the resulting messaging service
	// with the specified retry strategy
	WithConnectionRetryStrategy(retryStrategy config.RetryStrategy) MessagingServiceBuilder

	// WithMessageCompression configures the resulting messaging service
	// with the specified compression factor. The builder attempts to use
	// the specified compression-level with the provided host and port. It fails
	// to build if an an atempt is made to use compression on a non-secured and
	// non-compressed port.
	WithMessageCompression(compressionFactor int) MessagingServiceBuilder

	// WithReconnectionRetryStrategy configures the resulting messaging service
	// with the specified  reconnection strategy.
	WithReconnectionRetryStrategy(retryStrategy config.RetryStrategy) MessagingServiceBuilder

	// WithTransportSecurityStrategy configures the resulting messaging service
	// with the specified transport security strategy.
	WithTransportSecurityStrategy(transportSecurityStrategy config.TransportSecurityStrategy) MessagingServiceBuilder
}

// ReconnectionListener is a handler that can be registered to a MessagingService.
// It is called when a session was disconnected and subsequently reconnected.
type ReconnectionListener func(event ServiceEvent)

// ReconnectionAttemptListener is a handler that can be registered to a MessagingService.
// It is called when a session is disconnected and reconnection attempts have begun.
type ReconnectionAttemptListener func(event ServiceEvent)

// ServiceInterruptionListener is a handler that can be registered to a MessagingService.
// It is called when a session is disconncted and the connection is unrecoverable.
type ServiceInterruptionListener func(event ServiceEvent)

// ServiceEvent interface represents a messaging service event that applications can listen for.
type ServiceEvent interface {
	// GetTimestamp retrieves the timestamp of the event.
	GetTimestamp() time.Time
	// GetBrokerURI retrieves the URI of the broker.
	GetBrokerURI() string
	// GetMessage retrieves the message contents.
	GetMessage() string
	// GetCause retrieves the cause of the client error.
	GetCause() error
}
