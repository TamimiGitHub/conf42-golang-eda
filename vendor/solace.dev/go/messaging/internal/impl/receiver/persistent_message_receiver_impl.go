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

// Package receiver ise defined below
package receiver

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/internal/impl/executor"
	"solace.dev/go/messaging/internal/impl/logging"
	"solace.dev/go/messaging/internal/impl/message"
	"solace.dev/go/messaging/internal/impl/validation"
	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/config"
	apimessage "solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/message/rgmid"
	"solace.dev/go/messaging/pkg/solace/resource"
	"solace.dev/go/messaging/pkg/solace/subcode"
)

type persistentMessageReceiverImpl struct {
	basicMessageReceiver

	logger logging.LogLevelLogger

	internalFlowStopped    int32
	internalFlow           core.PersistentReceiver
	internalFlowProperties []string

	queue                    *resource.Queue
	doCreateMissingResources bool

	doAutoAck bool

	stateChangeListener solace.ReceiverStateChangeListener

	subscriptionsLock           sync.Mutex
	subscriptionTerminationLock sync.RWMutex
	subscriptions               []string

	// we want to synchronize calls to subscribe/unsubscribe to avoid crashing due to thread limitations
	subscriptionsSynchronizationLock sync.Mutex

	outstandingSubscriptionEventsLock sync.Mutex
	outstandingSubscriptionEvents     map[core.SubscriptionCorrelationID]struct{}

	highwater, lowwater int
	buffer              chan ccsmp.SolClientMessagePt
	bufferClosed        int32

	doPause        int32
	doUnpause      chan interface{}
	pauseInterrupt chan interface{}

	bufferEmptyOnTerminateFlag int32
	bufferEmptyOnTerminate     chan struct{}

	terminationNotification chan struct{}
	terminationComplete     chan struct{}

	rxCallbackSet chan bool
	rxCallback    unsafe.Pointer

	terminationHandlerID uint

	eventExecutor executor.Executor
}

type persistentMessageReceiverProps struct {
	flowProperties                     []string
	internalReceiver                   core.Receiver
	startupSubscriptions               []resource.Subscription
	endpoint                           *resource.Queue
	bufferHighwater, bufferLowwater    int
	doCreateMissingResource, doAutoAck bool
	stateChangeListener                solace.ReceiverStateChangeListener
}

func (receiver *persistentMessageReceiverImpl) construct(props *persistentMessageReceiverProps) {
	receiver.basicMessageReceiver.construct(props.internalReceiver)
	receiver.eventExecutor = executor.NewExecutor()
	receiver.internalFlowProperties = props.flowProperties

	receiver.subscriptions = make([]string, len(props.startupSubscriptions))
	for i, subscription := range props.startupSubscriptions {
		receiver.subscriptions[i] = subscription.GetName()
	}

	const maxWindowSize = 255
	// keep some extra space in the buffer so that we can handle spikes and slowdowns
	bufferSize := 2*maxWindowSize + 2*receiver.highwater
	receiver.highwater = props.bufferHighwater
	receiver.lowwater = props.bufferLowwater
	receiver.buffer = make(chan ccsmp.SolClientMessagePt, bufferSize)
	receiver.bufferClosed = 0

	receiver.terminationNotification = make(chan struct{})
	receiver.terminationComplete = make(chan struct{})

	receiver.logger = logging.For(receiver)

	receiver.queue = props.endpoint
	receiver.doCreateMissingResources = props.doCreateMissingResource
	receiver.doAutoAck = props.doAutoAck

	receiver.stateChangeListener = props.stateChangeListener

	receiver.bufferEmptyOnTerminateFlag = 0
	receiver.bufferEmptyOnTerminate = make(chan struct{})

	atomic.StorePointer(&receiver.rxCallback, nil)
	receiver.rxCallbackSet = make(chan bool, 1)

	receiver.doPause = 0
	// These both are of size 1 such that at most one event is outstanding
	receiver.doUnpause = make(chan interface{}, 1)
	receiver.pauseInterrupt = make(chan interface{}, 1)

	receiver.outstandingSubscriptionEvents = make(map[core.SubscriptionCorrelationID]struct{})
}

func (receiver *persistentMessageReceiverImpl) onDownEvent(eventInfo core.SessionEventInfo) {
	receiver.logger.Debug("Received session event down error! Terminating...")
	// terminate immediately and skip subscription deregistration as the connection is dead at this point
	go receiver.unsolicitedTermination(eventInfo, false)
	// we never clean up native memory on down events aside from messages. This is because we assume that the session is destroyed
	// and thus any other native memory (aside from messages) will be destroyed as well
}

func (receiver *persistentMessageReceiverImpl) onFlowEvent(event ccsmp.SolClientFlowEvent, eventInfo core.FlowEventInfo) {
	switch event {
	case ccsmp.SolClientFlowEventUpNotice:
		// noop
		receiver.logger.Debug("Received flow event up notice")
	case ccsmp.SolClientFlowEventDownError:
		// handle case of down, this is unrecoverable. we also want to clean up native memory in this case
		receiver.logger.Debug("Received flow event down error! Terminating...")
		go receiver.unsolicitedTermination(eventInfo, true)
	case ccsmp.SolClientFlowEventBindFailedError:
		// TODO what do we want to do in this case?
		receiver.logger.Debug("Received flow event bind failed! Info string: " + eventInfo.GetInfoString())
	case ccsmp.SolClientFlowEventActive:
		receiver.logger.Debug("Received flow event active")
		// call activation passivation support
		timestamp := time.Now()
		listener := receiver.stateChangeListener
		if listener != nil {
			receiver.eventExecutor.Submit(func() {
				listener(solace.ReceiverPassive, solace.ReceiverActive, timestamp)
			})
		}
	case ccsmp.SolClientFlowEventInactive:
		receiver.logger.Debug("Received flow event inactive")
		// call activation passivation support
		timestamp := time.Now()
		listener := receiver.stateChangeListener
		if listener != nil {
			receiver.eventExecutor.Submit(func() {
				listener(solace.ReceiverActive, solace.ReceiverPassive, timestamp)
			})
		}
	case ccsmp.SolClientFlowEventReconnecting:
		// we are reconnecting, nothing to do
		receiver.logger.Debug("Received flow event reconnecting")
	case ccsmp.SolClientFlowEventReconnected:
		// we are reconnected, nothing to do
		receiver.logger.Debug("Received flow event reconnected")
	case ccsmp.SolClientFlowEventSessionDown:
		// the session has gone down , potentially because of reconnection
		// we ignore this event and rely on the session down events to tell us what to do
		receiver.logger.Debug("Received flow event session down")
	case ccsmp.SolClientFlowEventRejectedMsgError:
		// Never raised, ignore
	}
}

