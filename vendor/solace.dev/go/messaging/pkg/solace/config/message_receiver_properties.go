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

// ReceiverPropertiesConfigurationProvider describes the behavior of a configuration provider
// that provides properties for receivers.
type ReceiverPropertiesConfigurationProvider interface {
	GetConfiguration() ReceiverPropertyMap
}

// ReceiverProperty is a property that can be set on a receiver.
type ReceiverProperty string

// ReceiverPropertyMap is a map of ReceiverProperty keys to values.
type ReceiverPropertyMap map[ReceiverProperty]interface{}

// GetConfiguration returns a copy of the ReceiverPropertyMap.
func (receiverPropertyMap ReceiverPropertyMap) GetConfiguration() ReceiverPropertyMap {
	ret := make(ReceiverPropertyMap)
	for key, value := range receiverPropertyMap {
		ret[key] = value
	}
	return ret
}

// MarshalJSON implements the json.Marshaler interface.
func (receiverPropertyMap ReceiverPropertyMap) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	for k, v := range receiverPropertyMap {
		m[string(k)] = v
	}
	return nestJSON(m)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (receiverPropertyMap ReceiverPropertyMap) UnmarshalJSON(b []byte) error {
	m, err := flattenJSON(b)
	if err != nil {
		return err
	}
	for key, val := range m {
		receiverPropertyMap[ReceiverProperty(key)] = val
	}
	return nil
}

const (
	// ReceiverPropertyDirectBackPressureStrategy defines a direct receiver back pressure strategy.
	// Valid values are BUFFER_DROP_LATEST_WHEN_FULL where the latest incoming message is dropped
	// or BUFFER_DROP_OLDEST_WHEN_FULL where the oldest undelivered message is dropped.
	ReceiverPropertyDirectBackPressureStrategy ReceiverProperty = "solace.messaging.receiver.direct.back-pressure.strategy"

	// ReceiverPropertyDirectBackPressureBufferCapacity defines the direct receiver back pressure buffer capacity
	// measured in messages. This property only has effect in conjunction with the back pressure strategy.
	ReceiverPropertyDirectBackPressureBufferCapacity ReceiverProperty = "solace.messaging.receiver.direct.back-pressure.buffer-capacity"

	// ReceiverPropertyPersistentMissingResourceCreationStrategy specifies if and how missing remote resource (such as queues) are to be created
	// on a broker prior to receiving persistent messages. Valid values are of type MissingResourceCreationStrategy, either
	// MissingResourceDoNotCreate or MissingResourceCreateOnStart.
	ReceiverPropertyPersistentMissingResourceCreationStrategy ReceiverProperty = "solace.messaging.receiver.persistent.missing-resource-creation-strategy"

	// ReceiverPropertyPersistentMessageSelectorQuery specifies the message-selection query based on the message header parameter
	// and message properties values. When a selector is applied then the receiver receives only
	// messages whose headers and properties match the selector. A message selector cannot select
	// messages on the basis of the content of the message body.
	ReceiverPropertyPersistentMessageSelectorQuery ReceiverProperty = "solace.messaging.receiver.persistent.selector-query"

	// ReceiverPropertyPersistentStateChangeListener specifies to use the callback ReceiverStateChangeListener and enable
	// activation and passivation support.
	ReceiverPropertyPersistentStateChangeListener ReceiverProperty = "solace.messaging.receiver.persistent.state-change-listener"

	// ReceiverPropertyPersistentMessageAckStrategy specifies the acknowledgement strategy for the message receiver.
	ReceiverPropertyPersistentMessageAckStrategy ReceiverProperty = "solace.messaging.receiver.persistent.ack.strategy"

	// ReceiverPropertyPersistentMessageReplayStrategy enables message replay and to specify a replay strategy.
	ReceiverPropertyPersistentMessageReplayStrategy ReceiverProperty = "solace.messaging.receiver.persistent.replay.strategy"

	// ReceiverPropertyPersistentMessageReplayStrategyTimeBasedStartTime configures time-based replay strategy with a start time.
	ReceiverPropertyPersistentMessageReplayStrategyTimeBasedStartTime ReceiverProperty = "solace.messaging.receiver.persistent.replay.timebased-start-time"

	// ReceiverPropertyPersistentMessageReplayStrategyIDBasedReplicationGroupMessageID configures the ID based replay strategy with a specified  replication group message ID.
	ReceiverPropertyPersistentMessageReplayStrategyIDBasedReplicationGroupMessageID ReceiverProperty = "solace.messaging.receiver.persistent.replay.replication-group-message-id"
)
