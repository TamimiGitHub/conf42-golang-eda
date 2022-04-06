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

package ccsmp

/*
#include <stdlib.h>
#include <string.h>

#include "solclient/solClient.h"
#include "solclient/solClientMsg.h"
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"

	"solace.dev/go/messaging/internal/impl/logging"
)

// SolClientMessagePt is assigned a value
type SolClientMessagePt = C.solClient_opaqueMsg_pt

// SolClientDestination is assigned a value
type SolClientDestination = C.solClient_destination_t

// SolClientMessageID is assigned a value
type SolClientMessageID = C.solClient_msgId_t

// SolClientDeliveryModeDirect is assigned a value
const SolClientDeliveryModeDirect = C.SOLCLIENT_DELIVERY_MODE_DIRECT

// SolClientDeliveryModeNonPersistent is assigned a value
const SolClientDeliveryModeNonPersistent = C.SOLCLIENT_DELIVERY_MODE_NONPERSISTENT

// SolClientDeliveryModePersistent is assigned a value
const SolClientDeliveryModePersistent = C.SOLCLIENT_DELIVERY_MODE_PERSISTENT

// TODO the calls to handleCcsmpError are slow since they lock the thread.
// Ideally, we wrap these calls in C such that the golang scheduler cannot
// interrupt us, and then there is no need to lock the thread. This should
// be done for all datapath functionality, ie. the contents of this file.

// SolClientMessageAlloc function
func SolClientMessageAlloc() (messageP SolClientMessagePt, errorInfo *SolClientErrorInfoWrapper) {
	errorInfo = handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_alloc(&messageP)
	})
	return messageP, errorInfo
}

// SolClientMessageFree function
func SolClientMessageFree(messageP *SolClientMessagePt) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_free(messageP)
	})
}

// SolClientMessageDup function
func SolClientMessageDup(messageP SolClientMessagePt) (duplicateP SolClientMessagePt, errorInfo *SolClientErrorInfoWrapper) {
	errorInfo = handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_dup(messageP, &duplicateP)
	})
	return duplicateP, errorInfo
}

// Payload handlers

// SolClientMessageGetBinaryAttachmentAsBytes function
func SolClientMessageGetBinaryAttachmentAsBytes(messageP SolClientMessagePt) ([]byte, bool) {
	var dataPtr unsafe.Pointer
	var size C.uint
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getBinaryAttachmentPtr(messageP, &dataPtr, &size)
	})
	if errorInfo != nil {
		if errorInfo.ReturnCode == SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Encountered error fetching payload as bytes: %s, subcode: %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
		return nil, false
	}
	return C.GoBytes(dataPtr, C.int(size)), true
}

// SolClientMessageGetXMLAttachmentAsBytes function
func SolClientMessageGetXMLAttachmentAsBytes(messageP SolClientMessagePt) ([]byte, bool) {
	var dataPtr unsafe.Pointer
	var size C.uint
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getXmlPtr(messageP, &dataPtr, &size)
	})
	if errorInfo != nil {
		if errorInfo.ReturnCode == SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Encountered error fetching XML payload as bytes: %s, subcode: %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
		return nil, false
	}
	return C.GoBytes(dataPtr, C.int(size)), true
}

// SolClientMessageSetBinaryAttachmentAsBytes function
func SolClientMessageSetBinaryAttachmentAsBytes(messageP SolClientMessagePt, arr []byte) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		if len(arr) > 0 {
			// Use set binary attachment pointer to avoid an additional copy
			// We also cannot safely get the address of the array if it happens to be zero length
			return C.solClient_msg_setBinaryAttachment(messageP, unsafe.Pointer(&arr[0]), C.solClient_uint32_t(len(arr)))
		}
		return C.solClient_msg_setBinaryAttachment(messageP, nil, C.solClient_uint32_t(0))
	})
}

// SolClientMessageGetBinaryAttachmentAsString function
func SolClientMessageGetBinaryAttachmentAsString(messageP SolClientMessagePt) (string, bool) {
	var dataP *C.char
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getBinaryAttachmentString(messageP, &dataP)
	})
	if errorInfo != nil {
		if errorInfo.ReturnCode == SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Encountered error fetching payload as string: %s, subcode: %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
		return "", false
	}
	return C.GoString(dataP), true
}

// SolClientMessageSetBinaryAttachmentString function
func SolClientMessageSetBinaryAttachmentString(messageP SolClientMessagePt, str string) *SolClientErrorInfoWrapper {
	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))
	return handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setBinaryAttachmentString(messageP, cStr)
	})
}

// SolClientMessageGetBinaryAttachmentAsStream function
func SolClientMessageGetBinaryAttachmentAsStream(messageP SolClientMessagePt) (*SolClientOpaqueContainer, bool) {
	var opaqueContainerPt SolClientOpaqueContainerPt
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getBinaryAttachmentStream(messageP, &opaqueContainerPt)
	})
	if errorInfo != nil {
		if errorInfo.ReturnCode == SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Encountered error fetching payload as stream: %s, subcode: %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
		return nil, false
	}
	return &SolClientOpaqueContainer{
		pointer: opaqueContainerPt,
		Type:    SolClientOpaqueContainerStream,
	}, true
}

// SolClientMessageGetBinaryAttachmentAsMap function
func SolClientMessageGetBinaryAttachmentAsMap(messageP SolClientMessagePt) (*SolClientOpaqueContainer, bool) {
	var opaqueContainerPt SolClientOpaqueContainerPt
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getBinaryAttachmentMap(messageP, &opaqueContainerPt)
	})
	if errorInfo != nil {
		if errorInfo.ReturnCode == SolClientReturnCodeFail {
			logging.Default.Warning(fmt.Sprintf("Encountered error fetching payload as stream: %s, subcode: %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
		return nil, false
	}
	return &SolClientOpaqueContainer{
		pointer: opaqueContainerPt,
		Type:    SolClientOpaqueContainerMap,
	}, true
}

// User properties

// SolClientMessageGetUserPropertyMap function
func SolClientMessageGetUserPropertyMap(messageP SolClientMessagePt) (*SolClientOpaqueContainer, *SolClientErrorInfoWrapper) {
	var opaqueContainerPt SolClientOpaqueContainerPt
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getUserPropertyMap(messageP, &opaqueContainerPt)
	})
	if errorInfo != nil {
		return nil, errorInfo
	}
	return &SolClientOpaqueContainer{
		pointer: opaqueContainerPt,
		Type:    SolClientOpaqueContainerMap,
	}, nil
}

// Message properties

// SolClientMessageGetApplicationMessageType function
func SolClientMessageGetApplicationMessageType(messageP SolClientMessagePt) (string, *SolClientErrorInfoWrapper) {
	var cChar *C.char
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getApplicationMsgType(messageP, &cChar)
	})
	return C.GoString(cChar), errorInfo
}

// SolClientMessageSetApplicationMessageType function
func SolClientMessageSetApplicationMessageType(messageP SolClientMessagePt, msgType string) *SolClientErrorInfoWrapper {
	cStr := C.CString(msgType)
	defer C.free(unsafe.Pointer(cStr))
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setApplicationMsgType(messageP, cStr)
	})
	return errorInfo
}

// SolClientMessageGetCorrelationID function
func SolClientMessageGetCorrelationID(messageP SolClientMessagePt) (string, *SolClientErrorInfoWrapper) {
	var cChar *C.char
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getCorrelationId(messageP, &cChar)
	})
	return C.GoString(cChar), errorInfo
}

// SolClientMessageSetCorrelationID function
func SolClientMessageSetCorrelationID(messageP SolClientMessagePt, correlationID string) *SolClientErrorInfoWrapper {
	cStr := C.CString(correlationID)
	defer C.free(unsafe.Pointer(cStr))
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setCorrelationId(messageP, cStr)
	})
	return errorInfo
}

// SolClientMessageGetApplicationMessageID function
func SolClientMessageGetApplicationMessageID(messageP SolClientMessagePt) (string, *SolClientErrorInfoWrapper) {
	var cChar *C.char
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getApplicationMsgType(messageP, &cChar)
	})
	return C.GoString(cChar), errorInfo
}

// SolClientMessageSetApplicationMessageID function
func SolClientMessageSetApplicationMessageID(messageP SolClientMessagePt, msgID string) *SolClientErrorInfoWrapper {
	cStr := C.CString(msgID)
	defer C.free(unsafe.Pointer(cStr))
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setApplicationMsgType(messageP, cStr)
	})
	return errorInfo
}

// SolClientMessageGetDestinationName function
func SolClientMessageGetDestinationName(messageP SolClientMessagePt) (destName string, errorInfo *SolClientErrorInfoWrapper) {
	var dest *SolClientDestination = &SolClientDestination{}
	errorInfo = handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getDestination(messageP, dest, (C.size_t)(unsafe.Sizeof(*dest)))
	})
	if errorInfo == nil {
		destName = C.GoString(dest.dest)
	}
	return destName, errorInfo
}

// SolClientMessageSetDestination function
func SolClientMessageSetDestination(messageP SolClientMessagePt, destinationString string) *SolClientErrorInfoWrapper {
	destination := &SolClientDestination{}
	destination.destType = C.SOLCLIENT_TOPIC_DESTINATION
	destination.dest = C.CString(destinationString)
	defer C.free(unsafe.Pointer(destination.dest))
	return handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setDestination(messageP, destination, (C.size_t)(unsafe.Sizeof(*destination)))
	})
}

// SolClientMessageGetExpiration function
func SolClientMessageGetExpiration(messageP SolClientMessagePt) (time.Time, *SolClientErrorInfoWrapper) {
	var cint64 C.longlong
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getExpiration(messageP, &cint64)
	})
	timeInMillis := int64(cint64)
	// convert milliseconds to seconds for first argument, nanoseconds for second argument
	return time.Unix(timeInMillis/1e3, (timeInMillis%1e3)*1e6), errorInfo
}

// SolClientMessageSetExpiration function
func SolClientMessageSetExpiration(messageP SolClientMessagePt, expiration int64) *SolClientErrorInfoWrapper {
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setExpiration(messageP, C.solClient_int64_t(expiration))
	})
	return errorInfo
}

// SolClientMessageGetPriority function
func SolClientMessageGetPriority(messageP SolClientMessagePt) (int, *SolClientErrorInfoWrapper) {
	var cint C.int
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getPriority(messageP, &cint)
	})
	return int(cint), errorInfo
}

// SolClientMessageSetPriority function
func SolClientMessageSetPriority(messageP SolClientMessagePt, priority int) *SolClientErrorInfoWrapper {
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setPriority(messageP, C.solClient_int32_t(priority))
	})
	return errorInfo
}

// SolClientMessageGetHTTPContentType function
func SolClientMessageGetHTTPContentType(messageP SolClientMessagePt) (string, *SolClientErrorInfoWrapper) {
	var cChar *C.char
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getHttpContentType(messageP, &cChar)
	})
	return C.GoString(cChar), errorInfo
}

// SolClientMessageSetHTTPContentType function
func SolClientMessageSetHTTPContentType(messageP SolClientMessagePt, contentType string) *SolClientErrorInfoWrapper {
	cStr := C.CString(contentType)
	defer C.free(unsafe.Pointer(cStr))
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setHttpContentType(messageP, cStr)
	})
	return errorInfo
}

// SolClientMessageGetHTTPContentEncoding function
func SolClientMessageGetHTTPContentEncoding(messageP SolClientMessagePt) (string, *SolClientErrorInfoWrapper) {
	var cChar *C.char
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getHttpContentEncoding(messageP, &cChar)
	})
	return C.GoString(cChar), errorInfo
}

// SolClientMessageSetHTTPContentEncoding function
func SolClientMessageSetHTTPContentEncoding(messageP SolClientMessagePt, contentEncoding string) *SolClientErrorInfoWrapper {
	cStr := C.CString(contentEncoding)
	defer C.free(unsafe.Pointer(cStr))
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setHttpContentEncoding(messageP, cStr)
	})
	return errorInfo
}

// SolClientMessageGetSequenceNumber function
func SolClientMessageGetSequenceNumber(messageP SolClientMessagePt) (int64, *SolClientErrorInfoWrapper) {
	var cint64 C.longlong
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getSequenceNumber(messageP, &cint64)
	})
	return int64(cint64), errorInfo
}

// SolClientMessageSetSequenceNumber function
func SolClientMessageSetSequenceNumber(messageP SolClientMessagePt, sequenceNumber uint64) *SolClientErrorInfoWrapper {
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setSequenceNumber(messageP, C.solClient_uint64_t(sequenceNumber))
	})
	return errorInfo
}

// SolClientMessageGetClassOfService function
func SolClientMessageGetClassOfService(messageP SolClientMessagePt) (int, *SolClientErrorInfoWrapper) {
	var cint C.solClient_uint32_t
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getClassOfService(messageP, &cint)
	})
	return int(cint), errorInfo
}

// SolClientMessageSetClassOfService function
func SolClientMessageSetClassOfService(messageP SolClientMessagePt, classOfService int) *SolClientErrorInfoWrapper {
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setClassOfService(messageP, C.solClient_uint32_t(classOfService))
	})
	return errorInfo
}

// SolClientMessageGetSenderID function
func SolClientMessageGetSenderID(messageP SolClientMessagePt) (string, *SolClientErrorInfoWrapper) {
	var cString *C.char
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getSenderId(messageP, &cString)
	})
	if errorInfo != nil {
		return "", errorInfo
	}
	return C.GoString(cString), nil
}

// SolClientMessageSetSenderID function
func SolClientMessageSetSenderID(messageP SolClientMessagePt, senderID string) *SolClientErrorInfoWrapper {
	cStr := C.CString(senderID)
	defer C.free(unsafe.Pointer(cStr))
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setSenderId(messageP, cStr)
	})
	return errorInfo
}

// Read only properties

// SolClientMessageGetMessageDiscardNotification function
func SolClientMessageGetMessageDiscardNotification(messageP SolClientMessagePt) bool {
	var hasDiscardNotification C.solClient_bool_t = C.solClient_msg_isDiscardIndication(messageP)
	return *(*bool)(unsafe.Pointer(&hasDiscardNotification))
}

// SolClientMessageGetMessageIsRedelivered function
func SolClientMessageGetMessageIsRedelivered(messageP SolClientMessagePt) bool {
	var isRedelivered C.solClient_bool_t = C.solClient_msg_isRedelivered(messageP)
	return *(*bool)(unsafe.Pointer(&isRedelivered))
}

// SolClientMessageGetMessageID function
func SolClientMessageGetMessageID(messageP SolClientMessagePt) (SolClientMessageID, *SolClientErrorInfoWrapper) {
	var messageID C.solClient_msgId_t
	err := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getMsgId(messageP, &messageID)
	})
	return messageID, err
}

// SolClientMessageGetSenderTimestamp function
func SolClientMessageGetSenderTimestamp(messageP SolClientMessagePt) (time.Time, *SolClientErrorInfoWrapper) {
	var cint64 C.solClient_int64_t
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getSenderTimestamp(messageP, &cint64)
	})
	if errorInfo != nil {
		return time.Time{}, errorInfo
	}
	timeInMillis := int64(cint64)
	return time.Unix(timeInMillis/1e3, (timeInMillis%1e3)*1e6), nil
}

// SolClientMessageGetTimestamp function
func SolClientMessageGetTimestamp(messageP SolClientMessagePt) (time.Time, *SolClientErrorInfoWrapper) {
	var cint64 C.solClient_int64_t
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getRcvTimestamp(messageP, &cint64)
	})
	if errorInfo != nil {
		return time.Time{}, errorInfo
	}
	timeInMillis := int64(cint64)
	return time.Unix(timeInMillis/1e3, (timeInMillis%1e3)*1e6), nil
}

// SolClientMessageGetRGMID function
func SolClientMessageGetRGMID(messageP SolClientMessagePt) (SolClientRGMIDPt, *SolClientErrorInfoWrapper) {
	var rmid SolClientRGMID
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getReplicationGroupMessageId(messageP, &rmid, SolClientRGMIDSize)
	})
	if errorInfo != nil {
		return nil, errorInfo
	}
	return &rmid, nil
}

// Write only properties

// SolClientMessageSetAckImmediately function
func SolClientMessageSetAckImmediately(messageP SolClientMessagePt, val bool) *SolClientErrorInfoWrapper {
	var isAckImmediately uint8
	if val {
		isAckImmediately = 1
	}
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setAckImmediately(messageP, C.solClient_bool_t(isAckImmediately))
	})
	return errorInfo
}

// SolClientMessageSetCorrelationTag function
func SolClientMessageSetCorrelationTag(messageP SolClientMessagePt, correlationTag []byte) *SolClientErrorInfoWrapper {
	if len(correlationTag) > 0 {
		errorInfo := handleCcsmpError(func() SolClientReturnCode {
			return C.solClient_msg_setCorrelationTag(messageP, unsafe.Pointer(&correlationTag[0]), C.solClient_uint32_t(len(correlationTag)))
		})
		return errorInfo
	}
	return nil
}

// SolClientMessageSetDeliveryMode function
func SolClientMessageSetDeliveryMode(messageP SolClientMessagePt, deliveryMode uint32) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setDeliveryMode(messageP, C.solClient_uint32_t(deliveryMode))
	})
}

// SolClientMessageSetDMQEligible function
func SolClientMessageSetDMQEligible(messageP SolClientMessagePt, DMQEligible bool) *SolClientErrorInfoWrapper {
	var isDMQEligible C.solClient_bool_t = 0
	if DMQEligible {
		isDMQEligible = 1
	}
	return handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setDMQEligible(messageP, isDMQEligible)
	})
}

// SolClientMessageSetElidingEligible function
func SolClientMessageSetElidingEligible(messageP SolClientMessagePt, elidingEligible bool) *SolClientErrorInfoWrapper {
	var isElidingEligible C.solClient_bool_t = 0
	if elidingEligible {
		isElidingEligible = 1
	}
	return handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setElidingEligible(messageP, isElidingEligible)
	})
}

// SolClientMessageSetTimeToLive function
func SolClientMessageSetTimeToLive(messageP SolClientMessagePt, timeToLive int64) *SolClientErrorInfoWrapper {
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_setTimeToLive(messageP, C.solClient_int64_t(timeToLive))
	})
	return errorInfo
}

// Utility functions

const defaultMsgDumpBufferSize = 1000
const msgDumpMultiplier = 5
const maxDumpSize = 10000

// SolClientMessageDump function
func SolClientMessageDump(messageP SolClientMessagePt) string {
	// couple of things we need to do: first, get the size of the payload
	// then allocate a byte buffer of that size to pass to msg_dump
	// then convert that into a string and return making sure to free any memory
	var dataPtr unsafe.Pointer
	var payloadSize C.uint
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_getBinaryAttachmentPtr(messageP, &dataPtr, &payloadSize)
	})
	// we failed to get the binary attachment
	if errorInfo != nil {
		// if we got an actual error, return
		if errorInfo.ReturnCode != SolClientReturnCodeNotFound {
			logging.Default.Warning("Failed to retrieve binary attachment pointer: " + errorInfo.String())
			return ""
		}
		// otherwise, we check for a string attachment
		var cStr *C.char
		errorInfo = handleCcsmpError(func() SolClientReturnCode {
			return C.solClient_msg_getBinaryAttachmentString(messageP, &cStr)
		})
		// if we got an actual error, return
		if errorInfo != nil {
			if errorInfo.ReturnCode != SolClientReturnCodeNotFound {
				logging.Default.Warning("Failed to retrieve string attachment: " + errorInfo.String())
				return ""
			}
			errorInfo = handleCcsmpError(func() SolClientReturnCode {
				return C.solClient_msg_getXmlPtr(messageP, &dataPtr, &payloadSize)
			})
			if errorInfo != nil {
				if errorInfo.ReturnCode != SolClientReturnCodeNotFound {
					logging.Default.Warning("Failed to retrieve xml attachment pointer: " + errorInfo.String())
					return ""
				}
				// we did not find any attachment, set the payload to 0
				payloadSize = 0
			}
		} else {
			// if we did not, process the cstring into a length for payloadSize
			payloadSize = C.uint(C.strlen(cStr))
		}
	}

	bufferSize := C.ulong(defaultMsgDumpBufferSize + payloadSize*msgDumpMultiplier)
	// Truncate the message after 10,000 characters, SOL-62945
	if bufferSize > maxDumpSize {
		bufferSize = maxDumpSize
	}
	buffer := (*C.char)(C.malloc(bufferSize))
	defer C.free(unsafe.Pointer(buffer))

	errorInfo = handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_dump(messageP, buffer, bufferSize)
	})
	if errorInfo != nil {
		logging.Default.Warning(errorInfo.String())
		return ""
	}
	return C.GoString(buffer)
}
