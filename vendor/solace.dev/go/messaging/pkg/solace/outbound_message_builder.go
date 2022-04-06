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
	"solace.dev/go/messaging/pkg/solace/message/sdt"
)

// OutboundMessageBuilder allows construction of messages to be sent.
type OutboundMessageBuilder interface {
	// Build creates an OutboundMessage instance based on the configured properties.
	// Accepts additional configuration providers to apply only to the built message, with the
	// last in the list taking precedence.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	Build(additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (message.OutboundMessage, error)
	// BuildWithByteArrayPayload creates a message with a byte array payload.
	// Accepts additional configuration providers to apply only to the built message, with the
	// last in the list taking precedence.
	// Returns the built message, otherwise an error if one occurred.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	BuildWithByteArrayPayload(payload []byte, additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (message message.OutboundMessage, err error)
	// BuildWithStringPayload builds a new message with a string payload.
	// Accepts additional configuration providers to apply only to the built message, with the
	// last in the list taking precedence.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	BuildWithStringPayload(payload string, additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (message message.OutboundMessage, err error)
	// BuildWithMapPayload builds a new message with a SDTMap payload.
	// Accepts additional configuration providers to apply only to the built message, with the
	// last in the list taking precedence.
	// If invalid data, ie. data not allowed as SDTData, is found in the
	// map, this function will return a nil OutboundMessage and an error.
	// Returns a solace/errors.*IllegalArgumentError if an invalid payload is specified.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	BuildWithMapPayload(payload sdt.Map, additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (message message.OutboundMessage, err error)
	// BuildWithStreamPayload builds a new message with a SDTStream payload.
	// Accepts additional configuration providers to apply only to the built message, with the
	// last in the list taking precedence.
	// If invalid data, ie. data not allowed as SDTData, is found in the
	// stream, this function returns a nil OutboundMessage and an error.
	// Returns a solace/errors.*IllegalArgumentError if an invalid payload is specified.
	// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
	BuildWithStreamPayload(payload sdt.Stream, additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (message message.OutboundMessage, err error)
	// FromConfigurationProvider sets the given message properties to the resulting message.
	// Both Solace defined config.MessageProperty keys as well as arbitrary user-defined
	// property keys are accepted. If using custom defined properties, the date type can be
	// any of sdt.Data supported types.
	FromConfigurationProvider(properties config.MessagePropertiesConfigurationProvider) OutboundMessageBuilder
	// WithProperty sets an individual message property on the resulting message.
	// Both Solace defined config.MessageProperty keys as well as arbitrary user-defined
	// property keys are accepted. If using custom defined properties, the date type can be
	// any of sdt.Data supported types.
	WithProperty(propertyName config.MessageProperty, propertyValue interface{}) OutboundMessageBuilder
	// WithExpiration sets the message expiration time to the given time.
	WithExpiration(t time.Time) OutboundMessageBuilder
	// WithHTTPContentHeader sets the specified HTTP content-header on the message.
	WithHTTPContentHeader(contentType, contentEncoding string) OutboundMessageBuilder
	// WithPriority sets the priority of the message, where the priority is a value between 0 (lowest) and 255 (highest).
	WithPriority(priority int) OutboundMessageBuilder
	// WithApplicationMessageId sets the application message ID of the message. It is carried in the message metadata
	// and is used for application to application signaling.
	WithApplicationMessageID(messageID string) OutboundMessageBuilder
	// WithApplicationMessageType sets the application message type for a message. It is carried in the message metadata
	// and is used for application to application signaling.
	WithApplicationMessageType(messageType string) OutboundMessageBuilder
	// WithSequenceNumber sets the sequence number for the message. The sequence number is carried in the message metadata
	// and is used for application to application signaling.
	WithSequenceNumber(sequenceNumber uint64) OutboundMessageBuilder
	// WithSenderID sets the sender ID for a message from a string. If config.ServicePropertyGenerateSenderID is enabled on
	// the messaging service, then passing a string to this method will override the API generated sender ID.
	WithSenderID(senderID string) OutboundMessageBuilder
	// WithCorrelationID sets the correlation ID for the message. The correlation ID is user-defined and carried end-to-end.
	// It can be matched in a selector, but otherwise is not relevant to the event broker. The correlation ID may be used
	// for peer-to-peer message synchronization. In JMS applications, this field is carried as the JMSCorrelationID Message
	// Header Field.
	WithCorrelationID(correlationID string) OutboundMessageBuilder
}
