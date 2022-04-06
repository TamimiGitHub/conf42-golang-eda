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

// Package message is defined below
package message

import (
	"fmt"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/logging"
	"solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/message/rgmid"
)

// InboundMessageImpl structure
type InboundMessageImpl struct {
	MessageImpl
	internalDiscard bool
}

// NewInboundMessage returns a new Message object that can be used
func NewInboundMessage(msgP ccsmp.SolClientMessagePt, discard bool) *InboundMessageImpl {
	ret := newInboundMessage(msgP)
	ret.internalDiscard = discard
	return ret
}

func newInboundMessage(msgP ccsmp.SolClientMessagePt) *InboundMessageImpl {
	ret := &InboundMessageImpl{MessageImpl: MessageImpl{messagePointer: msgP, disposed: 0}}
	runtime.SetFinalizer(ret, freeInboundMessage)
	return ret
}

// Dispose will free all underlying resources of the Disposable instance.
// Dispose is idempotent, and will remove any redundant finalizers on the
// instance, substantially improving garbage collection performance.
// This function is threadsafe, and subsequent calls to Dispose will
// block waiting for the first call to complete. Additional calls
// will return immediately. The instance is considered unusable after Dispose
// has been called.
func (message *InboundMessageImpl) Dispose() {
	proceed := atomic.CompareAndSwapInt32(&message.disposed, 0, 1)
	if proceed {
		// free ccsmp message pointer
		freeInboundMessage(message)
		// clear the finalizer
		runtime.SetFinalizer(message, nil)
	}
}

// free will free the underlying message pointer
func freeInboundMessage(message *InboundMessageImpl) {
	err := ccsmp.SolClientMessageFree(&message.messagePointer)
	if err != nil && logging.Default.IsErrorEnabled() {
		logging.Default.Error("encountered unexpected error while freeing message pointer: " + err.GetMessageAsString() + " [sub code = " + strconv.Itoa(int(err.SubCode)) + "]")
	}
}

// GetDestinationName gets the destination name on which the message was received.
// The destination may be either a topic or a queue.
// Returns an empty string if the information is not available.
func (message *InboundMessageImpl) GetDestinationName() string {
	destName, errorInfo := ccsmp.SolClientMessageGetDestinationName(message.messagePointer)
	if errorInfo != nil {
		if errorInfo.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Debug(fmt.Sprintf("Unable to retrieve the destination this message was published to: %s, subcode: %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
	}
	return destName
}

// GetTimeStamp will get the timestamp as time.Time.
// This timestamp represents the time that the message was received by the API.
// This may differ from the time that the message is received by the MessageReceiver.
func (message *InboundMessageImpl) GetTimeStamp() (time.Time, bool) {
	t, errInfo := ccsmp.SolClientMessageGetTimestamp(message.messagePointer)
	if errInfo != nil {
		if errInfo.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Debug(fmt.Sprintf("Encountered error retrieving Sender Timestamp: %s, subcode: %d", errInfo.GetMessageAsString(), errInfo.SubCode))
		}
		return t, false
	}
	return t, true
}

// GetSenderTimestamp will get the timestamp as time.Time.
// This timestamp is often set automatically when the message is published.
func (message *InboundMessageImpl) GetSenderTimestamp() (time.Time, bool) {
	t, errInfo := ccsmp.SolClientMessageGetSenderTimestamp(message.messagePointer)
	if errInfo != nil {
		if errInfo.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Debug(fmt.Sprintf("Encountered error retrieving Sender Timestamp: %s, subcode: %d", errInfo.GetMessageAsString(), errInfo.SubCode))
		}
		return t, false
	}
	return t, true
}

// GetSenderID will get the sender ID set on the message.
func (message *InboundMessageImpl) GetSenderID() (string, bool) {
	id, errInfo := ccsmp.SolClientMessageGetSenderID(message.messagePointer)
	if errInfo != nil {
		if errInfo.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Debug(fmt.Sprintf("Encountered error retrieving Sender ID: %s, subcode: %d", errInfo.GetMessageAsString(), errInfo.SubCode))
		}
		return id, false
	}
	return id, true
}

// IsRedelivered function
func (message *InboundMessageImpl) IsRedelivered() bool {
	return ccsmp.SolClientMessageGetMessageIsRedelivered(message.messagePointer)
}

// GetReplicationGroupMessageID function
func (message *InboundMessageImpl) GetReplicationGroupMessageID() (rgmid.ReplicationGroupMessageID, bool) {
	rmidPt, errInfo := ccsmp.SolClientMessageGetRGMID(message.messagePointer)
	if errInfo != nil {
		if errInfo.ReturnCode == ccsmp.SolClientReturnCodeFail {
			logging.Default.Debug(fmt.Sprintf("Encountered error retrieving ReplicationGroupMessageID: %s, subcode: %d", errInfo.GetMessageAsString(), errInfo.SubCode))
		}
		return nil, false
	}
	return &replicationGroupMessageID{rmidPt: rmidPt}, true
}

// GetMessageDiscardNotification retrieves the message discard notification about
// previously discarded messages. Returns a MessageDiscardNotification, not expected
// to be nil.
func (message *InboundMessageImpl) GetMessageDiscardNotification() message.MessageDiscardNotification {
	if message.IsDisposed() {
		logging.Default.Warning("Failed to retrieve discard notification: Bad msg_p pointer '0x0'")
		return nil
	}
	return &discardNotification{
		internalDiscard: message.internalDiscard,
		brokerDiscard:   ccsmp.SolClientMessageGetMessageDiscardNotification(message.messagePointer),
	}
}

type discardNotification struct {
	internalDiscard, brokerDiscard bool
}

// HasBrokerDiscardIndication determines whether the broker has discarded one
// or more messages prior to the current message.
// Returns true if messages (one or more) were previously discarded by the broker,
// otherwise false.
func (notification *discardNotification) HasBrokerDiscardIndication() bool {
	return notification.brokerDiscard
}

// HasInternalDiscardIndication determines if the API has discarded one or more messages
// prior to the current message (i.e., in a back-pressure situation).
// Returns true if messages (one or more) were previously discarded by the API, otherwise false.
func (notification *discardNotification) HasInternalDiscardIndication() bool {
	return notification.internalDiscard
}

func (notification *discardNotification) String() string {
	return fmt.Sprintf("MessageDiscardNotification{internalDiscard: %t, brokerDiscard: %t}", notification.internalDiscard, notification.brokerDiscard)
}

// MessageID defined
type MessageID = ccsmp.SolClientMessageID

// GetMessageID function
func GetMessageID(message *InboundMessageImpl) (MessageID, bool) {
	id, err := ccsmp.SolClientMessageGetMessageID(message.messagePointer)
	if err != nil {
		return 0, false
	}
	return id, true
}
