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

package receiver

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/internal/impl/logging"
	"solace.dev/go/messaging/internal/impl/message"
	"solace.dev/go/messaging/internal/impl/validation"
	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/config"
	apimessage "solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/resource"
)

type receiverBackpressureStrategy byte

const (
	strategyDropOldest receiverBackpressureStrategy = iota
	strategyDropLatest receiverBackpressureStrategy = iota
)

type discardValue = int32

const (
	discardFalse discardValue = iota
	discardTrue  discardValue = iota
)

type directMessageReceiverImpl struct {
	basicMessageReceiver

	logger logging.LogLevelLogger

	subscriptionsLock           sync.Mutex
	subscriptionTerminationLock sync.RWMutex
	subscriptions               []string
	// we want to synchronize calls to subscribe/unsubscribe to avoid crashing due to thread limitations
	subscriptionsSynchronizationLock sync.Mutex

	shareName            *resource.ShareName
	buffer               chan *directInboundMessage
	bufferClosed         int32
	backpressureStrategy receiverBackpressureStrategy

	bufferEmptyOnTerminateFlag int32
	bufferEmptyOnTerminate     chan struct{}

	terminationNotification chan struct{}
	terminationComplete     chan struct{}

	rxCallbackSet chan bool
	rxCallback    unsafe.Pointer
	isDiscard     int32

	dispatch uintptr

	terminationHandlerID uint
}

type directInboundMessage struct {
	pointer ccsmp.SolClientMessagePt
	discard bool
}

type directMessageReceiverProps struct {
	internalReceiver       core.Receiver
	startupSubscriptions   []resource.Subscription
	backpressureStrategy   receiverBackpressureStrategy
	backpressureBufferSize int
	shareName              *resource.ShareName
}

func (receiver *directMessageReceiverImpl) construct(props *directMessageReceiverProps) {
	receiver.basicMessageReceiver.construct(props.internalReceiver)
	receiver.shareName = props.shareName
	receiver.subscriptions = make([]string, len(props.startupSubscriptions))
	for i, subscription := range props.startupSubscriptions {
		receiver.subscriptions[i] = receiver.buildSubscription(subscription)
	}
	receiver.buffer = make(chan *directInboundMessage, props.backpressureBufferSize)
	receiver.bufferClosed = 0
	receiver.backpressureStrategy = props.backpressureStrategy
	receiver.isDiscard = 0

	receiver.terminationNotification = make(chan struct{})
	receiver.terminationComplete = make(chan struct{})

	receiver.bufferEmptyOnTerminateFlag = 0
	receiver.bufferEmptyOnTerminate = make(chan struct{})

	receiver.logger = logging.For(receiver)

	atomic.StorePointer(&receiver.rxCallback, nil)
	receiver.rxCallbackSet = make(chan bool, 1)
}

func (receiver *directMessageReceiverImpl) onDownEvent(eventInfo core.SessionEventInfo) {
	// terminate immediately and skip subscription deregistration as the connection is dead at this point
	go receiver.unsolicitedTermination(eventInfo)
}

