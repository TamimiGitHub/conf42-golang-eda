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

package core

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/internal/impl/logging"
	"solace.dev/go/messaging/pkg/solace"
)

// SubscriptionCorrelationID defined
type SubscriptionCorrelationID = uintptr

// SubscriptionEvent is the event passed to a channel on completion of a subscription
type SubscriptionEvent interface {
	GetID() SubscriptionCorrelationID
	GetError() error
}

// Receiver interface
type Receiver interface {
	// checks if the internal receiver is running
	IsRunning() bool
	// Events returns SolClientEvents
	Events() Events
	// Register an RX callback, returns a correlation pointer used when adding and removing subscriptions
	RegisterRXCallback(msgCallback RxCallback) uintptr
	// Remove the callback allowing GC to cleanup the function registered
	UnregisterRXCallback(ptr uintptr)
	// Add a subscription to the given correlation pointer
	Subscribe(topic string, ptr uintptr) (SubscriptionCorrelationID, <-chan SubscriptionEvent, ErrorInfo)
	// Remove a subscription from the given correlation pointer
	Unsubscribe(topic string, ptr uintptr) (SubscriptionCorrelationID, <-chan SubscriptionEvent, ErrorInfo)
	// Clears the subscription correlation with the given ID
	ClearSubscriptionCorrelation(id SubscriptionCorrelationID)
	// ProvisionEndpoint will provision an endpoint
	ProvisionEndpoint(queueName string, isExclusive bool) ErrorInfo
	// EndpointUnsubscribe will call endpoint unsubscribe on the endpoint
	EndpointUnsubscribe(queueName string, topic string) (SubscriptionCorrelationID, <-chan SubscriptionEvent, ErrorInfo)
	// Increments receiver metrics
	IncrementMetric(metric NextGenMetric, amount uint64)
	// Creates a new persistent receiver with the given callback
	NewPersistentReceiver(properties []string, callback RxCallback, eventCallback PersistentEventCallback) (PersistentReceiver, ErrorInfo)
}

// PersistentReceiver interface
type PersistentReceiver interface {
	// Destroy destroys the flow
	Destroy(freeMemory bool) ErrorInfo
	// Start will start the receiption of messages
	// Persistent Receivers are started by default after creation
	Start() ErrorInfo
	// Stop will stop the reception of messages
	Stop() ErrorInfo
	// Subscribe will add a subscription to the persistent receiver
	Subscribe(topic string) (SubscriptionCorrelationID, <-chan SubscriptionEvent, ErrorInfo)
	// Unsubscribe will remove the subscription from the persistent receiver
	Unsubscribe(topic string) (SubscriptionCorrelationID, <-chan SubscriptionEvent, ErrorInfo)
	// Ack will acknowledge the given message
	Ack(msgID MessageID) ErrorInfo
	// Destionation returns the destination if retrievable, or an error if one occurred
	Destination() (destination string, durable bool, errorInfo ErrorInfo)
}

// Receivable type defined
type Receivable = ccsmp.SolClientMessagePt

// MessageID type defined
type MessageID = ccsmp.SolClientMessageID

// RxCallback type defined
type RxCallback func(msg Receivable) bool

// PersistentEventCallback type defined
type PersistentEventCallback func(event ccsmp.SolClientFlowEvent, eventInfo FlowEventInfo)

// FlowEventInfo interface
type FlowEventInfo interface {
	EventInfo
}

// Implementation
type ccsmpBackedReceiver struct {
	events  *ccsmpBackedEvents
	metrics *ccsmpBackedMetrics
	session *ccsmp.SolClientSession
	running int32
	// TODO if performance becomes a concern, consider substituting maps and mutex for sync.Map
	rxLock      sync.RWMutex
	rxMap       map[uintptr]RxCallback
	dispatchMap map[uintptr]*ccsmp.SolClientSessionRxMsgDispatchFuncInfo
	dispatchID  uint64
	//
	subscriptionCorrelationLock sync.Mutex
	subscriptionCorrelation     map[SubscriptionCorrelationID]chan SubscriptionEvent
	subscriptionCorrelationID   SubscriptionCorrelationID

	subscriptionOkEvent, subscriptionErrorEvent uint
}

