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

import (
	"fmt"
	"unsafe"

	"solace.dev/go/messaging/internal/impl/logging"
)

/*
#include <stdlib.h>

#include "solclient/solClient.h"
#include "solclient/solClientMsg.h"
*/
import "C"

// SolClientOpaqueContainerPt is assigned the value C.solClient_opaqueContainer_pt
type SolClientOpaqueContainerPt = C.solClient_opaqueContainer_pt

// SolClientOpaquePointerPt is assigned the value C.solClient_opaquePointer_pt
type SolClientOpaquePointerPt = C.solClient_opaquePointer_pt

// SolClientField is assigned the value C.solClient_field_t
type SolClientField = C.solClient_field_t

// SolClientWChar is assigned the value C.solClient_wchar_t
type SolClientWChar = C.solClient_wchar_t

// SolClientOpaqueContainerType is of type uint8
type SolClientOpaqueContainerType uint8

const (
	// SolClientOpaqueContainerMap is initialized to 0
	SolClientOpaqueContainerMap SolClientOpaqueContainerType = iota
	// SolClientOpaqueContainerStream is initialized to 1
	SolClientOpaqueContainerStream
)

// SolClientOpaqueContainer is a structure containing variables of type SolClientOpaqueContainerPt and SolClientOpaqueContainerType
type SolClientOpaqueContainer struct {
	pointer SolClientOpaqueContainerPt
	Type    SolClientOpaqueContainerType
}

// SolClientContainerDestType is of type uint8
type SolClientContainerDestType uint8

const (
	// SolClientContainerDestQueue is initialized to 0
	SolClientContainerDestQueue SolClientContainerDestType = iota
	// SolClientContainerDestTopic is initialized to 1
	SolClientContainerDestTopic
)

var destTypeMappingFromCCSMP = map[C.solClient_destinationType_t]SolClientContainerDestType{
	C.SOLCLIENT_TOPIC_DESTINATION:      SolClientContainerDestTopic,
	C.SOLCLIENT_TOPIC_TEMP_DESTINATION: SolClientContainerDestTopic,
	C.SOLCLIENT_QUEUE_DESTINATION:      SolClientContainerDestQueue,
	C.SOLCLIENT_QUEUE_TEMP_DESTINATION: SolClientContainerDestQueue,
}

var destTypeMappingToCCSMP = map[SolClientContainerDestType]C.solClient_destinationType_t{
	SolClientContainerDestTopic: C.SOLCLIENT_TOPIC_TEMP_DESTINATION,
	SolClientContainerDestQueue: C.SOLCLIENT_QUEUE_DESTINATION,
}

// SolClientContainerDest is a structure container variables of type string and SolClientContainerDestType
type SolClientContainerDest struct {
	Dest     string
	DestType SolClientContainerDestType
}

const containerSize = C.uint(10240)

// SolClientMessageCreateUserPropertyMap creates a user property map
func SolClientMessageCreateUserPropertyMap(messageP SolClientMessagePt) (*SolClientOpaqueContainer, *SolClientErrorInfoWrapper) {
	var opaqueContainerPt SolClientOpaqueContainerPt
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_createUserPropertyMap(messageP, &opaqueContainerPt, containerSize)
	})
	return &SolClientOpaqueContainer{
		pointer: opaqueContainerPt,
		Type:    SolClientOpaqueContainerMap,
	}, errorInfo
}

// SolClientMessageCreateBinaryAttachmentMap creates a binary attachment map
func SolClientMessageCreateBinaryAttachmentMap(messageP SolClientMessagePt) (*SolClientOpaqueContainer, *SolClientErrorInfoWrapper) {
	var opaqueContainerPt SolClientOpaqueContainerPt
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_createBinaryAttachmentMap(messageP, &opaqueContainerPt, containerSize)
	})
	return &SolClientOpaqueContainer{
		pointer: opaqueContainerPt,
		Type:    SolClientOpaqueContainerMap,
	}, errorInfo
}

// SolClientMessageCreateBinaryAttachmentStream creates a binary attachment stream
func SolClientMessageCreateBinaryAttachmentStream(messageP SolClientMessagePt) (*SolClientOpaqueContainer, *SolClientErrorInfoWrapper) {
	var opaqueContainerPt SolClientOpaqueContainerPt
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_msg_createBinaryAttachmentStream(messageP, &opaqueContainerPt, containerSize)
	})
	return &SolClientOpaqueContainer{
		pointer: opaqueContainerPt,
		Type:    SolClientOpaqueContainerStream,
	}, errorInfo
}

// SolClientContainerClose optionally close a container. This is useful when copying out the data to Golang's memory space
// and then closing the container subsequently, freeing the memory to be reused
func (opaqueContainer *SolClientOpaqueContainer) SolClientContainerClose() *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_container_closeMapStream(&opaqueContainer.pointer)
	})
}

