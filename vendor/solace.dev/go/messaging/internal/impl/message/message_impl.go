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
	"sync/atomic"
	"time"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/internal/impl/logging"
	"solace.dev/go/messaging/pkg/solace/message/sdt"
	"solace.dev/go/messaging/pkg/solace/resource"
)

// MessageImpl structure
type MessageImpl struct {
	messagePointer ccsmp.SolClientMessagePt
	disposed       int32
}

// IsDisposed checks if the Disposable instance has been disposed by
// a call to Dispose. IsDisposeed returns true if Dispose has been called
// and false if it is still usable. Dispose may or may not have returned.
// The instance is considered unusable if IsDisposed returns true.
func (message *MessageImpl) IsDisposed() bool {
	return atomic.LoadInt32(&message.disposed) == 1
}

// GetProperties will return a map of properties where the keys are MessageProperty constants
func (message *MessageImpl) GetProperties() (propMap sdt.Map) {
	opaqueContainer, errorInfo := ccsmp.SolClientMessageGetUserPropertyMap(message.messagePointer)
	if errorInfo != nil {
		if errorInfo.ReturnCode != ccsmp.SolClientReturnCodeNotFound {
			logging.Default.Warning(fmt.Sprintf("Encountered error fetching user property map: %s, subcode: %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
		return nil
	}
	defer func() {
		errorInfo := opaqueContainer.SolClientContainerClose()
		if errorInfo != nil && logging.Default.IsDebugEnabled() {
			logging.Default.Debug(fmt.Sprintf("Encountered error while closing container: %s, errorCode %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
	}()
	return parseMap(opaqueContainer)
}

// GetProperty will return a property, and a boolean indicating if its present.
// Will return nil if not found. Property can be present and set to nil.
// propertyName is the MessageProperty to get. See MessageProperty constants for possible values.
func (message *MessageImpl) GetProperty(propertyName string) (propertyValue sdt.Data, present bool) {
	opaqueContainer, errorInfo := ccsmp.SolClientMessageGetUserPropertyMap(message.messagePointer)
	if errorInfo != nil {
		if errorInfo.ReturnCode != ccsmp.SolClientReturnCodeNotFound {
			logging.Default.Warning(fmt.Sprintf("Encountered error fetching user property map: %s, subcode: %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
		return nil, false
	}
	defer func() {
		errorInfo := opaqueContainer.SolClientContainerClose()
		if errorInfo != nil && logging.Default.IsDebugEnabled() {
			logging.Default.Debug(fmt.Sprintf("Encountered error while closing container: %s, errorCode %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
	}()
	val, ok := opaqueContainer.SolClientContainerGetField(propertyName)
	if !ok {
		return nil, false
	}
	return parseData(val)
}

// HasProperty will return whether or not a property is present in the Message.
// propertyName is the MessageProperty to get. See MessageProperty constants for possible values.
func (message *MessageImpl) HasProperty(propertyName string) bool {
	_, ok := message.GetProperty(propertyName)
	return ok
}

// GetPayloadAsBytes will attempt to get the payload of the message as a byte array.
// Will return bytes containing the byte array and an ok flag indicating if it was
// successful. If the content is not accessible in byte array form, an empty slice will
// be returned and the ok flag will be false.
func (message *MessageImpl) GetPayloadAsBytes() (bytes []byte, ok bool) {
	binaryAttachmentBytes, binaryAttachmentOk := ccsmp.SolClientMessageGetBinaryAttachmentAsBytes(message.messagePointer)
	xmlContentBytes, xmlContentOk := ccsmp.SolClientMessageGetXMLAttachmentAsBytes(message.messagePointer)
	if binaryAttachmentOk && xmlContentOk {
		logging.Default.Warning(fmt.Sprintf("Internal error: message %p contained multiple payloads", message))
		return nil, false
	}
	if binaryAttachmentOk {
		return binaryAttachmentBytes, true
	}
	if xmlContentOk {
		return xmlContentBytes, true
	}
	return nil, false
}

// GetPayloadAsString will attempt to get the payload of the message as a string.
// Will return a string containing the data stored in the message and an ok flag
// indicating if it was successful. If the content is not accessible in string form,
// an empty string will be returned and the ok flag will be false.
func (message *MessageImpl) GetPayloadAsString() (str string, ok bool) {
	binaryAttachmentString, binaryAttachmentOk := ccsmp.SolClientMessageGetBinaryAttachmentAsString(message.messagePointer)
	xmlContentBytes, xmlContentOk := ccsmp.SolClientMessageGetXMLAttachmentAsBytes(message.messagePointer)
	if binaryAttachmentOk && xmlContentOk {
		logging.Default.Warning(fmt.Sprintf("Internal error: message %p contained multiple payloads", message))
		return "", false
	}
	if binaryAttachmentOk {
		return binaryAttachmentString, true
	}
	if xmlContentOk {
		return string(xmlContentBytes), true
	}
	return "", false
}

// GetPayloadAsMap will attempt to get the payload of the message as an SDTMap.
// Will return a SDTMap instance containing the data stored in the message and
// an ok indicating if it was success. If the content is not accessible in SDTMap
// form, sdtMap will be nil and ok will be false.
func (message *MessageImpl) GetPayloadAsMap() (sdt.Map, bool) {
	container, ok := ccsmp.SolClientMessageGetBinaryAttachmentAsMap(message.messagePointer)
	if !ok {
		return nil, false
	}
	defer func() {
		errorInfo := container.SolClientContainerClose()
		if errorInfo != nil && logging.Default.IsDebugEnabled() {
			logging.Default.Debug(fmt.Sprintf("Encountered error while closing container: %s, errorCode %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
	}()
	parsedMap := parseMap(container)
	if parsedMap == nil {
		return nil, false
	}
	return parsedMap, true
}

// GetPayloadAsStream will attempt to get the payload of the message as an SDTStream.
// Will return a SDTStream instance containing the data stored in the message and
// an ok indicating if it was success. If the content is not accessible in SDTStream
// form, sdtStream will be nil and ok will be false.
func (message *MessageImpl) GetPayloadAsStream() (sdtStream sdt.Stream, ok bool) {
	container, ok := ccsmp.SolClientMessageGetBinaryAttachmentAsStream(message.messagePointer)
	if !ok {
		return nil, false
	}
	defer func() {
		errorInfo := container.SolClientContainerClose()
		if errorInfo != nil && logging.Default.IsDebugEnabled() {
			logging.Default.Debug(fmt.Sprintf("Encountered error while closing container: %s, errorCode %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
	}()
	parsedStream := parseStream(container)
	if parsedStream == nil {
		return nil, false
	}
	return parsedStream, true
}

// GetCorrelationID will return the correlation ID of the message.
// If not present, id will be an empty string and ok will be false.
func (message *MessageImpl) GetCorrelationID() (id string, ok bool) {
	var err core.ErrorInfo
	id, err = ccsmp.SolClientMessageGetCorrelationID(message.messagePointer)
	if err != nil {
		ok = false
		if err.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Failed to retrieve CorrelationID: "+err.GetMessageAsString()+", sub code %d", err.SubCode))
		}
	} else {
		ok = true
	}
	return id, ok
}

// GetExpiration will return the expiration time of the message.
// The expiration time is UTC time when the message is discarded or
// moved to the Dead Message Queue by the PubSub+ broker.
// A value of 0 (as determined by time.isZero()) indicates that the
// message never expires. The default value is 0.
func (message *MessageImpl) GetExpiration() time.Time {
	time, err := ccsmp.SolClientMessageGetExpiration(message.messagePointer)
	if err != nil {
		logging.Default.Warning(fmt.Sprintf("Failed to retrieve Expiration: "+err.GetMessageAsString()+", sub code %d", err.SubCode))
	}
	return time
}

// GetSequenceNumber will return the sequence number of the message.
// Sequence numbers may be set by the publisher applications or
// automatically generated by the publisher APIs. The sequence number
// is carried in the Message meta data in addition to the payload and
// may be retrieved by consumer applications. Returns a positive
// sequenceNumber if set, or ok of false if not set.
func (message *MessageImpl) GetSequenceNumber() (sequenceNumber int64, ok bool) {
	var err core.ErrorInfo
	sequenceNumber, err = ccsmp.SolClientMessageGetSequenceNumber(message.messagePointer)
	if err != nil {
		ok = false
		if err.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Failed to retrieve SequenceNumber: "+err.GetMessageAsString()+", sub code %d", err.SubCode))
		}
	} else {
		ok = true
	}
	return sequenceNumber, ok
}

// GetPriority will return the priority value. Valid priorities range from
// 0 to 255. Returns the priority, or ok of false if not set.
func (message *MessageImpl) GetPriority() (priority int, ok bool) {
	var err core.ErrorInfo
	priority, err = ccsmp.SolClientMessageGetPriority(message.messagePointer)
	if err != nil {
		ok = false
	} else {
		ok = true
	}
	return priority, ok && priority != -1
}

// GetHTTPContentType will return the HTTPContentType set on the message.
// If not set, will return an empty string and ok false.
func (message *MessageImpl) GetHTTPContentType() (contentType string, ok bool) {
	var err core.ErrorInfo
	contentType, err = ccsmp.SolClientMessageGetHTTPContentType(message.messagePointer)
	if err != nil {
		ok = false
		if err.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Failed to retrieve HTTPContentType: "+err.GetMessageAsString()+", sub code %d", err.SubCode))
		}
	} else {
		ok = true
	}
	return contentType, ok
}

// GetHTTPContentEncoding will return the HTTPContentEncoding set on the message.
// If not set, will return an empty string and ok false.
func (message *MessageImpl) GetHTTPContentEncoding() (contentEncoding string, ok bool) {
	var err core.ErrorInfo
	contentEncoding, err = ccsmp.SolClientMessageGetHTTPContentEncoding(message.messagePointer)
	if err != nil {
		ok = false
		if err.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Failed to retrieve HTTPContentEncoding: "+err.GetMessageAsString()+", sub code %d", err.SubCode))
		}
	} else {
		ok = true
	}
	return contentEncoding, ok
}

// GetApplicationMessageID will return the Application Message ID of the message.
// This value is used by applications only and is passed through the API untouched.
// If not set, will return an empty string and ok false.
func (message *MessageImpl) GetApplicationMessageID() (applicationMessageID string, ok bool) {
	var err core.ErrorInfo
	applicationMessageID, err = ccsmp.SolClientMessageGetApplicationMessageID(message.messagePointer)
	if err != nil {
		ok = false
		if err.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Failed to retrieve ApplicationMsgId: "+err.GetMessageAsString()+", sub code %d", err.SubCode))
		}
	} else {
		ok = true
	}
	return applicationMessageID, ok
}

// GetApplicationMessageType will return the Application Message Type of the message.
// This value is used by applications only and is passed through the API untouched.
// If not set, will return an empty string and ok false.
func (message *MessageImpl) GetApplicationMessageType() (applicationMessageType string, ok bool) {
	var err core.ErrorInfo
	applicationMessageType, err = ccsmp.SolClientMessageGetApplicationMessageType(message.messagePointer)
	if err != nil {
		ok = false
		if err.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Failed to retrieve ApplicationMsgType: "+err.GetMessageAsString()+", sub code %d", err.SubCode))
		}
	} else {
		ok = true
	}
	return applicationMessageType, ok
}

// GetClassOfService function
func (message *MessageImpl) GetClassOfService() int {
	classOfService, err := ccsmp.SolClientMessageGetClassOfService(message.messagePointer)
	if err != nil {
		logging.Default.Warning(fmt.Sprintf("Failed to retrieve ClassOfService: "+err.GetMessageAsString()+", sub code %d", err.SubCode))
	}
	return classOfService
}

func (message *MessageImpl) String() string {
	return ccsmp.SolClientMessageDump(message.messagePointer)
}

func parseMap(container *ccsmp.SolClientOpaqueContainer) sdt.Map {
	if container.Type != ccsmp.SolClientOpaqueContainerMap {
		return nil
	}
	m := sdt.Map{}
	var ok bool
	var key string
	var value ccsmp.SolClientField
	for {
		key, value, ok = container.SolClientContainerGetNextField()
		if !ok {
			// we are at the end of the fields
			break
		}
		if data, dataOk := parseData(value); dataOk {
			m[key] = data
		}
	}
	return m
}

func parseStream(container *ccsmp.SolClientOpaqueContainer) sdt.Stream {
	if container.Type != ccsmp.SolClientOpaqueContainerStream {
		return nil
	}
	s := sdt.Stream{}
	var ok bool
	var value ccsmp.SolClientField
	for {
		_, value, ok = container.SolClientContainerGetNextField()
		if !ok {
			break
		}
		if data, dataOk := parseData(value); dataOk {
			s = append(s, data)
		}
	}
	return s
}

// Turns CCSMP types into Go API Types
func parseData(field ccsmp.SolClientField) (interface{}, bool) {
	if data, dataOk := ccsmp.GetData(field); dataOk {
		if data == nil {
			return nil, true
		}
		switch casted := data.(type) {
		case *ccsmp.SolClientOpaqueContainer:
			if casted.Type == ccsmp.SolClientOpaqueContainerMap {
				return parseMap(casted), true
			} else if casted.Type == ccsmp.SolClientOpaqueContainerStream {
				return parseStream(casted), true
			} else {
				if logging.Default.IsDebugEnabled() {
					logging.Default.Debug(fmt.Sprintf("message.parseData: Unknown container type %d", casted.Type))
				}
				return nil, false
			}
		case *ccsmp.SolClientContainerDest:
			switch casted.DestType {
			case ccsmp.SolClientContainerDestQueue:
				return resource.QueueDurableExclusive(casted.Dest), true
			case ccsmp.SolClientContainerDestTopic:
				return resource.TopicOf(casted.Dest), true
			default:
				if logging.Default.IsDebugEnabled() {
					logging.Default.Debug(fmt.Sprintf("message.parseData: Unknown destination type %d", casted.DestType))
				}
				return nil, false
			}
		case ccsmp.SolClientWChar:
			return sdt.WChar(casted), true
		default:
			return casted, true
		}
	}
	return nil, false
}