// Start will start the service synchronously.
// Before this function is called, the service is considered
// off-duty. To operate normally, this function must be called on
// a receiver or publisher instance. This function is idempotent.
// Returns an error if one occurred or nil if successful.
func (receiver *persistentMessageReceiverImpl) Start() (err error) {
	// this will block until we are started if we are not first
	if proceed, err := receiver.starting(); !proceed {
		return err
	}
	receiver.logger.Debug("Start receiver start")
	defer func() {
		if err == nil {
			receiver.started(err)
			receiver.logger.Debug("Start receiver complete")
		} else {
			receiver.logger.Debug("Start receiver complete with error: " + err.Error())
			receiver.internalReceiver.Events().RemoveEventHandler(receiver.terminationHandlerID)
			if receiver.internalFlow != nil {
				receiver.internalFlow.Destroy(true)
			}
			receiver.terminated(nil)
			receiver.startFuture.Complete(err)
		}
	}()
	// if we get an error from provisioning the endpoint, we should not continue
	// we don't need to clean up in this case as we don't support deprovisioning
	err = receiver.provisionEndpoint()
	if err != nil {
		return err
	}
	receiver.terminationHandlerID = receiver.internalReceiver.Events().AddEventHandler(core.SolClientEventDown, receiver.onDownEvent)
	var errInfoWrapper core.ErrorInfo
	receiver.internalFlow, errInfoWrapper = receiver.internalReceiver.NewPersistentReceiver(receiver.internalFlowProperties, receiver.messageCallback, receiver.onFlowEvent)
	if errInfoWrapper != nil {
		return core.ToNativeError(errInfoWrapper, "error while creating receiver flow: ")
	}
	// After we've allocated the flow, lets start it, SOL-63525
	errInfoWrapper = receiver.internalFlow.Start()
	if errInfoWrapper != nil {
		return core.ToNativeError(errInfoWrapper, "error while starting receiver flow: ")
	}
	subscriptionResults := make([]<-chan core.SubscriptionEvent, len(receiver.subscriptions))
	outstandingCorrelations := []core.SubscriptionCorrelationID{}
	defer func() {
		// if we don't cleanup, when we destroy the flow we might orphan entries in the map causing a memory leak
		if err != nil {
			receiver.logger.Debug("Encountered error while adding subscriptions, removoing outstanding correlations")
			for _, id := range outstandingCorrelations {
				receiver.internalReceiver.ClearSubscriptionCorrelation(id)
			}
		}
	}()
	for i, subscription := range receiver.subscriptions {
		id, result, errInfo := receiver.internalFlow.Subscribe(subscription)
		if errInfo != nil {
			err = core.ToNativeError(errInfo, constants.FailedToAddSubscription)
			return err
		}
		subscriptionResults[i] = result
		outstandingCorrelations = append(outstandingCorrelations, id)
	}
	for _, result := range subscriptionResults {
		if result != nil {
			event := <-result
			if event.GetError() != nil {
				return event.GetError()
			}
		}
	}
	go receiver.eventExecutor.Run()
	go receiver.run()
	return nil
}

func (receiver *persistentMessageReceiverImpl) provisionEndpoint() error {
	if receiver.doCreateMissingResources && receiver.queue.IsDurable() {
		errInfo := receiver.internalReceiver.ProvisionEndpoint(receiver.queue.GetName(), receiver.queue.IsExclusivelyAccessible())
		if errInfo != nil {
			if subcode.Code(errInfo.SubCode) == subcode.EndpointAlreadyExists {
				receiver.logger.Info("Endpoint '" + receiver.queue.GetName() + "' already exists")
			} else {
				receiver.logger.Warning("Failed to provision endpoint '" + receiver.queue.GetName() + "', " + errInfo.GetMessageAsString())
				return core.ToNativeError(errInfo)
			}
		} else {
			receiver.logger.Info("Endpoint '" + receiver.queue.GetName() + "' provisioned successfully")
		}
	}
	return nil
}

// StartAsync will start the service asynchronously.
// Before this function is called, the service is considered
// off-duty. To operate normally, this function must be called on
// a receiver or publisher instance. This function is idempotent.
// Returns a channel that will receive an error if one occurred or
// nil if successful. Subsequent calls will return additional
// channels that can await an error, or nil if already started.
func (receiver *persistentMessageReceiverImpl) StartAsync() <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- receiver.Start()
		close(result)
	}()
	return result
}

// StartAsyncCallback will start the PersistentMessageReceiver asynchronously.
// Calls the callback when started with an error if one occurred or nil
// if successful.
func (receiver *persistentMessageReceiverImpl) StartAsyncCallback(callback func(solace.PersistentMessageReceiver, error)) {
	go func() {
		callback(receiver, receiver.Start())
	}()
}

// Terminate will terminate the service gracefully and synchronously.
// This function is idempotent. The only way to resume operation
// after this function is called is to create a new instance.
// Any attempt to call this function renders the instance
// permanently terminated, even if this function completes.
// A graceful shutdown will be attempted within the grace period.
// A grace period of 0 implies a non-graceful shutdown that ignores
// unfinished tasks or in-flight messages.
// This function blocks until the service is terminated.
// If gracePeriod is less than 0, the function will wait indefinitely.
func (receiver *persistentMessageReceiverImpl) Terminate(gracePeriod time.Duration) (err error) {
	if proceed, err := receiver.basicMessageReceiver.terminate(); !proceed {
		return err
	}
	receiver.logger.Debug("Terminate receiver start")
	// We must mutex protect termination as subscriptions must NOT be added after we have begun removing them.
	defer func() {
		receiver.terminated(err)
		if err != nil {
			receiver.logger.Debug("Terminate receiver complete with error: " + err.Error())
		} else {
			receiver.logger.Debug("Terminate receiver complete")
		}
	}()
	defer func() {
		// Last thing we want to do is destroy the flow
		errInfoWrapper := receiver.internalFlow.Destroy(true)
		if errInfoWrapper != nil {
			// We should not encounter issues destroying the flow
			receiver.logger.Info("Encountered error while trying to clean up receiver flow: " + errInfoWrapper.String())
		}
	}()
	defer func() {
		// we still might get subscription responses so we'll delay as long as possible
		receiver.outstandingSubscriptionEventsLock.Lock()
		defer receiver.outstandingSubscriptionEventsLock.Unlock()
		for k := range receiver.outstandingSubscriptionEvents {
			receiver.internalReceiver.ClearSubscriptionCorrelation(k)
			delete(receiver.outstandingSubscriptionEvents, k)
		}
	}()
	// First set the internal flow stopped to terminating state, this way we will never resume flow
	// as we must do an atomic compare and set to call start or stop.
	// Note that there is a potential race here where we change the flow stopped state after the check and
	// call to flow start, however messages received while in the terminating state are dropped as termination
	// discarded, so that case should be handled.
	atomic.StoreInt32(&receiver.internalFlowStopped, 2)
	// Stop receiving new messages
	if errInfoWrapper := receiver.internalFlow.Stop(); err != nil {
		receiver.logger.Error("Encountered error while stopping flow: " + errInfoWrapper.String())
	}
	// Remove the termination event handler
	receiver.internalReceiver.Events().RemoveEventHandler(receiver.terminationHandlerID)
	defer func() {
		receiver.logger.Debug("Awaiting termination of event executor")
		// Allow the event executor to terminate, blocking until it does
		receiver.eventExecutor.AwaitTermination()
	}()
	// Unblock the receiver dispatch routine telling it to not continue
	select {
	case receiver.rxCallbackSet <- false:
		// success
	default:
		// we do not want to block if there is already a queued notification
	}
	// Block any new messages from making it into the buffer
	// This may result in a panic in the rx callback that gets handled and logged
	close(receiver.buffer)
	// Note that this is a very particular ordering of operations.
	// We must first close the buffer such that no additional messages have been added,
	// Then we must set the bufferClosed flag, then we must check if the length of the buffer is 0.
	// This guarantees that any additional messages that are received by synchronous receive will
	// in fact check if bufferClosed is true, then notify of buffer empty on terminate. There is no
	// way for the flag to be set when messages still exist in the queue.
	atomic.StoreInt32(&receiver.bufferClosed, 1)

	// Check first if the buffer is empty, if it is then we can proceed with a successful termination
	// If it is greater than 0, then we are guaranteed to get a buffer empty notification either from
	// synchronous receive or the receiver dispatch thread
	if len(receiver.buffer) == 0 && atomic.CompareAndSwapInt32(&receiver.bufferEmptyOnTerminateFlag, 0, 1) {
		// Close the buffer empty notification and move on to terminating the receiver in select below
		close(receiver.bufferEmptyOnTerminate)
		// we need to do this check since we may not have the dispatch thread running. This means that
		// we need to check ourselves if the buffer is empty as it is possible that no more receive calls
		// are made. If the buffer is not empty at this point, it is guaranteed that on the next call
		// to receive sync (if there is no async callback set), we will notify of an empty buffer.
	}

	// Wait for the message receiver goroutine to shutdown. It may not shut down if the message handler is blocking indefinitely
	timer := time.NewTimer(gracePeriod)
	select {
	case <-timer.C:
		// timed out waiting for messages to be delivered
		close(receiver.terminationNotification)
		// join receiver thread
		<-receiver.terminationComplete
		undeliveredCount := receiver.drainQueue()
		// we may have terminated on the last message, in which case we were successful.
		if undeliveredCount > 0 {
			if receiver.logger.IsDebugEnabled() {
				receiver.logger.Debug(fmt.Sprintf("Receiver terminated with %d undelivered messages", undeliveredCount))
			}
			err := solace.NewError(&solace.IncompleteMessageDeliveryError{}, fmt.Sprintf(constants.IncompleteMessageReceptionMessage, undeliveredCount), nil)
			receiver.internalReceiver.IncrementMetric(core.MetricReceivedMessagesTerminationDiscarded, uint64(undeliveredCount))
			return err
		}
	case <-receiver.bufferEmptyOnTerminate:
		timer.Stop()
		// join receiver thread. we want to make sure that if we enter with 0 messages in the buffer but one message
		// is still being processed by the async callback, we will not terminate until that message callback is complete
		<-receiver.terminationComplete
	}
	return nil
}