// Start will start the service synchronously.
// Before this function is called, the service is considered
// off-duty. To operate normally, this function must be called on
// a receiver or publisher instance. This function is idempotent.
// Returns an error if one occurred or nil if successful.
func (receiver *directMessageReceiverImpl) Start() (err error) {
	// this will block until we are started if we are not first
	if proceed, err := receiver.starting(); !proceed {
		return err
	}
	receiver.logger.Debug("Start receiver start")
	subscriptionsAdded := make([]string, 0)
	defer func() {
		if err == nil {
			receiver.started(err)
			receiver.logger.Debug("Start receiver complete")
		} else {
			receiver.logger.Debug("Start receiver complete with error: " + err.Error())
			unsubResults := make([]<-chan core.SubscriptionEvent, len(subscriptionsAdded))
			for i, subscription := range subscriptionsAdded {
				_, unsubResult, unsubErr := receiver.internalReceiver.Unsubscribe(subscription, receiver.dispatch)
				if unsubErr != nil {
					receiver.logger.Debug("Failed to unsubscribe from subscribed topic '" + subscription +
						"' when cleaning up on failed start: " + unsubErr.GetMessageAsString())
				} else {
					unsubResults[i] = unsubResult
				}
			}
			for i, unsubResult := range unsubResults {
				if unsubResult != nil {
					event := <-unsubResult
					if event.GetError() != nil {
						receiver.logger.Debug("Failed to unsubscribe from subscribed topic '" + subscriptionsAdded[i] +
							"' when cleaning up on failed start: " + event.GetError().Error())
					}
				}
			}
			receiver.internalReceiver.Events().RemoveEventHandler(receiver.terminationHandlerID)
			receiver.internalReceiver.UnregisterRXCallback(receiver.dispatch)
			receiver.terminated(nil)
			receiver.startFuture.Complete(err)
		}
	}()
	receiver.terminationHandlerID = receiver.internalReceiver.Events().AddEventHandler(core.SolClientEventDown, receiver.onDownEvent)
	receiver.dispatch = receiver.internalReceiver.RegisterRXCallback(receiver.messageCallback)
	subscriptionResults := make([]<-chan core.SubscriptionEvent, len(receiver.subscriptions))
	for i, subscription := range receiver.subscriptions {
		// we can safely ignore the IDs since we are in direct messaging so the results will be tied to the session, not this receiver
		_, result, errInfo := receiver.internalReceiver.Subscribe(subscription, receiver.dispatch)
		if errInfo != nil {
			return core.ToNativeError(errInfo, constants.FailedToAddSubscription)
		}
		subscriptionsAdded = append(subscriptionsAdded, subscription)
		subscriptionResults[i] = result
	}
	// wait on all the subscriptions we successfully added
	for _, result := range subscriptionResults {
		if result != nil {
			event := <-result
			if event.GetError() != nil {
				return event.GetError()
			}
		}
	}
	// we will start the receiver loop even if we do not yet have a callback
	go receiver.run()
	return nil
}

// StartAsync will start the service asynchronously.
// Before this function is called, the service is considered
// off-duty. To operate normally, this function must be called on
// a receiver or publisher instance. This function is idempotent.
// Returns a channel that will receive an error if one occurred or
// nil if successful. Subsequent calls will return additional
// channels that can await an error, or nil if already started.
func (receiver *directMessageReceiverImpl) StartAsync() <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- receiver.Start()
		close(result)
	}()
	return result
}

// StartAsyncCallback will start the DirectMessageReceiver asynchronously.
// Calls the callback when started with an error if one occurred or nil
// if successful.
func (receiver *directMessageReceiverImpl) StartAsyncCallback(callback func(solace.DirectMessageReceiver, error)) {
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
func (receiver *directMessageReceiverImpl) Terminate(gracePeriod time.Duration) (err error) {
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
	// On an ungraceful termination, we want to skip subscription removal
	receiver.cleanupSubscriptions()
	// Remove the dispatch callback from the internal receiver
	receiver.internalReceiver.UnregisterRXCallback(receiver.dispatch)
	// Remove the termination event handler
	receiver.internalReceiver.Events().RemoveEventHandler(receiver.terminationHandlerID)

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
		// successfully drained buffer
		timer.Stop()
		// join receiver thread. we want to make sure that if we enter with 0 messages in the buffer but one message
		// is still being processed by the async callback, we will not terminate until that message callback is complete
		<-receiver.terminationComplete
	}
	return nil
}