func newCcsmpReceiver(session *ccsmp.SolClientSession, events *ccsmpBackedEvents, metrics *ccsmpBackedMetrics) *ccsmpBackedReceiver {
	receiver := &ccsmpBackedReceiver{}
	receiver.events = events
	receiver.metrics = metrics
	receiver.session = session
	receiver.running = 0
	receiver.rxMap = make(map[uintptr]RxCallback)
	receiver.dispatchMap = make(map[uintptr]*ccsmp.SolClientSessionRxMsgDispatchFuncInfo)
	receiver.dispatchID = 0
	receiver.subscriptionCorrelation = make(map[SubscriptionCorrelationID]chan SubscriptionEvent)
	receiver.subscriptionCorrelationID = 0
	return receiver
}

func (receiver *ccsmpBackedReceiver) start() {
	if !atomic.CompareAndSwapInt32(&receiver.running, 0, 1) {
		return
	}
	receiver.session.SetMessageCallback(receiver.rxCallback)
	receiver.subscriptionOkEvent = receiver.events.AddEventHandler(SolClientSubscriptionOk, receiver.handleSubscriptionOK)
	receiver.subscriptionErrorEvent = receiver.events.AddEventHandler(SolClientSubscriptionError, receiver.handleSubscriptionErr)
}

func (receiver *ccsmpBackedReceiver) terminate() {
	if !atomic.CompareAndSwapInt32(&receiver.running, 1, 0) {
		return
	}
	receiver.session.SetMessageCallback(nil)
	receiver.events.RemoveEventHandler(receiver.subscriptionOkEvent)
	receiver.events.RemoveEventHandler(receiver.subscriptionErrorEvent)
	// Clear out the subscription correlation map when terminating, interrupt all awaiting items
	receiver.subscriptionCorrelationLock.Lock()
	defer receiver.subscriptionCorrelationLock.Unlock()
	terminationError := solace.NewError(&solace.ServiceUnreachableError{}, constants.CouldNotConfirmSubscriptionServiceUnavailable, nil)
	for id, result := range receiver.subscriptionCorrelation {
		delete(receiver.subscriptionCorrelation, id)
		result <- &subscriptionEvent{
			id:  id,
			err: terminationError,
		}
	}
}

// checks if the internal receiver is running
func (receiver *ccsmpBackedReceiver) IsRunning() bool {
	return atomic.LoadInt32(&receiver.running) == 1
}

func (receiver *ccsmpBackedReceiver) Events() Events {
	return receiver.events
}

func (receiver *ccsmpBackedReceiver) rxCallback(msg Receivable, userP unsafe.Pointer) bool {
	receiver.rxLock.RLock()
	defer receiver.rxLock.RUnlock()
	callback, ok := receiver.rxMap[uintptr(userP)]
	if !ok {
		if logging.Default.IsDebugEnabled() {
			logging.Default.Debug(fmt.Sprintf("receive callback called but no receive function is registered for user pointer %v", userP))
		}
		return false
	}
	return callback(msg)
}

// Register an RX callback, returns a correlation pointer used when adding and removing subscriptions
func (receiver *ccsmpBackedReceiver) RegisterRXCallback(msgCallback RxCallback) uintptr {
	dispatch, dispatchPointer := ccsmp.NewSessionDispatch(atomic.AddUint64(&receiver.dispatchID, 1))
	receiver.rxLock.Lock()
	defer receiver.rxLock.Unlock()
	receiver.dispatchMap[dispatchPointer] = dispatch
	receiver.rxMap[dispatchPointer] = msgCallback
	return dispatchPointer
}

// Remove the callback allowing GC to cleanup the function registered
func (receiver *ccsmpBackedReceiver) UnregisterRXCallback(ptr uintptr) {
	receiver.rxLock.Lock()
	defer receiver.rxLock.Unlock()
	delete(receiver.dispatchMap, ptr)
	delete(receiver.rxMap, ptr)
}

// Add a subscription to the given correlation pointer
func (receiver *ccsmpBackedReceiver) Subscribe(topic string, ptr uintptr) (SubscriptionCorrelationID, <-chan SubscriptionEvent, ErrorInfo) {
	receiver.rxLock.RLock()
	defer receiver.rxLock.RUnlock()
	dispatch := receiver.dispatchMap[ptr]
	id, c := receiver.newSubscriptionCorrelation()
	errInfo := receiver.session.SolClientSessionSubscribe(topic, dispatch, id)
	if errInfo != nil {
		receiver.ClearSubscriptionCorrelation(id)
		return 0, nil, errInfo
	}
	return id, c, nil
}