func (receiver *persistentMessageReceiverImpl) unsolicitedTermination(eventInfo core.EventInfo, shouldCleanUpNative bool) {
	if proceed, _ := receiver.basicMessageReceiver.terminate(); !proceed {
		// we are already terminated, nothing to do
		return
	}
	receiver.logger.Debug("Received unsolicited termination with event info " + eventInfo.GetInfoString())
	defer receiver.logger.Debug("Unsolicited termination complete")
	timestamp := time.Now()
	// Clean up the flow and use shouldCleanupNative to determine whether or not to destroy the underlying flow
	// This is a hack to get around a bug in CCSMP where destroying a flow AFTER destroying the session cores
	receiver.internalFlow.Destroy(shouldCleanUpNative)
	// Remove the event handler
	receiver.internalReceiver.Events().RemoveEventHandler(receiver.terminationHandlerID)
	// Shutdown the event executor but do not wait for remaining events to be processed
	receiver.eventExecutor.Terminate()
	// Block any new messages from making it into the buffer
	// This may result in a panic in the rx callback that gets handled and logged
	close(receiver.buffer)
	atomic.StoreInt32(&receiver.bufferClosed, 1)

	// Unblock the receiver dispatch routine telling it to not continue
	select {
	case receiver.rxCallbackSet <- false:
		// success
	default:
		// we do not want to block if there is already a queued notification
	}
	// Shut down the receiver's termination notification
	close(receiver.terminationNotification)
	var err error = nil
	undeliveredCount := receiver.drainQueue()
	if undeliveredCount > 0 {
		if receiver.logger.IsDebugEnabled() {
			receiver.logger.Debug(fmt.Sprintf("Terminated with %d undelivered messages", undeliveredCount))
		}
		err = solace.NewError(&solace.IncompleteMessageDeliveryError{}, fmt.Sprintf(constants.IncompleteMessageReceptionMessage, undeliveredCount), nil)
		receiver.internalReceiver.IncrementMetric(core.MetricReceivedMessagesTerminationDiscarded, uint64(undeliveredCount))
	}
	// notify of termination with error, this will be retrievable with subsequent calls to "Terminate"
	receiver.terminated(err)
	// Call the callback
	if receiver.terminationListener != nil {
		receiver.terminationListener(&receiverTerminationEvent{
			timestamp,
			eventInfo.GetError(),
		})
	}
}

// drainQueue will drain out all remaining messages in the receiver buffer and will return the
// number of messages drained. There is a potential race between this function and the synchronous
// ReceiveMessage function whereby message order will be lost. This is expected behaviour as
// we are terminating ungracefully when drainQueue is called, thus there is no more guarantee of
// functionality. There are two potential workarounds if the race causes issues: 1. terminate gracefully
// and 2. use receive async.
func (receiver *persistentMessageReceiverImpl) drainQueue() uint64 {
	undeliveredCount := uint64(0)
	for msg := range receiver.buffer {
		undeliveredCount++
		ccsmp.SolClientMessageFree(&msg)
	}
	if atomic.CompareAndSwapInt32(&receiver.bufferEmptyOnTerminateFlag, 0, 1) {
		close(receiver.bufferEmptyOnTerminate)
	}
	return undeliveredCount
}

// TerminateAsync will terminate the service asynchronously.
// This function is idempotent. The only way to resume operation
// after this function is called is to create a new instance.
// Any attempt to call this function renders the instance
// permanently terminated, even if this function completes.
// A graceful shutdown will be attempted within the grace period.
// A grace period of 0 implies a non-graceful shutdown that ignores
// unfinished tasks or in-flight messages.
// Returns a channel that will receive an error if one occurred or
// nil if successfully and gracefully terminated.
// If gracePeriod is less than 0, the function will wait indefinitely.
func (receiver *persistentMessageReceiverImpl) TerminateAsync(gracePeriod time.Duration) <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- receiver.Terminate(gracePeriod)
		close(result)
	}()
	return result
}

// TerminateAsyncCallback will terminate the PersistentMessageReceiver asynchronously.
// Calls the callback when terminated with nil if successful or an error if
// one occurred. If gracePeriod is less than 0, the function will wait indefinitely.
func (receiver *persistentMessageReceiverImpl) TerminateAsyncCallback(gracePeriod time.Duration, callback func(error)) {
	go func() {
		callback(receiver.Terminate(gracePeriod))
	}()
}

// AddSubscription will subscribe to another message source on a PubSub+ Broker to receive messages from.
// Will block until subscription is added.
// Returns a solace/errors.*IllegalStateError if the service is not running.
// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
// Returns nil if successful.
func (receiver *persistentMessageReceiverImpl) AddSubscription(subscription resource.Subscription) error {
	// Check the state first un the event that we are currently holding the termination lock in the terminate function
	// This will fail much faster and will avoid hanging
	currentState := receiver.getState()
	if currentState != messageReceiverStateStarted {
		return solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToModifySubscriptionBadState, messageReceiverStateNames[currentState]), nil)
	}
	if err := checkPersistentMessageReceiverSubscriptionType(subscription); err != nil {
		return err
	}
	result, err := receiver.addSubscription(subscription)
	if err != nil {
		return err
	}
	if receiver.logger.IsDebugEnabled() {
		receiver.logger.Debug("AddSubscription awaiting confirm on subscription '" + subscription.GetName() + "'")
	}
	event := <-result
	if receiver.logger.IsDebugEnabled() {
		if event.GetError() != nil {
			receiver.logger.Debug("AddSubscription received error on subscription '" + subscription.GetName() + "': " + event.GetError().Error())
		} else {
			receiver.logger.Debug("AddSubscription received confirm on subscription '" + subscription.GetName() + "'")
		}
	}
	receiver.clearCorrelation(event)
	return event.GetError()
}

