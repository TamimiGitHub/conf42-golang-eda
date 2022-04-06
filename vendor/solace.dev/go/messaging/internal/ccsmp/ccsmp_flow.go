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
	"sync"
	"sync/atomic"
	"unsafe"

	"solace.dev/go/messaging/internal/impl/logging"
)

/*
#include <stdlib.h>
#include <stdio.h>

#include <string.h>

#include "solclient/solClient.h"
#include "solclient/solClientMsg.h"
#include "./ccsmp_helper.h"

solClient_rxMsgCallback_returnCode_t flowMessageReceiveCallback ( solClient_opaqueFlow_pt opaqueFlow_p, solClient_opaqueMsg_pt msg_p, void *user_p );
void flowEventCallback ( solClient_opaqueFlow_pt opaqueFlow_p, solClient_flow_eventCallbackInfo_pt eventInfo_p, void *user_p );
*/
import "C"

// SolClientFlowPt is assigned a value
type SolClientFlowPt = C.solClient_opaqueFlow_pt

// SolClientFlowEventInfoPt is assigned a value
type SolClientFlowEventInfoPt = C.solClient_flow_eventCallbackInfo_pt

// SolClientFlowRxMsgDispatchFuncInfo is assigned a value
type SolClientFlowRxMsgDispatchFuncInfo = C.solClient_flow_rxMsgDispatchFuncInfo_t

// Callbacks

// SolClientFlowMessageCallback is assigned a function
type SolClientFlowMessageCallback = func(msgP SolClientMessagePt) bool

// SolClientFlowEventCallback is assigned a function
type SolClientFlowEventCallback = func(flowEvent SolClientFlowEvent, responseCode SolClientResponseCode, info string)

var flowID uintptr
var flowToEventCallbackMap sync.Map
var flowToRXCallbackMap sync.Map

//export goFlowMessageReceiveCallback
func goFlowMessageReceiveCallback(flowP SolClientFlowPt, msgP SolClientMessagePt, userP unsafe.Pointer) C.solClient_rxMsgCallback_returnCode_t {
	if callback, ok := flowToRXCallbackMap.Load(uintptr(userP)); ok {
		if callback.(SolClientFlowMessageCallback)(msgP) {
			return C.SOLCLIENT_CALLBACK_TAKE_MSG
		}
		return C.SOLCLIENT_CALLBACK_OK
	}
	logging.Default.Error("Received message from core API without an associated session callback")
	return C.SOLCLIENT_CALLBACK_OK
}

//export goFlowEventCallback
func goFlowEventCallback(flowP SolClientFlowPt, eventInfoP SolClientFlowEventInfoPt, userP unsafe.Pointer) {
	if callback, ok := flowToEventCallbackMap.Load(uintptr(userP)); ok {
		callback.(SolClientFlowEventCallback)(SolClientFlowEvent(eventInfoP.flowEvent), eventInfoP.responseCode, C.GoString(eventInfoP.info_p))
	} else {
		logging.Default.Debug("Received event callback from core API without an associated session callback")
	}
}

// Type definitions

// SolClientFlow structure
type SolClientFlow struct {
	pointer SolClientFlowPt
	userP   uintptr
}

// Functionality