func (receiver *directMessageReceiverImpl) unsolicitedTermination(eventInfo core.SessionEventInfo) {
	if proceed, _ := receiver.basicMessageReceiver.terminate(); !proceed {
		// we are already terminated, nothing to do
		return
	}
	receiver.logger.Debug("Received unsolicited termination with event info " + eventInfo.GetInfoString())
	defer receiver.logger.Debug("Unsolicited termination complete")
	timestamp := time.Now()
	// Remove the dispatch callback from the internal receiver in case we still get any messages
	receiver.internalReceiver.UnregisterRXCallback(receiver.dispatch)
	// Remove the event handler
	receiver.internalReceiver.Events().RemoveEventHandler(receiver.terminationHandlerID)

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
	atomic.StoreInt32(&receiver.bufferClosed, 1)

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

func (receiver *directMessageReceiverImpl) cleanupSubscriptions() {
	receiver.subscriptionTerminationLock.Lock()
	defer receiver.subscriptionTerminationLock.Unlock()
	receiver.logger.Debug("Cleaning up subscriptions")
	results := make([]<-chan core.SubscriptionEvent, len(receiver.subscriptions))
	for i, subscription := range receiver.subscriptions {
		_, result, err := receiver.internalReceiver.Unsubscribe(subscription, receiver.dispatch)
		if err != nil {
			receiver.logger.Error("encountered error unsubscribing from topic in direct receiver terminate: " + err.GetMessageAsString())
			// we don't want to return this error, this may be expected behaviour in certain scenarios and we should continue to shutdown
			// for example, if we get a down event after we are already terminating, we want to continue with shutdown
		} else {
			results[i] = result
		}
	}
	// we trust that ccsmp will give us a result, if the messaging service is terminated we will get an event
	for i, result := range results {
		if result != nil {
			event := <-result
			if event.GetError() != nil {
				receiver.logger.Debug("Failed to unsubscribe from subscribed topic '" + receiver.subscriptions[i] +
					"' when cleaning up subscriptions: " + event.GetError().Error())
			}
		}
	}
}

// drainQueue will drain out all remaining messages in the receiver buffer and will return the
// number of messages drained. There is a potential race between this function and the synchronous
// ReceiveMessage function whereby message order will be lost. This is expected behaviour as
// we are terminating ungracefully when drainQueue is called, thus there is no more guarantee of
// functionality. There are two potential workarounds if the race causes issues: 1. terminate gracefully
// and 2. use receive async.
func (receiver *directMessageReceiverImpl) drainQueue() uint64 {
	undeliveredCount := uint64(0)
	for msg := range receiver.buffer {
		undeliveredCount++
		ccsmp.SolClientMessageFree(&msg.pointer)
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
func (receiver *directMessageReceiverImpl) TerminateAsync(gracePeriod time.Duration) <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- receiver.Terminate(gracePeriod)
		close(result)
	}()
	return result
}

// TerminateAsyncCallback will terminate the DirectMessageReceiver asynchronously.
// Calls the callback when terminated with nil if successful or an error if
// one occurred. If gracePeriod is less than 0, the function will wait indefinitely.
func (receiver *directMessageReceiverImpl) TerminateAsyncCallback(gracePeriod time.Duration, callback func(error)) {
	go func() {
		callback(receiver.Terminate(gracePeriod))
	}()
}

// AddSubscription will subscribe to another message source on a PubSub+ Broker to receive messages from.
// Will block until subscription is added.
// Returns a solace/errors.*IllegalStateError if the service is not running.
// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
// Returns nil if successful.
func (receiver *directMessageReceiverImpl) AddSubscription(subscription resource.Subscription) error {
	// Check the state first un the event that we are currently holding the termination lock in the terminate function
	// This will fail much faster and will avoid hanging
	currentState := receiver.getState()
	if currentState != messageReceiverStateStarted {
		return solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToModifySubscriptionBadState, messageReceiverStateNames[currentState]), nil)
	}
	if err := checkDirectMessageReceiverSubscriptionType(subscription); err != nil {
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
	return event.GetError()
}

// common addSubscription without first check for state shared by sync and async
func (receiver *directMessageReceiverImpl) addSubscription(subscription resource.Subscription) (<-chan core.SubscriptionEvent, error) {
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

	topic := receiver.buildSubscription(subscription)
	_, result, internalErr := receiver.subscribe(topic)
	if internalErr != nil {
		return nil, core.ToNativeError(internalErr)
	}

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

func (receiver *directMessageReceiverImpl) buildSubscription(subscription resource.Subscription) string {
	if receiver.shareName != nil {
		return "#share/" + receiver.shareName.GetName() + "/" + subscription.GetName()
	}
	return subscription.GetName()
}

func (receiver *directMessageReceiverImpl) subscribe(topic string) (core.SubscriptionCorrelationID, <-chan core.SubscriptionEvent, core.ErrorInfo) {
	receiver.subscriptionsSynchronizationLock.Lock()
	defer receiver.subscriptionsSynchronizationLock.Unlock()
	return receiver.internalReceiver.Subscribe(topic, receiver.dispatch)
}

// RemoveSubscription will unsubscribe from a previously subscribed message source on a broker
// such that no more messages will be received from it.
// Will block until subscription is removed.
// Returns an solace/errors.*IllegalStateError if the service is not running.
// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
// Returns nil if successful.
func (receiver *directMessageReceiverImpl) RemoveSubscription(subscription resource.Subscription) error {
	// Check the state first un the event that we are currently holding the termination lock in the terminate function
	// This will fail much faster and will avoid hanging
	currentState := receiver.getState()
	if currentState != messageReceiverStateStarted {
		return solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToModifySubscriptionBadState, messageReceiverStateNames[currentState]), nil)
	}
	if err := checkDirectMessageReceiverSubscriptionType(subscription); err != nil {
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
	return event.GetError()
}