// Remove a subscription from the given correlation pointer
func (receiver *ccsmpBackedReceiver) Unsubscribe(topic string, ptr uintptr) (SubscriptionCorrelationID, <-chan SubscriptionEvent, ErrorInfo) {
	receiver.rxLock.RLock()
	defer receiver.rxLock.RUnlock()
	dispatch := receiver.dispatchMap[ptr]
	id, c := receiver.newSubscriptionCorrelation()
	errInfo := receiver.session.SolClientSessionUnsubscribe(topic, dispatch, id)
	if errInfo != nil {
		receiver.ClearSubscriptionCorrelation(id)
		return 0, nil, errInfo
	}
	return id, c, nil
}

func (receiver *ccsmpBackedReceiver) ProvisionEndpoint(queueName string, isExclusive bool) ErrorInfo {
	properties := getEndpointProperties(queueName)
	// Set if we are exclusive or not for provisioning, not needed for other cases
	properties = append(properties, ccsmp.SolClientEndpointPropAccesstype)
	if isExclusive {
		properties = append(properties, ccsmp.SolClientEndpointPropAccesstypeExclusive)
	} else {
		properties = append(properties, ccsmp.SolClientEndpointPropAccesstypeNonexclusive)
	}
	return receiver.session.SolClientEndpointProvision(properties)
}

// Unsubscribe will remove the subscription from the persistent receiver
func (receiver *ccsmpBackedReceiver) EndpointUnsubscribe(queueName string, topic string) (SubscriptionCorrelationID, <-chan SubscriptionEvent, ErrorInfo) {
	properties := getEndpointProperties(queueName)
	id, c := receiver.newSubscriptionCorrelation()
	errInfo := receiver.session.SolClientEndpointUnsusbcribe(properties, topic, id)
	if errInfo != nil {
		receiver.ClearSubscriptionCorrelation(id)
		return 0, nil, errInfo
	}
	return id, c, nil
}

func getEndpointProperties(queueName string) []string {
	return []string{
		ccsmp.SolClientEndpointPropID, ccsmp.SolClientEndpointPropQueue,
		ccsmp.SolClientEndpointPropDurable, ccsmp.SolClientPropEnableVal,
		ccsmp.SolClientEndpointPropName, queueName,
	}
}

func (receiver *ccsmpBackedReceiver) IncrementMetric(metric NextGenMetric, amount uint64) {
	receiver.metrics.IncrementMetric(metric, amount)
}

func (receiver *ccsmpBackedReceiver) ClearSubscriptionCorrelation(id SubscriptionCorrelationID) {
	receiver.subscriptionCorrelationLock.Lock()
	defer receiver.subscriptionCorrelationLock.Unlock()
	delete(receiver.subscriptionCorrelation, id)
}

func (receiver *ccsmpBackedReceiver) newSubscriptionCorrelation() (SubscriptionCorrelationID, chan SubscriptionEvent) {
	newID := atomic.AddUintptr(&receiver.subscriptionCorrelationID, 1)
	// we always want to have a space of 1 in the channel so we don't even block
	resultChan := make(chan SubscriptionEvent, 1)
	receiver.subscriptionCorrelationLock.Lock()
	receiver.subscriptionCorrelation[newID] = resultChan
	receiver.subscriptionCorrelationLock.Unlock()
	return newID, resultChan
}

func (receiver *ccsmpBackedReceiver) handleSubscriptionOK(event SessionEventInfo) {
	receiver.handleSubscriptionEvent(event, nil)
}

func (receiver *ccsmpBackedReceiver) handleSubscriptionErr(event SessionEventInfo) {
	receiver.handleSubscriptionEvent(event, event.GetError())
}

func (receiver *ccsmpBackedReceiver) handleSubscriptionEvent(event SessionEventInfo, err error) {
	receiver.subscriptionCorrelationLock.Lock()
	defer receiver.subscriptionCorrelationLock.Unlock()
	corrP := uintptr(event.GetCorrelationPointer())
	if resultChan, ok := receiver.subscriptionCorrelation[corrP]; ok {
		delete(receiver.subscriptionCorrelation, corrP)
		resultChan <- &subscriptionEvent{
			id:  corrP,
			err: err,
		}
	}
}