// SolClientSessionCreateFlow function
func (session *SolClientSession) SolClientSessionCreateFlow(properties []string, msgCallback SolClientFlowMessageCallback, eventCallback SolClientFlowEventCallback) (*SolClientFlow, *SolClientErrorInfoWrapper) {
	flowPropsP, sessionPropertiesFreeFunction := ToCArray(properties, true)
	defer sessionPropertiesFreeFunction()

	var flowCreateFuncInfo C.solClient_flow_createFuncInfo_t

	flowID := atomic.AddUintptr(&flowID, 1)

	// These are not a misuse of unsafe.Pointer as the value is used for correlation
	flowCreateFuncInfo.rxMsgInfo.callback_p = (C.solClient_flow_rxMsgCallbackFunc_t)(unsafe.Pointer(C.flowMessageReceiveCallback))
	flowCreateFuncInfo.rxMsgInfo.user_p = C.uintptr_to_void_p(C.solClient_uint64_t(flowID))
	flowCreateFuncInfo.eventInfo.callback_p = (C.solClient_flow_eventCallbackFunc_t)(unsafe.Pointer(C.flowEventCallback))
	flowCreateFuncInfo.eventInfo.user_p = C.uintptr_to_void_p(C.solClient_uint64_t(flowID))

	flowToRXCallbackMap.Store(flowID, msgCallback)
	flowToEventCallbackMap.Store(flowID, eventCallback)

	flow := &SolClientFlow{}
	flow.userP = flowID
	err := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_session_createFlow(flowPropsP, session.pointer, &flow.pointer, &flowCreateFuncInfo, (C.size_t)(unsafe.Sizeof(flowCreateFuncInfo)))
	})
	if err != nil {
		return nil, err
	}
	return flow, nil
}

// SolClientFlowRemoveCallbacks function
func (flow *SolClientFlow) SolClientFlowRemoveCallbacks() {
	flowToRXCallbackMap.Delete(flow.userP)
	flowToEventCallbackMap.Delete(flow.userP)
}

// SolClientFlowDestroy function
func (flow *SolClientFlow) SolClientFlowDestroy() *SolClientErrorInfoWrapper {
	errInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_flow_destroy(&flow.pointer)
	})
	return errInfo
}

// SolClientFlowStart function
func (flow *SolClientFlow) SolClientFlowStart() *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_flow_start(flow.pointer)
	})
}

// SolClientFlowStop function
func (flow *SolClientFlow) SolClientFlowStop() *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_flow_stop(flow.pointer)
	})
}

// SolClientFlowSubscribe function
func (flow *SolClientFlow) SolClientFlowSubscribe(topic string, correlationID uintptr) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		cString := C.CString(topic)
		defer C.free(unsafe.Pointer(cString))
		// This is not an unsafe usage of unsafe.Pointer as we are using correlationId as data, not as a pointer
		return C.solClient_flow_topicSubscribeWithDispatch(flow.pointer, C.SOLCLIENT_SUBSCRIBE_FLAGS_REQUEST_CONFIRM, cString, nil, C.uintptr_to_void_p(C.solClient_uint64_t(correlationID)))
	})
}

// SolClientFlowUnsubscribe function
func (flow *SolClientFlow) SolClientFlowUnsubscribe(topic string, correlationID uintptr) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		cString := C.CString(topic)
		defer C.free(unsafe.Pointer(cString))
		// This is not an unsafe usage of unsafe.Pointer as we are using correlationId as data, not as a pointer
		return C.solClient_flow_topicUnsubscribeWithDispatch(flow.pointer, C.SOLCLIENT_SUBSCRIBE_FLAGS_REQUEST_CONFIRM, cString, nil, C.uintptr_to_void_p(C.solClient_uint64_t(correlationID)))
	})
}

// SolClientFlowAck function
func (flow *SolClientFlow) SolClientFlowAck(msgID SolClientMessageID) *SolClientErrorInfoWrapper {
	return handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_flow_sendAck(flow.pointer, msgID)
	})
}

// SolClientFlowGetDestination function
func (flow *SolClientFlow) SolClientFlowGetDestination() (name string, durable bool, err *SolClientErrorInfoWrapper) {
	var dest *SolClientDestination = &SolClientDestination{}
	errorInfo := handleCcsmpError(func() SolClientReturnCode {
		return C.solClient_flow_getDestination(flow.pointer, dest, (C.size_t)(unsafe.Sizeof(*dest)))
	})
	if errorInfo != nil {
		return "", false, errorInfo
	}
	name = C.GoString(dest.dest)
	durable = dest.destType != C.SOLCLIENT_QUEUE_TEMP_DESTINATION
	return name, durable, errorInfo
}