// common code without first check for state shared between sync and async remove subscriptions
func (receiver *directMessageReceiverImpl) removeSubscription(subscription resource.Subscription) (<-chan core.SubscriptionEvent, error) {
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

	topic := receiver.buildSubscription(subscription)
	_, result, internalErr := receiver.unsubscribe(topic)
	if internalErr != nil {
		return nil, core.ToNativeError(internalErr)
	}
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

func (receiver *directMessageReceiverImpl) unsubscribe(topic string) (core.SubscriptionCorrelationID, <-chan core.SubscriptionEvent, core.ErrorInfo) {
	receiver.subscriptionsSynchronizationLock.Lock()
	defer receiver.subscriptionsSynchronizationLock.Unlock()
	return receiver.internalReceiver.Unsubscribe(topic, receiver.dispatch)
}

// AddSubscriptionAsync will subscribe to another message source on a PubSub+ Broker to receive messages from.
// Will block until subscription is added.
// Returns a solace/errors.*IllegalStateError if the service is not running.
// Returns a solace/errors.*IllegalArgumentError if unsupported Subscription type is passed.
// Returns nil if successful.
func (receiver *directMessageReceiverImpl) AddSubscriptionAsync(subscription resource.Subscription, listener solace.SubscriptionChangeListener) error {
	currentState := receiver.getState()
	if currentState != messageReceiverStateStarted {
		return solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToModifySubscriptionBadState, messageReceiverStateNames[currentState]), nil)
	}
	if err := checkDirectMessageReceiverSubscriptionType(subscription); err != nil {
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
func (receiver *directMessageReceiverImpl) RemoveSubscriptionAsync(subscription resource.Subscription, listener solace.SubscriptionChangeListener) error {
	currentState := receiver.getState()
	if currentState != messageReceiverStateStarted {
		return solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToModifySubscriptionBadState, messageReceiverStateNames[currentState]), nil)
	}
	if err := checkDirectMessageReceiverSubscriptionType(subscription); err != nil {
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
				listener(subscription, solace.SubscriptionRemoved, event.GetError())
			}
		}
	}()
	return nil
}

func (receiver *directMessageReceiverImpl) ReceiveMessage(timeout time.Duration) (apimessage.InboundMessage, error) {
	state := receiver.getState()
	if state == messageReceiverStateNotStarted || state == messageReceiverStateStarting {
		return nil, solace.NewError(&solace.IllegalStateError{}, constants.ReceiverCannotReceiveNotStarted, nil)
	}
	defer func() {
		// notify of termination
		if atomic.LoadInt32(&receiver.bufferClosed) == 1 && len(receiver.buffer) == 0 &&
			atomic.CompareAndSwapInt32(&receiver.bufferEmptyOnTerminateFlag, 0, 1) {
			close(receiver.bufferEmptyOnTerminate)
		}
	}()
	var msg *directInboundMessage
	var ok bool
	if timeout >= 0 {
		timer := time.NewTimer(timeout)
		select {
		case msg, ok = <-receiver.buffer:
			timer.Stop()
		case <-timer.C:
			return nil, solace.NewError(&solace.TimeoutError{}, constants.ReceiverTimedOutWaitingForMessage, nil)
		case <-receiver.bufferEmptyOnTerminate:
			timer.Stop()
			goto terminated
		}
	} else {
		select {
		case msg, ok = <-receiver.buffer:
			// success
		case <-receiver.bufferEmptyOnTerminate:
			goto terminated
		}
	}
	if !ok {
		goto terminated
	}
	// TODO there is a potential race condition here where a message is consumed
	// after a message has been discarded but before the notification is set.
	// This can only be fixed with mutex protection. This should be reevaluated
	// and the performance impact of mutex protecting should be assessed.
	if receiver.backpressureStrategy == strategyDropOldest {
		// when in drop oldest backpressure, we set the discard notification on the next consumed message
		msg.discard = atomic.CompareAndSwapInt32(&receiver.isDiscard, discardTrue, discardFalse)
	}
	if msg.discard {
		receiver.internalReceiver.IncrementMetric(core.MetricInternalDiscardNotifications, 1)
	}
	return message.NewInboundMessage(msg.pointer, msg.discard), nil
terminated:
	return nil, solace.NewError(&solace.IllegalStateError{}, constants.ReceiverCannotReceiveAlreadyTerminated, nil)
}