// common addSubscription without first check for state shared by sync and async
func (receiver *persistentMessageReceiverImpl) addSubscription(subscription resource.Subscription) (<-chan core.SubscriptionEvent, error) {
	// Acquire the termination lock such that we are not terminating over the course of subscription removal
	receiver.subscriptionTerminationLock.RLock()
	defer receiver.subscriptionTerminationLock.RUnlock()

	// Check the state again after acquiring the lock to make sure that we did not just terminate
	currentState := receiver.getState()
	if currentState != messageReceiverStateStarted {
		return nil, solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToModifySubscriptionBadState, messageReceiverStateNames[currentState]), nil)
	}

	if receiver.logger.IsDebugEnabled() {
		receiver.logger.Debug("addSubscription start with subscription " + subscription.GetName())
		defer receiver.logger.Debug("addSubscription end with subscription " + subscription.GetName())
	}

	topic := subscription.GetName()
	id, result, internalErr := receiver.subscribe(topic)
	if internalErr != nil {
		return nil, core.ToNativeError(internalErr)
	}
	receiver.addCorrelation(id)

	// Acquire the subscriptions lock only after it has been added in order to read and modify the list
	receiver.subscriptionsLock.Lock()
	defer receiver.subscriptionsLock.Unlock()
	// we must first check that we are not already subscribed to this topic
	for _, subscribedTopic := range receiver.subscriptions {
		if subscribedTopic == topic {
			return result, nil
		}
	}
	receiver.subscriptions = append(receiver.subscriptions, topic)
	return result, nil
}

func (receiver *persistentMessageReceiverImpl) subscribe(topic string) (core.SubscriptionCorrelationID, <-chan core.SubscriptionEvent, core.ErrorInfo) {
	receiver.subscriptionsSynchronizationLock.Lock()
	defer receiver.subscriptionsSynchronizationLock.Unlock()
	return receiver.internalFlow.Subscribe(topic)
}

// RemoveSubscription will unsubscribe from a previously subscribed message source on a broker
// such that no more messages will be received from it.
// Will block until subscription is removed.
// Returns an solace/errors.*IllegalStateError if the service is not running.
// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
// Returns nil if successful.
func (receiver *persistentMessageReceiverImpl) RemoveSubscription(subscription resource.Subscription) error {
	// Check the state first un the event that we are currently holding the termination lock in the terminate function
	// This will fail much faster and will avoid hanging
	currentState := receiver.getState()
	if currentState != messageReceiverStateStarted {
		return solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToModifySubscriptionBadState, messageReceiverStateNames[currentState]), nil)
	}
	if err := checkPersistentMessageReceiverSubscriptionType(subscription); err != nil {
		return err
	}
	result, err := receiver.removeSubscription(subscription)
	if err != nil {
		return err
	}
	if receiver.logger.IsDebugEnabled() {
		receiver.logger.Debug("RemoveSubscription awaiting confirm on subscription '" + subscription.GetName() + "'")
	}
	event := <-result
	if receiver.logger.IsDebugEnabled() {
		if event.GetError() != nil {
			receiver.logger.Debug("RemoveSubscription received error on subscription '" + subscription.GetName() + "': " + event.GetError().Error())
		} else {
			receiver.logger.Debug("RemoveSubscription received confirm on subscription '" + subscription.GetName() + "'")
		}
	}
	receiver.clearCorrelation(event)
	return event.GetError()
}

// common code without first check for state shared between sync and async remove subscriptions
func (receiver *persistentMessageReceiverImpl) removeSubscription(subscription resource.Subscription) (<-chan core.SubscriptionEvent, error) {
	// Acquire the termination lock such that we are not terminating over the course of subscription removal
	receiver.subscriptionTerminationLock.RLock()
	defer receiver.subscriptionTerminationLock.RUnlock()

	// Check the state again after acquiring the lock to make sure that we did not just terminate
	currentState := receiver.getState()
	if currentState != messageReceiverStateStarted {
		return nil, solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToModifySubscriptionBadState, messageReceiverStateNames[currentState]), nil)
	}

	if receiver.logger.IsDebugEnabled() {
		receiver.logger.Debug("removeSubscription start with subscription " + subscription.GetName())
		defer receiver.logger.Debug("removeSubscription end with subscription " + subscription.GetName())
	}

	topic := subscription.GetName()
	id, result, internalErr := receiver.unsubscribe(topic)
	if internalErr != nil {
		return nil, core.ToNativeError(internalErr)
	}
	receiver.addCorrelation(id)
	// Acquire the subscriptions lock only after the subscription has been removed to modify the list
	receiver.subscriptionsLock.Lock()
	defer receiver.subscriptionsLock.Unlock()
	spliceIndex := -1
	for i, subscribedTopic := range receiver.subscriptions {
		if subscribedTopic == topic {
			spliceIndex = i
		}
	}
	if spliceIndex >= 0 {
		receiver.subscriptions = append(receiver.subscriptions[:spliceIndex], receiver.subscriptions[spliceIndex+1:]...)
	}
	return result, nil
}

func (receiver *persistentMessageReceiverImpl) isSubscribed(topic string) bool {
	receiver.subscriptionsLock.Lock()
	defer receiver.subscriptionsLock.Unlock()
	for _, subscribedTopic := range receiver.subscriptions {
		if subscribedTopic == topic {
			return true
		}
	}
	return false
}

func (receiver *persistentMessageReceiverImpl) unsubscribe(topic string) (core.SubscriptionCorrelationID, <-chan core.SubscriptionEvent, core.ErrorInfo) {
	receiver.subscriptionsSynchronizationLock.Lock()
	defer receiver.subscriptionsSynchronizationLock.Unlock()
	// if we are not subscribed, we might want to use endpoint unsubscribe if we are a durable endpoint
	isSubscribed := receiver.isSubscribed(topic)
	if !isSubscribed {
		name, durable, errorInfo := receiver.internalFlow.Destination()
		if errorInfo != nil {
			return 0, nil, errorInfo
		}
		if durable {
			return receiver.internalReceiver.EndpointUnsubscribe(name, topic)
		}
		// if we are not durable, fall through to internalFlow.Unsubscribe
	}
	return receiver.internalFlow.Unsubscribe(topic)
}

func (receiver *persistentMessageReceiverImpl) addCorrelation(id core.SubscriptionCorrelationID) {
	receiver.outstandingSubscriptionEventsLock.Lock()
	defer receiver.outstandingSubscriptionEventsLock.Unlock()
	receiver.outstandingSubscriptionEvents[id] = struct{}{}
}

func (receiver *persistentMessageReceiverImpl) clearCorrelation(event core.SubscriptionEvent) {
	receiver.outstandingSubscriptionEventsLock.Lock()
	defer receiver.outstandingSubscriptionEventsLock.Unlock()
	delete(receiver.outstandingSubscriptionEvents, event.GetID())
}

// AddSubscriptionAsync will subscribe to another message source on a PubSub+ Broker to receive messages from.
// Will block until subscription is added.
// Returns a solace/errors.*IllegalStateError if the service is not running.
// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
// Returns nil if successful.
func (receiver *persistentMessageReceiverImpl) AddSubscriptionAsync(subscription resource.Subscription, listener solace.SubscriptionChangeListener) error {
	currentState := receiver.getState()
	if currentState != messageReceiverStateStarted {
		return solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToModifySubscriptionBadState, messageReceiverStateNames[currentState]), nil)
	}
	if err := checkPersistentMessageReceiverSubscriptionType(subscription); err != nil {
		return err
	}
	go func() {
		result, err := receiver.addSubscription(subscription)
		if listener != nil {
			if err != nil {
				listener(subscription, solace.SubscriptionAdded, err)
			} else {
				if receiver.logger.IsDebugEnabled() {
					receiver.logger.Debug("AddSubscriptionAsync awaiting confirm on subscription '" + subscription.GetName() + "'")
				}
				event := <-result
				if receiver.logger.IsDebugEnabled() {
					if event.GetError() != nil {
						receiver.logger.Debug("AddSubscriptionAsync received error on subscription '" + subscription.GetName() + "': " + event.GetError().Error())
					} else {
						receiver.logger.Debug("AddSubscriptionAsync received confirm on subscription '" + subscription.GetName() + "'")
					}
				}
				receiver.clearCorrelation(event)
				listener(subscription, solace.SubscriptionAdded, event.GetError())
			}
		}
	}()
	return nil
}