// SolClientContainerGetNextField fetches the next field
func (opaqueContainer *SolClientOpaqueContainer) SolClientContainerGetNextField() (string, SolClientField, bool) {
	var fieldT SolClientField
	var name *C.char
	var namePtr **C.char
	if opaqueContainer.Type == SolClientOpaqueContainerMap {
		namePtr = &name
	}
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_container_getNextField(opaqueContainer.pointer, &fieldT, (C.size_t)(unsafe.Sizeof(fieldT)), namePtr)
	})
	if errorInfo != nil {
		// we expect and end of stream, but an error is logged if we fail
		if errorInfo.ReturnCode == SolClientReturnCodeFail {
			logging.Default.Debug(fmt.Sprintf("ccsmp.SolClientContainerGetNextField: Unable to retrieve next field: %s, subcode: %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
		return "", fieldT, false
	}
	var key string
	if opaqueContainer.Type == SolClientOpaqueContainerMap {
		key = C.GoString(name)
	}
	return key, fieldT, true
}

// SolClientContainerGetField fetches a field
func (opaqueContainer *SolClientOpaqueContainer) SolClientContainerGetField(key string) (SolClientField, bool) {
	var fieldT SolClientField
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_getField(opaqueContainer.pointer, &fieldT, (C.size_t)(unsafe.Sizeof(fieldT)), cKey)
	})
	if errorInfo != nil {
		// we expect and end of stream, but an error is logged if we fail
		if errorInfo.ReturnCode == SolClientReturnCodeFail {
			logging.Default.Debug(fmt.Sprintf("ccsmp.SolClientContainerGetField: unable to retrieve next field: %s, subcode: %d", errorInfo.GetMessageAsString(), errorInfo.SubCode))
		}
		return fieldT, false
	}
	return fieldT, true
}

// GetData converts the field into the relevant golang type. In particular,
// fbool, uint8-64, int8-64, string, byte array, float, double, nil,
// wchar, *SolClientOpaqueContainer or *SolClientContainerDest
func GetData(field SolClientField) (interface{}, bool) {
	switch field._type {
	case C.SOLCLIENT_BOOL:
		return *(*bool)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_UINT8:
		return *(*uint8)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_INT8:
		return *(*int8)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_UINT16:
		return *(*uint16)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_INT16:
		return *(*int16)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_UINT32:
		return *(*uint32)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_INT32:
		return *(*int32)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_UINT64:
		return *(*uint64)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_INT64:
		return *(*int64)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_STRING:
		return C.GoString(*(**C.char)(unsafe.Pointer(&field.value))), true
	case C.SOLCLIENT_BYTEARRAY:
		return C.GoBytes(unsafe.Pointer(*(**C.char)(unsafe.Pointer(&field.value))), C.int(field.length)), true
	case C.SOLCLIENT_FLOAT:
		return *(*float32)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_DOUBLE:
		return *(*float64)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_NULL:
		return nil, true
	case C.SOLCLIENT_WCHAR:
		return *(*SolClientWChar)(unsafe.Pointer(&field.value)), true
	case C.SOLCLIENT_MAP:
		return &SolClientOpaqueContainer{
			pointer: *(*SolClientOpaqueContainerPt)(unsafe.Pointer(&field.value)),
			Type:    SolClientOpaqueContainerMap,
		}, true
	case C.SOLCLIENT_STREAM:
		return &SolClientOpaqueContainer{
			pointer: *(*SolClientOpaqueContainerPt)(unsafe.Pointer(&field.value)),
			Type:    SolClientOpaqueContainerStream,
		}, true
	case C.SOLCLIENT_DESTINATION:
		destination := *(*C.solClient_destination_t)(unsafe.Pointer(&field.value))
		destType, ok := destTypeMappingFromCCSMP[destination.destType]
		if !ok {
			if logging.Default.IsDebugEnabled() {
				logging.Default.Debug(fmt.Sprintf("ccsmp.GetData: unhandled destination type %d", destination.destType))
			}
			return nil, false
		}
		return &SolClientContainerDest{
			Dest:     C.GoString(destination.dest),
			DestType: destType,
		}, true
	case C.SOLCLIENT_SMF:
		fallthrough
	case C.SOLCLIENT_UNKNOWN:
		fallthrough
	default:
		logging.Default.Debug(fmt.Sprintf("ccsmp.GetData: got unsupported field type %d", field._type))
		return nil, false
	}
}

// ContainerAddBoolean adds a boolean
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddBoolean(b bool, key string) *SolClientErrorInfoWrapper {
	var boolValue uint8
	if b {
		boolValue = 1
	} else {
		boolValue = 0
	}
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addBoolean(opaqueContainer.pointer, C.solClient_bool_t(boolValue), cKey)
	})
}