// ReceiveAsync will register a callback to be called when new messages
// are received. Returns an error one occurred while registering the callback.
// If a callback is already registered, it will be replaced by the given
// callback.
func (receiver *directMessageReceiverImpl) ReceiveAsync(callback solace.MessageHandler) (err error) {
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

func (receiver *directMessageReceiverImpl) messageCallback(msg core.Receivable) (ret bool) {
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
	setDiscard := false
	// When we are in backpressure drop latest, we set the discard notification on the next pushed message
	if receiver.backpressureStrategy == strategyDropLatest {
		setDiscard = atomic.CompareAndSwapInt32(&receiver.isDiscard, discardTrue, discardFalse)
	}
	// push a new message to the receiver buffer
	toPush := &directInboundMessage{msg, setDiscard}
	select {
	case receiver.buffer <- toPush:
		// success
	default:
		discard := true
		// backpressure
		switch receiver.backpressureStrategy {
		case strategyDropOldest:

			select {
			case <-receiver.buffer:
				// We successfully removed a message from the buffer to drop
			default:
				// there may be a race if the buffer size is very small and the queue has been drained since the push
				// in this case, we do not need to discard any messages and we can instead queue the message normally.
				// we also do not want to set the discard notification or increment the discard metric.
				discard = false
			}
			if discard {
				atomic.StoreInt32(&receiver.isDiscard, discardTrue)
			}
			// now there is guaranteed to be space as the rx callback is run on the context thread, so no additional messages
			// are queued in the time between the previous operation and this operation.
			receiver.buffer <- toPush
		case strategyDropLatest:
			// we are dropping the current message, noop
			atomic.StoreInt32(&receiver.isDiscard, discardTrue)
		}
		if discard {
			// increment stats
			receiver.internalReceiver.IncrementMetric(core.MetricReceivedMessagesBackpressureDiscarded, uint64(1))
			// keep the message (true) if we have buffered it (ie. backpressure strategy is drop oldest)
			// otherwise, we use a small optimization where we return false indicating to CCSMP that the message can be freed
			return receiver.backpressureStrategy == strategyDropOldest
		}
	}
	return true
}

func (receiver *directMessageReceiverImpl) run() {
	// When the function returns, notify of completion
	defer close(receiver.terminationComplete)
	// Block until an rx callback is set
	cont := <-receiver.rxCallbackSet
	// We will send false on the rxCallbackSet channel when we are terminating, indicating that we should shut down
	// and not continue to the loop below
	if !cont {
		return
	}
	for {
		// First thing we do in the loop is check if we are terminated.
		// We must do this first as a select statement will arbitrarily choose a path if both
		// are not blocked.
		select {
		case <-receiver.terminationNotification:
			// sometime between the last receive and now we have been told to terminate
			return
		default:
			// we have not been told to terminate now yet, proceed
		}
		// either receive from the buffer, or be interrupted by the termination notification
		select {
		case received, ok := <-receiver.buffer:
			if ok {
				callback := (*solace.MessageHandler)(atomic.LoadPointer(&receiver.rxCallback))
				// TODO there is a potential race condition here where a message is consumed
				// after a message has been discarded but before the notification is set.
				// This can only be fixed with mutex protection. This should be reevaluated
				// and the performance impact of mutex protecting should be assessed.
				if receiver.backpressureStrategy == strategyDropOldest {
					// when in drop oldest backpressure, we set the discard notification on the next consumed message
					received.discard = atomic.CompareAndSwapInt32(&receiver.isDiscard, discardTrue, discardFalse)
				}
				if received.discard {
					receiver.internalReceiver.IncrementMetric(core.MetricInternalDiscardNotifications, 1)
				}
				msg := message.NewInboundMessage(received.pointer, received.discard)
				if callback != nil {
					func() {
						defer func() {
							if r := recover(); r != nil {
								receiver.logger.Warning("Message receiver callback paniced: " + fmt.Sprint(r))
							}
						}()
						(*callback)(msg)
					}()
				}
			} else {
				// We must safely handle closing of receiver.bufferEmpty
				if atomic.CompareAndSwapInt32(&receiver.bufferEmptyOnTerminateFlag, 0, 1) {
					close(receiver.bufferEmptyOnTerminate)
				}
				// exit
				return
			}
		case <-receiver.terminationNotification:
			// we are being forced to terminate while awaiting a message
			return
		}
	}
}

func (receiver *directMessageReceiverImpl) String() string {
	return fmt.Sprintf("solace.DirectMessageReceiver at %p", receiver)
}

type directMessageReceiverBuilderImpl struct {
	internalReceiver core.Receiver
	properties       map[config.ReceiverProperty]interface{}
	subscriptions    []resource.Subscription
}

// NewDirectMessageReceiverBuilderImpl function
func NewDirectMessageReceiverBuilderImpl(internalReceiver core.Receiver) solace.DirectMessageReceiverBuilder {
	return &directMessageReceiverBuilderImpl{
		internalReceiver: internalReceiver,
		properties:       constants.DefaultDirectReceiverProperties.GetConfiguration(),
	}
}

// Build will build a new DirectMessageReceiver with the given properties.
// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
func (builder *directMessageReceiverBuilderImpl) Build() (messageReceiver solace.DirectMessageReceiver, err error) {
	return builder.BuildWithShareName(nil)
}

func (builder *directMessageReceiverBuilderImpl) BuildWithShareName(shareName *resource.ShareName) (messageReceiver solace.DirectMessageReceiver, err error) {
	if shareName != nil {
		if err := validateShareName(shareName.GetName()); err != nil {
			return nil, err
		}
	}
	var receiverBackpressureStrategyString string
	var receiverBackpressureBufferSize int
	if receiverBackpressureStrategyString, _, err = validation.StringPropertyValidation(
		string(config.ReceiverPropertyDirectBackPressureStrategy),
		builder.properties[config.ReceiverPropertyDirectBackPressureStrategy],
		config.ReceiverBackPressureStrategyDropLatest,
		config.ReceiverBackPressureStrategyDropOldest,
	); err != nil {
		return nil, err
	}
	if receiverBackpressureBufferSize, _, err = validation.IntegerPropertyValidation(
		string(config.ReceiverPropertyDirectBackPressureBufferCapacity),
		builder.properties[config.ReceiverPropertyDirectBackPressureBufferCapacity],
	); err != nil {
		return nil, err
	}
	if receiverBackpressureBufferSize < 1 {
		return nil, solace.NewError(&solace.InvalidConfigurationError{}, constants.DirectReceiverBackpressureMustBeGreaterThan0, nil)
	}

	// Validate that subscriptions are of correct type
	for _, subscription := range builder.subscriptions {
		if err = checkDirectMessageReceiverSubscriptionType(subscription); err != nil {
			return nil, err
		}
	}

	var receiverBackpressureStrategyEnum receiverBackpressureStrategy
	switch receiverBackpressureStrategyString {
	case config.ReceiverBackPressureStrategyDropLatest:
		receiverBackpressureStrategyEnum = strategyDropLatest
	case config.ReceiverBackPressureStrategyDropOldest:
		receiverBackpressureStrategyEnum = strategyDropOldest
	}

	receiver := &directMessageReceiverImpl{}
	receiver.construct(
		&directMessageReceiverProps{
			internalReceiver:       builder.internalReceiver,
			startupSubscriptions:   builder.subscriptions,
			backpressureStrategy:   receiverBackpressureStrategyEnum,
			backpressureBufferSize: receiverBackpressureBufferSize,
			shareName:              shareName,
		},
	)

	return receiver, nil
}

// WithSubscriptions will set a list of TopicSubscriptions to subscribe
// to when starting the receiver.
func (builder *directMessageReceiverBuilderImpl) WithSubscriptions(topics ...resource.Subscription) solace.DirectMessageReceiverBuilder {
	builder.subscriptions = topics
	return builder
}

// FromConfigurationProvider will configure the direct receiver with the given properties.
// Built in ReceiverPropertiesConfigurationProvider implementations include:
//   ReceiverPropertyMap, a map of ReceiverProperty keys to values
func (builder *directMessageReceiverBuilderImpl) FromConfigurationProvider(provider config.ReceiverPropertiesConfigurationProvider) solace.DirectMessageReceiverBuilder {
	if provider == nil {
		return builder
	}
	for key, value := range provider.GetConfiguration() {
		builder.properties[key] = value
	}
	return builder
}

// OnBackPressureDropLatest will configure the receiver with the given buffer size. If the buffer
// is full and a message arrives, the incoming message will be discarded.
// bufferCapacity must be >= 1
func (builder *directMessageReceiverBuilderImpl) OnBackPressureDropLatest(bufferCapacity uint) solace.DirectMessageReceiverBuilder {
	return builder.FromConfigurationProvider(config.ReceiverPropertyMap{
		config.ReceiverPropertyDirectBackPressureBufferCapacity: bufferCapacity,
		config.ReceiverPropertyDirectBackPressureStrategy:       config.ReceiverBackPressureStrategyDropLatest,
	})
}

// OnBackPressureDropOldest will configure the receiver with the given buffer size. If the buffer
// is full and a message arrives, the oldest message in the buffer will be discarded.
// bufferCapacity must be >= 1
func (builder *directMessageReceiverBuilderImpl) OnBackPressureDropOldest(bufferCapacity uint) solace.DirectMessageReceiverBuilder {
	return builder.FromConfigurationProvider(config.ReceiverPropertyMap{
		config.ReceiverPropertyDirectBackPressureBufferCapacity: bufferCapacity,
		config.ReceiverPropertyDirectBackPressureStrategy:       config.ReceiverBackPressureStrategyDropOldest,
	})
}

func (builder *directMessageReceiverBuilderImpl) String() string {
	return fmt.Sprintf("solace.DirectMessageReceiverBuilder at %p", builder)
}

// Validate the subscription type is one supported by
func checkDirectMessageReceiverSubscriptionType(subscription resource.Subscription) error {
	switch subscription.(type) {
	case *resource.TopicSubscription:
		return nil
	}
	return solace.NewError(&solace.IllegalArgumentError{}, fmt.Sprintf(constants.DirectReceiverUnsupportedSubscriptionType, subscription), nil)
}

// disallow > and * characters
var validateShareNamePattern, _ = regexp.Compile(`.*[\>\*].*`)

func validateShareName(name string) error {
	if name == "" {
		return solace.NewError(&solace.IllegalArgumentError{}, constants.ShareNameMustNotBeEmpty, nil)
	}
	if validateShareNamePattern.MatchString(name) {
		return solace.NewError(&solace.IllegalArgumentError{}, constants.ShareNameMustNotContainInvalidCharacters, nil)
	}
	return nil
}

type receiverTerminationEvent struct {
	eventTime time.Time
	cause     error
}

// GetTimestamp retrieves the timestamp of the event.
func (event *receiverTerminationEvent) GetTimestamp() time.Time {
	return event.eventTime
}

// GetMessage retrieves the event message.
func (event *receiverTerminationEvent) GetMessage() string {
	return fmt.Sprintf("Receiver Termination Event - timestamp: %s, cause: %s", event.eventTime, event.cause)
}

// GetCause retrieves the cause of the client exception if any.
// Returns the error event or nil if no cause is present.
func (event *receiverTerminationEvent) GetCause() error {
	return event.cause
}