type subscriptionEvent struct {
	id  SubscriptionCorrelationID
	err error
}

func (event *subscriptionEvent) GetID() SubscriptionCorrelationID {
	return event.id
}

func (event *subscriptionEvent) GetError() error {
	return event.err
}

type ccsmpBackedPersistentReceiver struct {
	flow   *ccsmp.SolClientFlow
	parent *ccsmpBackedReceiver
}

func (receiver *ccsmpBackedReceiver) NewPersistentReceiver(properties []string, rxCallback RxCallback, eventCallback PersistentEventCallback) (PersistentReceiver, ErrorInfo) {
	if rxCallback == nil || eventCallback == nil {
		logging.Default.Debug("attempted to create a new receiver with nil callbacks")
		return nil, nil
	}
	flowMsgCallback := func(msgP ccsmp.SolClientMessagePt) bool {
		return rxCallback(msgP)
	}
	flowEventCallback := func(flowEvent ccsmp.SolClientFlowEvent, responseCode ccsmp.SolClientResponseCode, info string) {
		lastErrorInfo := ccsmp.GetLastErrorInfo(0)
		var err error
		if lastErrorInfo.SubCode != ccsmp.SolClientSubCodeOK {
			err = ToNativeError(lastErrorInfo)
		}
		eventCallback(flowEvent, &flowEventInfo{err: err, infoString: info})
	}
	flow, err := receiver.session.SolClientSessionCreateFlow(properties, flowMsgCallback, flowEventCallback)
	if err != nil {
		return nil, err
	}
	return &ccsmpBackedPersistentReceiver{
		flow:   flow,
		parent: receiver,
	}, nil
}

// Destroy destroys the flow
func (receiver *ccsmpBackedPersistentReceiver) Destroy(freeMemory bool) ErrorInfo {
	var err ErrorInfo
	if freeMemory {
		err = receiver.flow.SolClientFlowDestroy()
	}
	receiver.flow.SolClientFlowRemoveCallbacks()
	return err
}

// Start will start the receiption of messages
// Persistent Receivers are started by default after creation
func (receiver *ccsmpBackedPersistentReceiver) Start() ErrorInfo {
	return receiver.flow.SolClientFlowStart()
}

// Stop will stop the reception of messages
func (receiver *ccsmpBackedPersistentReceiver) Stop() ErrorInfo {
	return receiver.flow.SolClientFlowStop()
}

// Subscribe will add a subscription to the persistent receiver
func (receiver *ccsmpBackedPersistentReceiver) Subscribe(topic string) (SubscriptionCorrelationID, <-chan SubscriptionEvent, ErrorInfo) {
	id, c := receiver.parent.newSubscriptionCorrelation()
	errInfo := receiver.flow.SolClientFlowSubscribe(topic, id)
	if errInfo != nil {
		receiver.parent.ClearSubscriptionCorrelation(id)
		return 0, nil, errInfo
	}
	return id, c, nil
}

// Unsubscribe will remove the subscription from the persistent receiver
func (receiver *ccsmpBackedPersistentReceiver) Unsubscribe(topic string) (SubscriptionCorrelationID, <-chan SubscriptionEvent, ErrorInfo) {
	id, c := receiver.parent.newSubscriptionCorrelation()
	errInfo := receiver.flow.SolClientFlowUnsubscribe(topic, id)
	if errInfo != nil {
		receiver.parent.ClearSubscriptionCorrelation(id)
		return 0, nil, errInfo
	}
	return id, c, nil
}

func (receiver *ccsmpBackedPersistentReceiver) Ack(msgID MessageID) ErrorInfo {
	return receiver.flow.SolClientFlowAck(msgID)
}

func (receiver *ccsmpBackedPersistentReceiver) Destination() (destination string, durable bool, errorInfo ErrorInfo) {
	return receiver.flow.SolClientFlowGetDestination()
}

type flowEventInfo struct {
	err        error
	infoString string
}

func (info *flowEventInfo) GetError() error {
	return info.err
}

func (info *flowEventInfo) GetInfoString() string {
	return info.infoString
}

// we use the user pointer internally to correlate, it should NOT be used anywhere else
func (info *flowEventInfo) GetUserPointer() unsafe.Pointer {
	return nil
}