// RemoveSubscriptionAsymc will unsubscribe from a previously subscribed message source on a broker
// such that no more messages will be received from it. Will block until subscription is removed.
// Returns an solace/errors.*IllegalStateError if the service is not running.
// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
// Returns nil if successful.
func (receiver *persistentMessageReceiverImpl) RemoveSubscriptionAsync(subscription resource.Subscription, listener solace.SubscriptionChangeListener) error {
	currentState := receiver.getState()
	if currentState != messageReceiverStateStarted {
		return solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToModifySubscriptionBadState, messageReceiverStateNames[currentState]), nil)
	}
	if err := checkPersistentMessageReceiverSubscriptionType(subscription); err != nil {
		return err
	}
	go func() {
		result, err := receiver.removeSubscription(subscription)
		if listener != nil {
			if err != nil {
				listener(subscription, solace.SubscriptionRemoved, err)
			} else {
				if receiver.logger.IsDebugEnabled() {
					receiver.logger.Debug("RemoveSubscriptionAsync awaiting confirm on subscription '" + subscription.GetName() + "'")
				}
				event := <-result
				if receiver.logger.IsDebugEnabled() {
					if event.GetError() != nil {
						receiver.logger.Debug("RemoveSubscriptionAsync received error on subscription '" + subscription.GetName() + "': " + event.GetError().Error())
					} else {
						receiver.logger.Debug("RemoveSubscriptionAsync received confirm on subscription '" + subscription.GetName() + "'")
					}
				}
				receiver.clearCorrelation(event)
				listener(subscription, solace.SubscriptionRemoved, event.GetError())
			}
		}
	}()
	return nil
}

func (receiver *persistentMessageReceiverImpl) ReceiverInfo() (solace.PersistentReceiverInfo, error) {
	if receiver.getState() != messageReceiverStateStarted {
		return nil, solace.NewError(&solace.IllegalStateError{}, "cannot access receiver info when not in started state", nil)
	}
	name, durable, errorInfo := receiver.internalFlow.Destination()
	if errorInfo != nil {
		return nil, core.ToNativeError(errorInfo)
	}
	return &persistentReceiverInfoImpl{
		resourceInfo: &resourceInfoImpl{
			name:      name,
			isDurable: durable,
		},
	}, nil
}

// Ack will acknowledge a message
func (receiver *persistentMessageReceiverImpl) Ack(msg apimessage.InboundMessage) error {
	state := receiver.getState()
	if state != messageReceiverStateStarted && state != messageReceiverStateTerminating {
		var message string
		if state == messageReceiverStateTerminated {
			message = constants.UnableToAcknowledgeAlreadyTerminated
		} else {
			message = constants.UnableToAcknowledgeNotStarted
		}
		return solace.NewError(&solace.IllegalStateError{}, message, nil)
	}
	msgImpl, ok := msg.(*message.InboundMessageImpl)
	if !ok {
		return solace.NewError(&solace.IllegalArgumentError{}, fmt.Sprintf(constants.InvalidInboundMessageType, msg), nil)
	}
	msgID, present := message.GetMessageID(msgImpl)
	if !present {
		return solace.NewError(&solace.IllegalArgumentError{}, constants.UnableToRetrieveMessageID, nil)
	}
	errInfo := receiver.internalFlow.Ack(msgID)
	if errInfo != nil {
		return core.ToNativeError(errInfo)
	}
	return nil
}

// Pause will pause the receiver's message delivery to asynchronous message handlers.
// Pausing an already paused receiver will have no effect.
// Returns an IllegalStateErorr if the receiver is not started or already terminated.
func (receiver *persistentMessageReceiverImpl) Pause() error {
	state := receiver.getState()
	if state != messageReceiverStateStarted && state != messageReceiverStateTerminating {
		return solace.NewError(&solace.IllegalStateError{}, constants.PersistentReceiverCannotPauseBadState, nil)
	}
	receiver.logger.Info("Pausing message receiption")

	// drain any queued unpause events
	select {
	case <-receiver.doUnpause:
	default:
	}

	// tell the loop that on the next iteration message reception should be paused
	atomic.StoreInt32(&receiver.doPause, 1)

	// queue a pause interrupt event
	select {
	case receiver.pauseInterrupt <- nil:
	default:
	}
	return nil
}

// Resume will unpause the receiver's message delivery to asynchronous message handlers.
// Resume a receiver that is not paused will have no effect.
// Returns an IllegalStateErorr if the receiver is not started or already terminated.
func (receiver *persistentMessageReceiverImpl) Resume() error {
	state := receiver.getState()
	if state != messageReceiverStateStarted && state != messageReceiverStateTerminating {
		return solace.NewError(&solace.IllegalStateError{}, constants.PersistentReceiverCannotUnpauseBadState, nil)
	}
	receiver.logger.Info("Resuming message receiption")

	// drain any queue interrupt events
	select {
	case <-receiver.pauseInterrupt:
	default:
	}

	// tell the receiver loop to not stop and check
	atomic.StoreInt32(&receiver.doPause, 0)

	// notify the unpause channel
	select {
	case receiver.doUnpause <- nil:
	default: // we can only queue a single unpause event, so additional events have no meaning
	}
	return nil
}

// ReceiveAsync will register a callback to be called when new messages
// are received. Returns an error one occurred while registering the callback.
// If a callback is already registered, it will be replaced by the given
// callback.
func (receiver *persistentMessageReceiverImpl) ReceiveAsync(callback solace.MessageHandler) error {
	if receiver.IsTerminating() || receiver.IsTerminated() {
		return solace.NewError(&solace.IllegalStateError{}, constants.UnableToRegisterCallbackReceiverTerminating, nil)
	}
	if callback == nil {
		return solace.NewError(&solace.IllegalArgumentError{}, "callback may not be nil", nil)
	}
	// Check if we are the first to swap out, if we are notify the loop
	if atomic.CompareAndSwapPointer(&receiver.rxCallback, nil, unsafe.Pointer(&callback)) {
		select {
		case receiver.rxCallbackSet <- true:
			// success
		default:
			// we do not want to block if we cannot queue a message, that means one is already set
		}
	} else {
		atomic.StorePointer(&receiver.rxCallback, unsafe.Pointer(&callback))
	}
	return nil
}

