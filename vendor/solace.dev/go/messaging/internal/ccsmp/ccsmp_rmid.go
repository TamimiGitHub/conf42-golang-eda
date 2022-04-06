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
import "unsafe"

// SolClientRGMID reexports C.solClient_replicationGroupMessageId_t
type SolClientRGMID = C.solClient_replicationGroupMessageId_t

// SolClientRGMIDPt reexports C.solClient_replicationGroupMessageId_pt
type SolClientRGMIDPt = C.solClient_replicationGroupMessageId_pt

// Start with 45 bytes. This provides some flex room

// SolClientRGMIDStringLength represents the length of an RGMID string
const SolClientRGMIDStringLength = C.SOLCLIENT_REPLICATION_GROUP_MESSAGE_ID_STRING_LENGTH + 4

// SolClientRGMIDSize represents the size of an RGMID
const SolClientRGMIDSize = C.SOLCLIENT_REPLICATION_GROUP_MESSAGE_ID_SIZE

// SolClientRGMIDCompare compares RGMIDs using solClient_replicationGroupMessageId_compare
func SolClientRGMIDCompare(a, b SolClientRGMIDPt) (int, *SolClientErrorInfoWrapper) {
	var result C.int
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_replicationGroupMessageId_compare(a, b, &result)
	})
	return int(result), errorInfo
}

// SolClientRGMIDFromString creates RGMID from string
func SolClientRGMIDFromString(rmidAsString string) (SolClientRGMIDPt, *SolClientErrorInfoWrapper) {
	var newRGMID SolClientRGMID
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		cString := C.CString(rmidAsString)
		defer C.free(unsafe.Pointer(cString))
		return C.solClient_replicationGroupMessageId_fromString(&newRGMID, SolClientRGMIDSize, cString)
	})
	if errorInfo != nil {
		return nil, errorInfo
	}
	return &newRGMID, nil
}

// SolClientRGMIDToString produces a string from a rgmid
func SolClientRGMIDToString(rmid SolClientRGMIDPt) (string, *SolClientErrorInfoWrapper) {
	if rmid == nil {
		// don't proceed when nil, should never happen
		return "", nil
	}
	var cString *C.char = (*C.char)(C.malloc(SolClientRGMIDStringLength))
	defer C.free(unsafe.Pointer(cString))
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_replicationGroupMessageId_toString(rmid, SolClientRGMIDSize, cString, SolClientRGMIDStringLength)
	})
	if errorInfo != nil {
		return "", errorInfo
	}
	return C.GoString(cString), nil
}