// ContainerAddUint8 adds an integer
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddUint8(val uint8, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addUint8(opaqueContainer.pointer, C.solClient_uint8_t(val), cKey)
	})
}

// ContainerAddUint16 adds an integer
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddUint16(val uint16, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addUint16(opaqueContainer.pointer, C.solClient_uint16_t(val), cKey)
	})
}

// ContainerAddUint32 adds an integer
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddUint32(val uint32, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addUint32(opaqueContainer.pointer, C.solClient_uint32_t(val), cKey)
	})
}

// ContainerAddUint64 adds an integer
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddUint64(val uint64, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addUint64(opaqueContainer.pointer, C.solClient_uint64_t(val), cKey)
	})
}

// ContainerAddInt8 adds an integer
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddInt8(val int8, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addInt8(opaqueContainer.pointer, C.solClient_int8_t(val), cKey)
	})
}

// ContainerAddInt16 adds an integer
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddInt16(val int16, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addInt16(opaqueContainer.pointer, C.solClient_int16_t(val), cKey)
	})
}

// ContainerAddInt32 adds an integer
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddInt32(val int32, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addInt32(opaqueContainer.pointer, C.solClient_int32_t(val), cKey)
	})
}

// ContainerAddInt64 adds an integer
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddInt64(val int64, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addInt64(opaqueContainer.pointer, C.solClient_int64_t(val), cKey)
	})
}

// ContainerAddString adds a string
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddString(val string, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		cStr := C.CString(val)
		defer C.free(unsafe.Pointer(cStr))
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addString(opaqueContainer.pointer, cStr, cKey)
	})
}

// ContainerAddBytes adds an integer
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddBytes(val []byte, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		if len(val) > 0 {
			return C.solClient_container_addByteArray(opaqueContainer.pointer, (*C.solClient_uint8_t)(unsafe.Pointer(&(val)[0])), C.solClient_uint32_t(len(val)), cKey)
		}
		return C.solClient_container_addByteArray(opaqueContainer.pointer, nil, C.solClient_uint32_t(len(val)), cKey)
	})
}

// ContainerAddFloat32 adds a float
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddFloat32(val float32, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addFloat(opaqueContainer.pointer, C.float(val), cKey)
	})
}

// ContainerAddFloat64 adds a float
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddFloat64(val float64, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addDouble(opaqueContainer.pointer, C.double(val), cKey)
	})
}

// ContainerAddNil adds nil
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddNil(key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addNull(opaqueContainer.pointer, cKey)
	})
}

// ContainerAddWChar adds a char
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddWChar(wchar SolClientWChar, key string) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addWchar(opaqueContainer.pointer, wchar, cKey)
	})
}

// ContainerAddDestination adds a destination
func (opaqueContainer *SolClientOpaqueContainer) ContainerAddDestination(dest string, destType SolClientContainerDestType, key string) *SolClientErrorInfoWrapper {
	var destination C.solClient_destination_t
	destString := C.CString(dest)
	defer C.free(unsafe.Pointer(destString))
	destination.dest = destString
	destination.destType = destTypeMappingToCCSMP[destType]
	return handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_addDestination(opaqueContainer.pointer, &destination, (C.size_t)(unsafe.Sizeof(destination)), cKey)
	})
}

// ContainerOpenSubMap opens a sub map
func (opaqueContainer *SolClientOpaqueContainer) ContainerOpenSubMap(key string) (*SolClientOpaqueContainer, *SolClientErrorInfoWrapper) {
	var opaqueContainerPt SolClientOpaqueContainerPt
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_openSubMap(opaqueContainer.pointer, &opaqueContainerPt, cKey)
	})
	return &SolClientOpaqueContainer{
		pointer: opaqueContainerPt,
		Type:    SolClientOpaqueContainerMap,
	}, errorInfo
}

// ContainerOpenSubStream opens a sub stream
func (opaqueContainer *SolClientOpaqueContainer) ContainerOpenSubStream(key string) (*SolClientOpaqueContainer, *SolClientErrorInfoWrapper) {
	var opaqueContainerPt SolClientOpaqueContainerPt
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		var cKey *C.char
		if opaqueContainer.Type == SolClientOpaqueContainerMap {
			cKey = C.CString(key)
			defer C.free(unsafe.Pointer(cKey))
		}
		return C.solClient_container_openSubStream(opaqueContainer.pointer, &opaqueContainerPt, cKey)
	})
	return &SolClientOpaqueContainer{
		pointer: opaqueContainerPt,
		Type:    SolClientOpaqueContainerStream,
	}, errorInfo
}