func (receiver *persistentMessageReceiverImpl) ReceiveMessage(timeout time.Duration) (apimessage.InboundMessage, error) {
	state := receiver.getState()
	if state == messageReceiverStateNotStarted || state == messageReceiverStateStarting {
		return nil, solace.NewError(&solace.IllegalStateError{}, constants.ReceiverCannotReceiveNotStarted, nil)
	}

	defer receiver.checkEmptyBufferAndNotify()
	var msg *message.InboundMessageImpl
	var msgID message.MessageID
	var msgP ccsmp.SolClientMessagePt
	var ok bool
	if timeout >= 0 {
		timer := time.NewTimer(timeout)
		select {
		case msgP, ok = <-receiver.buffer:
			timer.Stop()
		case <-timer.C:
			return nil, solace.NewError(&solace.TimeoutError{}, constants.ReceiverTimedOutWaitingForMessage, nil)
		case <-receiver.bufferEmptyOnTerminate:
			timer.Stop()
			goto terminated
		}
	} else {
		select {
		case msgP, ok = <-receiver.buffer:
			// success
		case <-receiver.bufferEmptyOnTerminate:
			goto terminated
		}
	}
	if !ok {
		goto terminated
	}
	// reenable underlying flow if we are below lowwater AND we are not terminating/terminated
	if len(receiver.buffer) <= receiver.lowwater && receiver.getState() == messageReceiverStateStarted {
		receiver.startFlow()
	}
	// Prepare message for delivery
	msg = message.NewInboundMessage(msgP, false)
	// Ack the message just prior to return
	if receiver.doAutoAck {
		msgID, ok = message.GetMessageID(msg)
		if ok {
			if errInfo := receiver.internalFlow.Ack(msgID); errInfo != nil {
				receiver.logger.Debug(fmt.Sprintf("Failed to acknowledge message with id %d: %s", msgID, errInfo.GetMessageAsString()))
				msg.Dispose()
				return nil, core.ToNativeError(errInfo)
			}
		} else {
			receiver.logger.Error(fmt.Sprintf("Could not retrieve message ID from message %s", msg))
		}
	}
	return msg, nil
terminated:
	return nil, solace.NewError(&solace.IllegalStateError{}, constants.ReceiverCannotReceiveAlreadyTerminated, nil)
}

// Checks if the buffer is empty and closed, and if it is, notify of termination if needed
func (receiver *persistentMessageReceiverImpl) checkEmptyBufferAndNotify() bool {
	if atomic.LoadInt32(&receiver.bufferClosed) == 1 && len(receiver.buffer) == 0 &&
		atomic.CompareAndSwapInt32(&receiver.bufferEmptyOnTerminateFlag, 0, 1) {
		close(receiver.bufferEmptyOnTerminate)
		return true
	}
	return false
}

// startFlow will start the message flow of the receiver's underlying flow
// it will do this idempotently and it depends on internalFlowStopped being true (1)
// if successful, it will set internalFlowStopped to 0 (false), preventing additional calls
// to startFlow from calling CCSMP's flow_start function
func (receiver *persistentMessageReceiverImpl) startFlow() bool {
	// we want to flip to not stopped BEFORE we resume, thus in the case of a stop and start racing, we will
	// always either be started, or the stopped flag will be set.
	if atomic.CompareAndSwapInt32(&receiver.internalFlowStopped, 1, 0) {
		errInfo := receiver.internalFlow.Start()
		if errInfo != nil {
			receiver.logger.Info("Encountered error when starting underlying message flow: " + errInfo.String())
			// this allows us to retry
			atomic.StoreInt32(&receiver.internalFlowStopped, 1)
			return false
		}
		receiver.logger.Info("Starting underlying message flow")
		return true
	}
	return false
}

// stopFlow will stop the message flow of the receiver's underlyign flow
// this function will not act idempotently, but it does rely on internalFlowStopped being false (0)
// it will stop the flow and set internalFlowStopped to true (1)
func (receiver *persistentMessageReceiverImpl) stopFlow() bool {
	// check first if we are stopped, we don't want to spam stop when we are already stopped
	if atomic.LoadInt32(&receiver.internalFlowStopped) == 0 {
		errInfo := receiver.internalFlow.Stop()
		if errInfo != nil {
			receiver.logger.Info("Encountered error when stopping underlying message flow: " + errInfo.String())
			return false
		}
		atomic.StoreInt32(&receiver.internalFlowStopped, 1)
		receiver.logger.Info("Stopping underlying message flow")
		return true
	}
	return false
}

func (receiver *persistentMessageReceiverImpl) messageCallback(msg core.Receivable) (ret bool) {
	currentState := receiver.getState()
	if currentState == messageReceiverStateTerminating || currentState == messageReceiverStateTerminated {
		// we should not be handling this message
		receiver.logger.Debug("received message after receiver was terminated, dropping message")
		receiver.internalReceiver.IncrementMetric(core.MetricReceivedMessagesTerminationDiscarded, uint64(1))
		return false
	}
	defer func() {
		if r := recover(); r != nil {
			// we may have a race where the receiver buffer is closed before this function is called if unsubscribes are slow
			if err, ok := r.(error); ok && err.Error() == "send on closed channel" {
				receiver.logger.Debug("Caught a channel closed panic when trying to write to the message buffer, receiver must be terminated.")
				receiver.internalReceiver.IncrementMetric(core.MetricReceivedMessagesTerminationDiscarded, uint64(1))
			} else {
				// this shouldn't ever happen, but panics are unpredictable. We want this message to make it into the logs
				receiver.logger.Error(fmt.Sprintf("Caught panic in message callback! %s\n%s", err, string(debug.Stack())))
			}
			ret = false
		}
	}()
	select {
	case receiver.buffer <- msg:
		// success

		// check if we are above the highwater mark and make sure we are not stopped
		if len(receiver.buffer) >= receiver.highwater && receiver.stopFlow() {
			// There is a potential issue here where the flow is stopped, but before the flow stopped flag is set
			// the entire queue drains, thus there is no unpause. This is resolved by resuming the flow after the
			// fact if we have dropped below the low water mark
			if len(receiver.buffer) < receiver.lowwater && receiver.getState() == messageReceiverStateStarted {
				receiver.startFlow()
			}
		}
	default:
		// backpressure
		receiver.logger.Error("Unable to push message to buffer")
	}
	return true
}

func (receiver *persistentMessageReceiverImpl) run() {
	defer close(receiver.terminationComplete)
	// Block until an rx callback is set
	cont := <-receiver.rxCallbackSet
	// We will send false on the rxCallbackSet channel when we are terminating, indicating that we should shut down
	// and not continue to the loop below
	if !cont {
		return
	}
	for {
		// First check if we should pause
		if atomic.LoadInt32(&receiver.doPause) == 1 {
			// There is a potential condition where the receiver is paused just after the last message
			// is processed. To avoid an ungraceful termination, we must make sure we are not terminating
			// with a buffer length of 0 (ie. empty)
			if receiver.checkEmptyBufferAndNotify() {
				return
			}
			// wait to be unpaused, or wait for a termination notification
			select {
			case <-receiver.doUnpause:
				// successfully unpaused, do nothing, continue to check receiver buffer
			case <-receiver.terminationNotification:
				// we have been told to terminate ungracefully, exit now
				return
			case <-receiver.bufferEmptyOnTerminate:
				// we have received a signal that the buffer is empty on terminate
				return
			}
		}
		// Then we should check explicitly to see if we should terminate now.
		select {
		case <-receiver.terminationNotification:
			// we have been told to terminate, exit.
			return
		default:
			// we have not been told to terminate, proceed
		}
		// We have not been told to pause yet. Receive a message, terminate or be interrupted with pause
		select {
		case msgP, ok := <-receiver.buffer:
			if ok {
				callback := (*solace.MessageHandler)(atomic.LoadPointer(&receiver.rxCallback))
				// we never set a discard notification on messages received by a persistent receiver
				// since we never discard any messages.
				msg := message.NewInboundMessage(msgP, false)
				msgID, present := message.GetMessageID(msg)
				if !present && receiver.logger.IsDebugEnabled() {
					receiver.logger.Debug(fmt.Sprintf("Could not retrieve message ID from message %s", msg))
				}
				callbackPanic := false
				if callback != nil {
					func() {
						defer func() {
							if r := recover(); r != nil {
								callbackPanic = true
								receiver.logger.Warning("Message receiver callback paniced: " + fmt.Sprint(r))
							}
						}()
						(*callback)(msg)
					}()
				}
				if receiver.doAutoAck {
					if callbackPanic {
						receiver.logger.Info("ReceiveAsync callback paniced, will not auto acknowledge")
					} else {
						errInfo := receiver.internalFlow.Ack(msgID)
						if errInfo != nil {
							receiver.logger.Warning("Failed to acknowledge message: " + errInfo.GetMessageAsString() + ", sub code: " + fmt.Sprint(errInfo.SubCode))
						}
					}
				}
				// reenable underlying flow if we are below lowwater AND we are not terminating/terminated
				if len(receiver.buffer) <= receiver.lowwater && receiver.getState() == messageReceiverStateStarted {
					receiver.startFlow()
				}
			} else {
				// Since we cannot receive any more messages, the receiver buffer is empty and we should exit
				if atomic.CompareAndSwapInt32(&receiver.bufferEmptyOnTerminateFlag, 0, 1) {
					close(receiver.bufferEmptyOnTerminate)
				}
				return
			}
		case <-receiver.pauseInterrupt:
			// someone has told us to pause, we have been interrupted from waiting for a message to be received, loop
		case <-receiver.terminationNotification:
			return
		}
	}
}

