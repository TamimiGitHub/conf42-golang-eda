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

package config

// MessagePropertiesConfigurationProvider describes the behavior of a configuration provider
// that provides properties for messages.
type MessagePropertiesConfigurationProvider interface {
	// GetConfiguration returns a copy of the configuration of the MessagePropertiesConfigurationProvider
	// as a MessagePropertyMap.
	GetConfiguration() MessagePropertyMap
}

// MessageProperty is a property that can be set on a messages.
type MessageProperty string

// MessagePropertyMap is a map of MessageProperty keys to values.
type MessagePropertyMap map[MessageProperty]interface{}

// GetConfiguration returns a copy of the MessagePropertyMap
func (messagePropertyMap MessagePropertyMap) GetConfiguration() MessagePropertyMap {
	ret := make(MessagePropertyMap)
	for key, value := range messagePropertyMap {
		ret[key] = value
	}
	return ret
}

// MarshalJSON implements the json.Marshaler interface.
func (messagePropertyMap MessagePropertyMap) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	for k, v := range messagePropertyMap {
		m[string(k)] = v
	}
	return nestJSON(m)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (messagePropertyMap MessagePropertyMap) UnmarshalJSON(b []byte) error {
	m, err := flattenJSON(b)
	if err != nil {
		return err
	}
	for key, val := range m {
		messagePropertyMap[MessageProperty(key)] = val
	}
	return nil
}

// These constants are used as keys in a MessagePropertyMap to configure the message.
const (
	// MessagePropertyApplicationMessageType specifies an application message type. This value is only used by applications,
	// and is passed through this API unmodified.
	MessagePropertyApplicationMessageType MessageProperty = "solace.messaging.message.application-message-type"

	// MessagePropertyApplicationMessageID specifies an application-specific message identifier. This value is used by
	// applications only, and is passed through the API unmodified.
	MessagePropertyApplicationMessageID MessageProperty = "solace.messaging.message.application-message-id"

	// MessagePropertyElidingEligible specifies whether the message is eligible for eliding.
	MessagePropertyElidingEligible MessageProperty = "solace.messaging.message.eliding-eligible"

	// MessagePropertyPriority specifies an optional message priority.
	// The valid priority value range is 0-255 with 0 as the lowest priority and 255 as the
	// highest. A value of -1 indicates the priority is not set and that a default priority value is used
	// instead.
	MessagePropertyPriority MessageProperty = "solace.messaging.message.priority"

	// MessagePropertyHTTPContentType specifies the HTTP content-type header value for interaction with an HTTP client.
	// Accepted values Defined in RFC7231, Section-3.1.2.2.
	MessagePropertyHTTPContentType MessageProperty = "solace.messaging.message.http-content"

	// MessagePropertyHTTPContentEncoding specifies the HTTP content-type encoding value for interaction with an HTTP client.
	// Accepted values Defined in rfc2616 section-14.11
	MessagePropertyHTTPContentEncoding MessageProperty = "solace.messaging.message.http-encoding"

	// MessagePropertyCorrelationID key specifies a correlation ID. The correlation ID is used for correlating a request
	// to a reply and should be as random as possible.
	// This variable is being also used for the Request-Reply API. For this reason, it is not recommended
	// to use this property on a message builder instance but only on a publisher interface
	// where it's available.
	MessagePropertyCorrelationID MessageProperty = "solace.messaging.message.correlation-id"

	// MessagePropertyPersistentTimeToLive specifies the number of milliseconds before the message is discarded or moved to a
	// Dead Message Queue. The default value is 0, which means the message never expires.
	// This key is valid only for persistent messages
	MessagePropertyPersistentTimeToLive MessageProperty = "solace.messaging.message.persistent.time-to-live"

	// MessagePropertyPersistentExpiration specifies the UTC time (in milliseconds, from midnight, January 1, 1970 UTC)
	// and indicates when the message is supposed to expire. Setting this property has no effect if the TimeToLive
	// is set in the same message. It is carried to clients that receive the message, unmodified, and
	// does not effect the life cycle of the message. This property is only valid for persistent messages.
	MessagePropertyPersistentExpiration MessageProperty = "solace.messaging.message.persistent.expiration"

	// MessagePropertyPersistentDMQEligible specifies if a message is eligible to be moved to a Dead Message Queue.
	// The default value is true. This property is valid only for persistent messages.
	MessagePropertyPersistentDMQEligible MessageProperty = "solace.messaging.message.persistent.dmq-eligible"

	// MessagePropertyPersistentAckImmediately specifies if an appliance should ACK this message immediately upon message receipt.
	// The default value is false. This property is valid only for persistent messages
	MessagePropertyPersistentAckImmediately MessageProperty = "solace.messaging.message.persistent.ack-immediately"

	// MessagePropertySequenceNumber specifies the sequence number of the message. If sequence number generation is
	// enabled, this key overrides the generated value. This field can be set
	// automatically during message publishing, but existing values are not overwritten when set, as
	// when a message is sent multiple times.
	MessagePropertySequenceNumber MessageProperty = "solace.messaging.message.persistent.sequence-number"

	// MessagePropertyClassOfService specifies the class of service to set on the message.
	// Valid class of service values are
	//  0 | COS_1
	//  1 | COS_2
	//  2 | COS_3
	MessagePropertyClassOfService MessageProperty = "solace.messaging.message.class-of-service"

	// MessagePropertySenderID specifies a custom sender ID in the message header.
	// The given value is converted to a string. If set to nil, the sender ID is not set,
	// however a sender ID is still generated if ServicePropertyGenerateSenderID is enabled.
	MessagePropertySenderID MessageProperty = "solace.messaging.message.sender-id"
)
