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
	"runtime"
	"strconv"
	"sync/atomic"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/internal/impl/logging"
)

// DeliveryMode type
type DeliveryMode uint32

// DeliveryModeDirect constant
const DeliveryModeDirect DeliveryMode = ccsmp.SolClientDeliveryModeDirect

// DeliveryModeNonPersistent constant
const DeliveryModeNonPersistent DeliveryMode = ccsmp.SolClientDeliveryModeNonPersistent

// DeliveryModePersistent constant
const DeliveryModePersistent DeliveryMode = ccsmp.SolClientDeliveryModePersistent

// OutboundMessageImpl structure
type OutboundMessageImpl struct {
	MessageImpl
}

// NewOutboundMessage returns a new Message object that can be used
func NewOutboundMessage() (*OutboundMessageImpl, error) {
	msgP, err := ccsmp.SolClientMessageAlloc()
	if err != nil {
		return nil, core.ToNativeError(err, "error allocating message: ")
	}
	return newOutboundMessage(msgP), nil
}

func newOutboundMessage(msgP ccsmp.SolClientMessagePt) *OutboundMessageImpl {
	ret := &OutboundMessageImpl{MessageImpl{messagePointer: msgP, disposed: 0}}
	runtime.SetFinalizer(ret, freeOutboundMessage)
	return ret
}

// Dispose will free all underlying resources of the Disposable instance.
// Dispose is idempotent, and will remove any redundant finalizers on the
// instance, substantially improving garbage collection performance.
// This function is threadsafe, and subsequent calls to Dispose will
// block waiting for the first call to complete. Additional calls
// will return immediately. The instance is considered unusable after Dispose
// has been called.
func (message *OutboundMessageImpl) Dispose() {
	proceed := atomic.CompareAndSwapInt32(&message.disposed, 0, 1)
	if proceed {
		// free ccsmp message pointer
		freeOutboundMessage(message)
		// clear the finalizer
		runtime.SetFinalizer(message, nil)
	}
}

// freeOutboundMessage will free the underlying message pointer
func freeOutboundMessage(message *OutboundMessageImpl) {
	err := ccsmp.SolClientMessageFree(&message.messagePointer)
	if err != nil && logging.Default.IsErrorEnabled() {
		logging.Default.Error("encountered unexpected error while freeing message pointer: " + err.GetMessageAsString() + " [sub code = " + strconv.Itoa(int(err.SubCode)) + "]")
	}
}

// DuplicateOutboundMessage will duplicate the message and return a new Message copying the original
func DuplicateOutboundMessage(message *OutboundMessageImpl) (*OutboundMessageImpl, error) {
	msgP, err := ccsmp.SolClientMessageDup(message.messagePointer)
	if err != nil {
		return nil, core.ToNativeError(err, "error duplicating message: ")
	}
	return newOutboundMessage(msgP), nil
}

// AttachCorrelationTag function
func AttachCorrelationTag(message *OutboundMessageImpl, bytes []byte) error {
	err := ccsmp.SolClientMessageSetCorrelationTag(message.messagePointer, bytes)
	if err != nil {
		return core.ToNativeError(err, "error attaching correlation key: ")
	}
	return nil
}

// SetDeliveryMode function
func SetDeliveryMode(message *OutboundMessageImpl, deliveryMode DeliveryMode) error {
	err := ccsmp.SolClientMessageSetDeliveryMode(message.messagePointer, uint32(deliveryMode))
	if err != nil {
		return core.ToNativeError(err, "error setting delivery mode: ")
	}
	return nil
}

// SetDestination function
func SetDestination(message *OutboundMessageImpl, destName string) error {
	err := ccsmp.SolClientMessageSetDestination(message.messagePointer, destName)
	if err != nil {
		return core.ToNativeError(err, "error setting destination: ")
	}
	return nil
}

// SetAckImmediately function
func SetAckImmediately(message *OutboundMessageImpl) error {
	err := ccsmp.SolClientMessageSetAckImmediately(message.messagePointer, true)
	if err != nil {
		return core.ToNativeError(err, "error setting ack immediately: ")
	}
	return nil
}

// GetOutboundMessagePointer function
func GetOutboundMessagePointer(message *OutboundMessageImpl) ccsmp.SolClientMessagePt {
	if message == nil {
		return nil
	}
	return message.messagePointer
}
