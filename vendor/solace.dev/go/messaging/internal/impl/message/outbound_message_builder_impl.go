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

package message

import (
	"fmt"
	"time"

	"solace.dev/go/messaging/internal/ccsmp"

	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/internal/impl/validation"

	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/message/sdt"
	"solace.dev/go/messaging/pkg/solace/resource"
)

// OutboundMessageBuilderImpl structure
type OutboundMessageBuilderImpl struct {
	properties config.MessagePropertyMap
}

// NewOutboundMessageBuilder function
func NewOutboundMessageBuilder() solace.OutboundMessageBuilder {
	return &OutboundMessageBuilderImpl{make(config.MessagePropertyMap)}
}

// Build method
func (builder *OutboundMessageBuilderImpl) Build(additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (message.OutboundMessage, error) {
	return builder.build(additionalConfiguration...)
}

func (builder *OutboundMessageBuilderImpl) build(additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (*OutboundMessageImpl, error) {
	msg, err := NewOutboundMessage()
	if err != nil {
		return nil, err
	}

	properties := builder.properties.GetConfiguration()
	for _, additionalConfig := range additionalConfiguration {
		mergeMessagePropertyMap(properties, additionalConfig)
	}

	// Override defaults to match documented edfaults
	// SOL-67545: DMQ Eligible should default to true when not configured
	if _, ok := properties[config.MessagePropertyPersistentDMQEligible]; !ok {
		properties[config.MessagePropertyPersistentDMQEligible] = true
	}

	err = SetProperties(msg, properties)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// SetProperties function
func SetProperties(message *OutboundMessageImpl, properties config.MessagePropertyMap) error {
	userProperties := sdt.Map{}
	for property, value := range properties {
		var setterErrorInfo core.ErrorInfo
		switch property {
		case config.MessagePropertyPersistentExpiration:
			if t, ok := value.(time.Time); ok {
				value = t.UnixNano() / int64(time.Millisecond)
			}
			propValue, present, err := validation.Int64PropertyValidation(string(config.MessagePropertyPersistentExpiration), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetExpiration(message.messagePointer, propValue)
			}

		case config.MessagePropertyPriority:
			propValue, present, err := validation.IntegerPropertyValidationWithRange(string(config.MessagePropertyPriority), value, 0, 255)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetPriority(message.messagePointer, propValue)
			}

		case config.MessagePropertyHTTPContentType:
			propValue, present, err := validation.StringPropertyValidation(string(config.MessagePropertyHTTPContentType), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetHTTPContentType(message.messagePointer, propValue)
			}

		case config.MessagePropertyHTTPContentEncoding:
			propValue, present, err := validation.StringPropertyValidation(string(config.MessagePropertyHTTPContentEncoding), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetHTTPContentEncoding(message.messagePointer, propValue)
			}

		case config.MessagePropertyApplicationMessageType:
			propValue, present, err := validation.StringPropertyValidation(string(config.MessagePropertyApplicationMessageType), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetApplicationMessageType(message.messagePointer, propValue)
			}

		case config.MessagePropertyApplicationMessageID:
			propValue, present, err := validation.StringPropertyValidation(string(config.MessagePropertyApplicationMessageID), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetApplicationMessageID(message.messagePointer, propValue)
			}

		case config.MessagePropertyElidingEligible:
			propValue, present, err := validation.BooleanPropertyValidation(string(config.MessagePropertyElidingEligible), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetElidingEligible(message.messagePointer, propValue)
			}

		case config.MessagePropertyCorrelationID:
			propValue, present, err := validation.StringPropertyValidation(string(config.MessagePropertyCorrelationID), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetCorrelationID(message.messagePointer, propValue)
			}

		case config.MessagePropertyPersistentTimeToLive:
			propValue, present, err := validation.Int64PropertyValidation(string(config.MessagePropertyPersistentTimeToLive), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetTimeToLive(message.messagePointer, propValue)
			}

		case config.MessagePropertyPersistentDMQEligible:
			propValue, present, err := validation.BooleanPropertyValidation(string(config.MessagePropertyPersistentDMQEligible), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetDMQEligible(message.messagePointer, propValue)
			}

		case config.MessagePropertyPersistentAckImmediately:
			propValue, present, err := validation.BooleanPropertyValidation(string(config.MessagePropertyPersistentDMQEligible), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetAckImmediately(message.messagePointer, propValue)
			}

		case config.MessagePropertySequenceNumber:
			propValue, present, err := validation.Uint64PropertyValidation(string(config.MessagePropertySequenceNumber), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetSequenceNumber(message.messagePointer, propValue)
			}
		case config.MessagePropertyClassOfService:
			propValue, present, err := validation.IntegerPropertyValidation(string(config.MessagePropertyClassOfService), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetClassOfService(message.messagePointer, propValue)
			}
		case config.MessagePropertySenderID:
			propValue, present, err := validation.StringPropertyValidation(string(config.MessagePropertySenderID), value)
			if present {
				if err != nil {
					return err
				}
				setterErrorInfo = ccsmp.SolClientMessageSetSenderID(message.messagePointer, propValue)
			}
		default:
			// If we don't recognize the property, then set it as a user property
			userProperties[string(property)] = value
		}

		if setterErrorInfo != nil {
			return core.ToNativeError(setterErrorInfo, "encountered error while allocating message: ")
		}
	}
	// set user properties
	if len(userProperties) > 0 {
		container, errInfo := ccsmp.SolClientMessageGetUserPropertyMap(message.messagePointer)
		if errInfo != nil {
			if errInfo.ReturnCode == ccsmp.SolClientReturnCodeNotFound {
				container, errInfo = ccsmp.SolClientMessageCreateUserPropertyMap(message.messagePointer)
				if errInfo != nil {
					return core.ToNativeError(errInfo)
				}
			} else {
				return core.ToNativeError(errInfo)
			}
		}
		// try and convert properties to an SDT Map
		if err := sdtMapToContainer(container, userProperties); err != nil {
			return err
		}
		errInfo = container.SolClientContainerClose()
		if errInfo != nil {
			return core.ToNativeError(errInfo)
		}
	}
	return nil
}

// BuildWithByteArrayPayload builds a new message with a byte array payload.
// Returns the built message or an error if one occurred.
// Returns solace/solace.*InvalidConfigurationError if an invalid configuration is provided.
func (builder *OutboundMessageBuilderImpl) BuildWithByteArrayPayload(payload []byte, additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (message.OutboundMessage, error) {
	msg, err := builder.build(additionalConfiguration...)
	if err != nil {
		return nil, err
	}
	errorInfo := ccsmp.SolClientMessageSetBinaryAttachmentAsBytes(msg.messagePointer, payload)
	if errorInfo != nil {
		return nil, core.ToNativeError(errorInfo, "encountered error while allocating message: ")
	}
	return msg, nil
}

// BuildWithStringPayload builds a new message with a string payload.
// Returns solace/solace.*InvalidConfigurationError if an invalid configuration is provided.
func (builder *OutboundMessageBuilderImpl) BuildWithStringPayload(payload string, additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (message.OutboundMessage, error) {
	msg, err := builder.build(additionalConfiguration...)
	if err != nil {
		return nil, err
	}
	errorInfo := ccsmp.SolClientMessageSetBinaryAttachmentString(msg.messagePointer, payload)
	if errorInfo != nil {
		return nil, core.ToNativeError(errorInfo, "encountered error while allocating message: ")
	}
	return msg, nil
}

// BuildWithMapPayload builds a new message with a SDTMap payload.
// If invalid data, ie. data not allowed as SDTData, is found in the
// map, this function will return a nil OutboundMessage and an error.
// Returns a solace/solace.*IllegalArgumentError if an invalid payload is given.
// Returns solace/solace.*InvalidConfigurationError if an invalid configuration is provided.
func (builder *OutboundMessageBuilderImpl) BuildWithMapPayload(payload sdt.Map, additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (message.OutboundMessage, error) {
	msg, err := builder.build(additionalConfiguration...)
	if err != nil {
		return nil, err
	}
	container, errorInfo := ccsmp.SolClientMessageCreateBinaryAttachmentMap(msg.messagePointer)
	if errorInfo != nil {
		return nil, core.ToNativeError(errorInfo)
	}
	err = sdtMapToContainer(container, payload)
	if err != nil {
		return nil, err
	}
	errorInfo = container.SolClientContainerClose()
	if errorInfo != nil {
		return nil, core.ToNativeError(errorInfo, "encountered error while allocating message: ")
	}
	return msg, nil
}

// BuildWithStreamPayload builds a new message with a SDTStream payload.
// If invalid data, ie. data not allowed as SDTData, is found in the
// stream, this function will return a nil OutboundMessage and an error.
// Returns a solace/solace.*IllegalArgumentError if an invalid payload is given.
// Returns solace/solace.*InvalidConfigurationError if an invalid configuration is provided.
func (builder *OutboundMessageBuilderImpl) BuildWithStreamPayload(payload sdt.Stream, additionalConfiguration ...config.MessagePropertiesConfigurationProvider) (message.OutboundMessage, error) {
	msg, err := builder.build(additionalConfiguration...)
	if err != nil {
		return nil, err
	}
	container, errorInfo := ccsmp.SolClientMessageCreateBinaryAttachmentStream(msg.messagePointer)
	if errorInfo != nil {
		return nil, core.ToNativeError(errorInfo)
	}
	err = sdtStreamToContainer(container, payload)
	if err != nil {
		return nil, err
	}
	errorInfo = container.SolClientContainerClose()
	if errorInfo != nil {
		return nil, core.ToNativeError(errorInfo, "encountered error while allocating message: ")
	}
	return msg, nil
}

// FromConfigurationProvider will set the given properties to the resulting message.
func (builder *OutboundMessageBuilderImpl) FromConfigurationProvider(properties config.MessagePropertiesConfigurationProvider) solace.OutboundMessageBuilder {
	mergeMessagePropertyMap(builder.properties, properties)
	return builder
}

// WithProperty sets an individual property on a message.
func (builder *OutboundMessageBuilderImpl) WithProperty(propertyName config.MessageProperty, propertyValue interface{}) solace.OutboundMessageBuilder {
	builder.properties[config.MessageProperty(propertyName)] = propertyValue
	return builder
}

// WithExpiration will set the message expiration time to the given time.
func (builder *OutboundMessageBuilderImpl) WithExpiration(t time.Time) solace.OutboundMessageBuilder {
	timeMs := int64(t.Unix() * 1000)
	builder.properties[config.MessagePropertyPersistentExpiration] = timeMs
	return builder
}

// WithHTTPContentHeader sets the desired http content header on the message.
func (builder *OutboundMessageBuilderImpl) WithHTTPContentHeader(key string, value string) solace.OutboundMessageBuilder {
	builder.properties[config.MessagePropertyHTTPContentType] = key
	builder.properties[config.MessagePropertyHTTPContentEncoding] = value
	return builder
}

// WithPriority sets the priority of the message where the priority is a value between 0 and 255.
func (builder *OutboundMessageBuilderImpl) WithPriority(priority int) solace.OutboundMessageBuilder {
	builder.properties[config.MessagePropertyPriority] = priority
	return builder
}

// WithApplicationMessageID sets the application message ID of the message.
func (builder *OutboundMessageBuilderImpl) WithApplicationMessageID(messageID string) solace.OutboundMessageBuilder {
	builder.properties[config.MessagePropertyApplicationMessageID] = messageID
	return builder
}

// WithApplicationMessageType sets the application message type for a message.
func (builder *OutboundMessageBuilderImpl) WithApplicationMessageType(messageType string) solace.OutboundMessageBuilder {
	builder.properties[config.MessagePropertyApplicationMessageType] = messageType
	return builder
}

// WithSequenceNumber sets the sequence number for the message
func (builder *OutboundMessageBuilderImpl) WithSequenceNumber(sequenceNumber uint64) solace.OutboundMessageBuilder {
	builder.properties[config.MessagePropertySequenceNumber] = sequenceNumber
	return builder
}

// WithSenderID sets the sender ID for a message from a string.
func (builder *OutboundMessageBuilderImpl) WithSenderID(senderID string) solace.OutboundMessageBuilder {
	builder.properties[config.MessagePropertySenderID] = senderID
	return builder
}

// WithCorrelationID sets the correlation ID for the message.
func (builder *OutboundMessageBuilderImpl) WithCorrelationID(correlationID string) solace.OutboundMessageBuilder {
	builder.properties[config.MessagePropertyCorrelationID] = correlationID
	return builder
}

func (builder *OutboundMessageBuilderImpl) String() string {
	return fmt.Sprintf("solace.OutboundMessageBuilder at %p", builder)
}

func mergeMessagePropertyMap(original config.MessagePropertyMap, new config.MessagePropertiesConfigurationProvider) {
	if new == nil {
		return
	}
	props := new.GetConfiguration()
	for key, value := range props {
		original[key] = value
	}
}

func sdtMapToContainer(container *ccsmp.SolClientOpaqueContainer, sdtMap sdt.Map) error {
	for key, item := range sdtMap {
		err := addItemToContainer(container, item, key)
		if err != nil {
			return err
		}
	}
	return nil
}

func sdtStreamToContainer(container *ccsmp.SolClientOpaqueContainer, sdtStream sdt.Stream) error {
	for _, item := range sdtStream {
		err := addItemToContainer(container, item, "")
		if err != nil {
			return err
		}
	}
	return nil
}

func addItemToContainer(container *ccsmp.SolClientOpaqueContainer, item sdt.Data, key string) error {
	var errInfo core.ErrorInfo
	if item == nil {
		errInfo = container.ContainerAddNil(key)
	} else {
		switch casted := item.(type) {
		case bool:
			errInfo = container.ContainerAddBoolean(casted, key)
		case int8:
			errInfo = container.ContainerAddInt8(casted, key)
		case int16:
			errInfo = container.ContainerAddInt16(casted, key)
		case int32:
			errInfo = container.ContainerAddInt32(casted, key)
		case int64:
			errInfo = container.ContainerAddInt64(casted, key)
		case int:
			errInfo = container.ContainerAddInt64(int64(casted), key)
		case uint8:
			errInfo = container.ContainerAddUint8(casted, key)
		case uint16:
			errInfo = container.ContainerAddUint16(casted, key)
		case uint32:
			errInfo = container.ContainerAddUint32(casted, key)
		case uint64:
			errInfo = container.ContainerAddUint64(casted, key)
		case uint:
			errInfo = container.ContainerAddUint64(uint64(casted), key)
		case float32:
			errInfo = container.ContainerAddFloat32(casted, key)
		case float64:
			errInfo = container.ContainerAddFloat64(casted, key)
		case string:
			errInfo = container.ContainerAddString(casted, key)
		case []byte:
			errInfo = container.ContainerAddBytes(casted, key)
		case sdt.WChar:
			errInfo = container.ContainerAddWChar(ccsmp.SolClientWChar(casted), key)
		case *resource.Topic:
			errInfo = container.ContainerAddDestination(casted.GetName(), ccsmp.SolClientContainerDestTopic, key)
		case *resource.Queue:
			errInfo = container.ContainerAddDestination(casted.GetName(), ccsmp.SolClientContainerDestQueue, key)
		case sdt.Map:
			var nestedMap *ccsmp.SolClientOpaqueContainer
			nestedMap, errInfo = container.ContainerOpenSubMap(key)
			if errInfo != nil {
				break
			}
			err := sdtMapToContainer(nestedMap, casted)
			if err != nil {
				return err
			}
			errInfo = nestedMap.SolClientContainerClose()
		case sdt.Stream:
			var nestedStream *ccsmp.SolClientOpaqueContainer
			nestedStream, errInfo = container.ContainerOpenSubStream(key)
			if errInfo != nil {
				break
			}
			err := sdtStreamToContainer(nestedStream, casted)
			if err != nil {
				return err
			}
			errInfo = nestedStream.SolClientContainerClose()
		default:
			return &sdt.IllegalTypeError{Data: casted}
		}
	}
	if errInfo != nil {
		return core.ToNativeError(errInfo)
	}
	return nil
}