func (receiver *persistentMessageReceiverImpl) String() string {
	return fmt.Sprintf("solace.PersistentMessageReceiver at %p", receiver)
}

type persistentMessageReceiverBuilderImpl struct {
	internalReceiver core.Receiver
	properties       map[config.ReceiverProperty]interface{}
	subscriptions    []resource.Subscription
}

// NewPersistentMessageReceiverBuilderImpl function
func NewPersistentMessageReceiverBuilderImpl(internalReceiver core.Receiver) solace.PersistentMessageReceiverBuilder {
	return &persistentMessageReceiverBuilderImpl{
		internalReceiver: internalReceiver,
		properties:       constants.DefaultPersistentReceiverProperties.GetConfiguration(),
	}
}

// Build will build a new PersistentMessageReceiver with the given properties.
// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
func (builder *persistentMessageReceiverBuilderImpl) Build(queue *resource.Queue) (messageReceiver solace.PersistentMessageReceiver, err error) {
	// Validate that subscriptions are of correct type
	for _, subscription := range builder.subscriptions {
		if err = checkPersistentMessageReceiverSubscriptionType(subscription); err != nil {
			return nil, err
		}
	}

	// some constants
	const bufferHighwaterDefault = 50
	const bufferLowwaterDefault = 40

	// For each property, make sure it is present, and then make sure that the appropriate ccsmp properties are set
	var doCreateMissingResource bool = false
	if strategy, ok := builder.properties[config.ReceiverPropertyPersistentMissingResourceCreationStrategy]; ok {
		if strategyAsMissingResourceCreationStrategy, ok := strategy.(config.MissingResourcesCreationStrategy); ok {
			strategy = string(strategyAsMissingResourceCreationStrategy)
		}
		prop, present, err := validation.StringPropertyValidation(string(config.ReceiverPropertyPersistentMissingResourceCreationStrategy), strategy,
			string(config.PersistentReceiverCreateOnStartMissingResources), string(config.PersistentReceiverDoNotCreateMissingResources))
		if present {
			if err != nil {
				return nil, err
			}
			doCreateMissingResource = prop == string(config.PersistentReceiverCreateOnStartMissingResources)
		}
	}

	var doAutoAck bool = false
	if ackStrategyInterface, ok := builder.properties[config.ReceiverPropertyPersistentMessageAckStrategy]; ok {
		ackStrategy, present, err := validation.StringPropertyValidation(string(config.ReceiverPropertyPersistentMessageAckStrategy), ackStrategyInterface,
			config.PersistentReceiverAutoAck, config.PersistentReceiverClientAck)
		if present {
			if err != nil {
				return nil, err
			}
			doAutoAck = ackStrategy == config.PersistentReceiverAutoAck
		}
	}

	var receiverStateChangeListener solace.ReceiverStateChangeListener = nil
	if stateChangeListenerInterface, ok := builder.properties[config.ReceiverPropertyPersistentStateChangeListener]; ok {
		receiverStateChangeListener, ok = stateChangeListenerInterface.(solace.ReceiverStateChangeListener)
		if !ok {
			return nil, solace.NewError(&solace.IllegalArgumentError{},
				fmt.Sprintf("invalid type for receiver state change listener, expected solace.ReceiverStateChangeListener, got %T", stateChangeListenerInterface), nil)
		}
	}

	// Now deal with the CCSMP flow properties
	var properties []string = []string{
		// set the entity type to queue
		ccsmp.SolClientFlowPropBindEntityID, ccsmp.SolClientFlowPropBindEntityQueue,
		// set the ackmode to client, we handle auto ack in the receiver, NOT through ccsmp
		ccsmp.SolClientFlowPropAckmode, ccsmp.SolClientFlowPropAckmodeClient,
		// set the active flow indicator to enabled
		ccsmp.SolClientFlowPropActiveFlowInd, ccsmp.SolClientPropEnableVal,
		// start the flow in the 'stopped' state, SOL-63525
		ccsmp.SolClientFlowPropStartState, ccsmp.SolClientPropDisableVal,
	}

	// Add queue name
	if queue == nil {
		return nil, solace.NewError(&solace.IllegalArgumentError{}, constants.PersistentReceiverMissingQueue, nil)
	}
	if queue.GetName() != "" || queue.IsDurable() {
		properties = append(properties, ccsmp.SolClientFlowPropBindName, queue.GetName())
	}

	// Set queue durability
	var isDurable string
	if queue.IsDurable() {
		isDurable = ccsmp.SolClientPropEnableVal
	} else {
		isDurable = ccsmp.SolClientPropDisableVal
	}
	properties = append(properties, ccsmp.SolClientFlowPropBindEntityDurable, isDurable)

	// If we have a selector configured, add that to the flow props
	if selector, ok := builder.properties[config.ReceiverPropertyPersistentMessageSelectorQuery]; ok {
		prop, present, err := validation.StringPropertyValidation(string(config.ReceiverPropertyPersistentMessageSelectorQuery), selector)
		if present {
			if err != nil {
				return nil, err
			}
			properties = append(properties, ccsmp.SolClientFlowPropSelector, prop)
		}
	}

	// If we have a replay strategy configured, add that to the flow props
	if replayStrategyInterface, ok := builder.properties[config.ReceiverPropertyPersistentMessageReplayStrategy]; ok {
		replayStrategy, present, err := validation.StringPropertyValidation(
			string(config.ReceiverPropertyPersistentMessageReplayStrategy),
			replayStrategyInterface,
			config.PersistentReplayAll,
			config.PersistentReplayTimeBased,
			config.PersistentReplayIDBased,
		)
		// First check the replay strategy, then configure the start time based off the configured strategy
		if present {
			if err != nil {
				return nil, err
			}
			var replayStart string
			switch replayStrategy {
			case config.PersistentReplayAll:
				replayStart = ccsmp.SolClientFlowPropReplayStartLocationBeginning
			case config.PersistentReplayTimeBased:
				if startTimeInterface, ok := builder.properties[config.ReceiverPropertyPersistentMessageReplayStrategyTimeBasedStartTime]; ok && startTimeInterface != nil {
					const datePrefix = "DATE:"
					if startTimeAsTime, ok := startTimeInterface.(time.Time); ok {
						replayStart = fmt.Sprintf("%s%d", datePrefix, startTimeAsTime.Unix())
					} else {
						// Any value that is not time.Time is assumed to have a valid value requiring no manipulation
						// This may fail at startup of the resulting receiver when the flow is bound
						replayStart = fmt.Sprintf("%s%v", datePrefix, startTimeInterface)
					}
				} else {
					return nil, solace.NewError(&solace.IllegalArgumentError{}, constants.PersistentReceiverMustSpecifyTime, nil)
				}
			case config.PersistentReplayIDBased:
				if rmidInterface, ok := builder.properties[config.ReceiverPropertyPersistentMessageReplayStrategyIDBasedReplicationGroupMessageID]; ok && rmidInterface != nil {
					if rmidType, ok := rmidInterface.(rgmid.ReplicationGroupMessageID); ok {
						// we want to accept ReplicationGroupMessageID as well as strings, so we should convert the RGMID to a string
						// we don't necessarily want to support all stringer linked types so we will leave the validator as is
						rmidInterface = rmidType.String()
					}
					rmid, _, err := validation.StringPropertyValidation(string(config.ReceiverPropertyPersistentMessageReplayStrategyIDBasedReplicationGroupMessageID), rmidInterface)
					if err != nil {
						return nil, err
					}
					if len(rmid) == 0 {
						return nil, solace.NewError(&solace.IllegalArgumentError{}, "message ID must not be empty", nil)
					}
					replayStart = rmid
				} else {
					return nil, solace.NewError(&solace.IllegalArgumentError{}, constants.PersistentReceiverMustSpecifyRGMID, nil)
				}
			}
			properties = append(properties, ccsmp.SolClientFlowPropReplayStartLocation, replayStart)
		}
	}

	// Create the receiver with the given properties
	receiver := &persistentMessageReceiverImpl{}
	receiver.construct(
		&persistentMessageReceiverProps{
			flowProperties:          properties,
			internalReceiver:        builder.internalReceiver,
			startupSubscriptions:    builder.subscriptions,
			bufferHighwater:         bufferHighwaterDefault,
			bufferLowwater:          bufferLowwaterDefault,
			endpoint:                queue,
			doCreateMissingResource: doCreateMissingResource,
			doAutoAck:               doAutoAck,
			stateChangeListener:     receiverStateChangeListener,
		},
	)

	return receiver, nil
}

// WithSubscriptions will set a list of TopicSubscriptions to subscribe
// to when starting the receiver. Accepts TopicSubscription and
func (builder *persistentMessageReceiverBuilderImpl) WithSubscriptions(topics ...resource.Subscription) solace.PersistentMessageReceiverBuilder {
	builder.subscriptions = topics
	return builder
}

// FromConfigurationProvider will configure the persistent receiver with the given properties.
// Built in ReceiverPropertiesConfigurationProvider implementations include:
//   ReceiverPropertyMap, a map of ReceiverProperty keys to values
func (builder *persistentMessageReceiverBuilderImpl) FromConfigurationProvider(provider config.ReceiverPropertiesConfigurationProvider) solace.PersistentMessageReceiverBuilder {
	if provider == nil {
		return builder
	}
	for key, value := range provider.GetConfiguration() {
		builder.properties[key] = value
	}
	return builder
}

// WithActivationPassivationSupport sets the listener to receiver broker notifications
// about state changes for the resulting receiver. This change can happen if there are
// multiple instances of the same receiver for high availability and activity is exchanged.
// This change is handled by the broker.
func (builder *persistentMessageReceiverBuilderImpl) WithActivationPassivationSupport(listener solace.ReceiverStateChangeListener) solace.PersistentMessageReceiverBuilder {
	builder.properties[config.ReceiverPropertyPersistentStateChangeListener] = listener
	return builder
}

// WithMessageAutoAcknowledgement sets whether or not the resulting PersistentMessageReceiver
// should auto acknowledge messages once received.
func (builder *persistentMessageReceiverBuilderImpl) WithMessageAutoAcknowledgement() solace.PersistentMessageReceiverBuilder {
	builder.properties[config.ReceiverPropertyPersistentMessageAckStrategy] = config.PersistentReceiverAutoAck
	return builder
}

func (builder *persistentMessageReceiverBuilderImpl) WithMessageClientAcknowledgement() solace.PersistentMessageReceiverBuilder {
	builder.properties[config.ReceiverPropertyPersistentMessageAckStrategy] = config.PersistentReceiverClientAck
	return builder
}

// WithMessageSelector will set the message selector to the given string.
// If an empty string is given, the filter will be cleared.
func (builder *persistentMessageReceiverBuilderImpl) WithMessageSelector(filterSelectorExpression string) solace.PersistentMessageReceiverBuilder {
	if filterSelectorExpression == "" {
		delete(builder.properties, config.ReceiverPropertyPersistentMessageSelectorQuery)
	} else {
		builder.properties[config.ReceiverPropertyPersistentMessageSelectorQuery] = filterSelectorExpression
	}
	return builder
}

// WithMissingResourcesCreationStrategy sets the missing resource creation strategy
// defining what actions the API may take when missing resources are detected.
func (builder *persistentMessageReceiverBuilderImpl) WithMissingResourcesCreationStrategy(strategy config.MissingResourcesCreationStrategy) solace.PersistentMessageReceiverBuilder {
	builder.properties[config.ReceiverPropertyPersistentMissingResourceCreationStrategy] = strategy
	return builder
}

// WithMessageReplay enables support for message replay using a specific replay
// strategy. Once started, the receiver will being replay using the given strategy.
// Valid strategies include config.ReplayStrategyAllMessages() and
// config.ReplayStrategyTimeBased.
func (builder *persistentMessageReceiverBuilderImpl) WithMessageReplay(strategy config.ReplayStrategy) solace.PersistentMessageReceiverBuilder {
	switch strategy.GetStrategy() {
	case config.PersistentReplayAll:
		builder.properties[config.ReceiverPropertyPersistentMessageReplayStrategy] = config.PersistentReplayAll
	case config.PersistentReplayTimeBased:
		builder.properties[config.ReceiverPropertyPersistentMessageReplayStrategy] = config.PersistentReplayTimeBased
		builder.properties[config.ReceiverPropertyPersistentMessageReplayStrategyTimeBasedStartTime] = strategy.GetData()
	case config.PersistentReplayIDBased:
		builder.properties[config.ReceiverPropertyPersistentMessageReplayStrategy] = config.PersistentReplayIDBased
		builder.properties[config.ReceiverPropertyPersistentMessageReplayStrategyIDBasedReplicationGroupMessageID] = strategy.GetData()
	default:
		logging.Default.Warning(builder.String() + ": Unknown strategy type passed to WithMessageReplay, was it manually instantiated?")
	}
	return builder
}

func (builder *persistentMessageReceiverBuilderImpl) String() string {
	return fmt.Sprintf("solace.PersistentMessageReceiverBuilder at %p", builder)
}

// Validate the subscription type is one supported by
func checkPersistentMessageReceiverSubscriptionType(subscription resource.Subscription) error {
	switch subscription.(type) {
	case *resource.TopicSubscription:
		return nil
	}
	return solace.NewError(&solace.IllegalArgumentError{}, fmt.Sprintf(constants.PersistentReceiverUnsupportedSubscriptionType, subscription), nil)
}

type persistentReceiverInfoImpl struct {
	resourceInfo *resourceInfoImpl
}

func (info *persistentReceiverInfoImpl) GetResourceInfo() solace.ResourceInfo {
	return info.resourceInfo
}

type resourceInfoImpl struct {
	name      string
	isDurable bool
}

func (info *resourceInfoImpl) GetName() string {
	return info.name
}

func (info *resourceInfoImpl) IsDurable() bool {
	return info.isDurable
}
