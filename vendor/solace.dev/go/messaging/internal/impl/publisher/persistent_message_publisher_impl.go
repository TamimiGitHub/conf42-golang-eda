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

package publisher

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/internal/impl/executor"

	"solace.dev/go/messaging/internal/impl/publisher/buffer"

	"solace.dev/go/messaging/internal/impl/logging"

	"solace.dev/go/messaging/internal/ccsmp"

	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/internal/impl/message"

	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/config"
	apimessage "solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/resource"
)

type persistentPublishable struct {
	message     *message.OutboundMessageImpl
	destination resource.Destination
	corr        correlationContext
}

type persistentMessagePublisherImpl struct {
	basicMessagePublisher
	logger logging.LogLevelLogger

	backpressureConfiguration backpressureConfiguration
	buffer                    chan *persistentPublishable
	taskBuffer                buffer.PublisherTaskBuffer

	terminateWaitInterrupt chan struct{}

	downEventHandlerID    uint
	canSendEventHandlerID uint

	publishReceiptListener unsafe.Pointer

	acknowledgementHandlerID uint64
	generateCorrelationTag   func() (uint64, []byte)

	correlationMap           map[uint64](correlationContext)
	correlationLock          *sync.Mutex
	correlationComplete      chan struct{}
	requestCorrelateComplete chan struct{}
}

func (publisher *persistentMessagePublisherImpl) construct(internalPublisher core.Publisher, backpressureConfig backpressureConfiguration, bufferSize int) {
	publisher.basicMessagePublisher.construct(internalPublisher)
	publisher.backpressureConfiguration = backpressureConfig
	if publisher.backpressureConfiguration != backpressureConfigurationDirect {
		// allocate buffers
		publisher.buffer = make(chan *persistentPublishable, bufferSize)
		publisher.taskBuffer = buffer.NewChannelBasedPublisherTaskBuffer(bufferSize, publisher.internalPublisher.TaskQueue)
	}
	publisher.terminateWaitInterrupt = make(chan struct{})
	publisher.logger = logging.For(publisher)

	publisher.correlationMap = make(map[uint64]correlationContext)
	publisher.correlationLock = &sync.Mutex{}
	publisher.correlationComplete = make(chan struct{})
	publisher.requestCorrelateComplete = make(chan struct{})
}

func (publisher *persistentMessagePublisherImpl) onDownEvent(eventInfo core.SessionEventInfo) {
	go publisher.unsolicitedTermination(eventInfo)
}

func (publisher *persistentMessagePublisherImpl) onCanSend(eventInfo core.SessionEventInfo) {
	// We want to offload from the context thread whenever possible, thus we will pass this
	// task off to a new goroutine. This should be sufficient as you are guaranteed to get the
	// can send, it is just not immediate.
	go publisher.notifyReady()
}

func (publisher *persistentMessagePublisherImpl) ackHandler(messageID uint64, persisted bool, err error) {
	publisher.correlationLock.Lock()
	context, ok := publisher.correlationMap[messageID]
	publisher.correlationLock.Unlock()
	if ok {
		// resolve the context OUTSIDE of the correlation table lock as we do not want to block registration of new contexts
		context.resolve(persisted, err)
		// reacquire the lock
		publisher.correlationLock.Lock()
		delete(publisher.correlationMap, messageID)
		// Check state and then check if we should mark the correlation map as complete under the correlation lock
		if publisher.getState() == messagePublisherStateTerminating {
			select {
			case <-publisher.requestCorrelateComplete:
				// we have been asked to give a correlation complete
				if len(publisher.correlationMap) == 0 {
					close(publisher.correlationComplete)
				}
			default:
				// we haven't been asked to give a correlation complete signal yet
			}
		}
		publisher.correlationLock.Unlock()
	}
}

// Start will start the service synchronously.
// Before this function is called, the service is considered
// off-duty. To operate normally, this function must be called on
// a receiver or publisher instance. This function is idempotent.
// Returns an error if one occurred or nil if successful.
func (publisher *persistentMessagePublisherImpl) Start() (err error) {
	// TODO we need to handle the case of no GM?
	// this will block until we are started if we are not first
	if proceed, err := publisher.starting(); !proceed {
		return err
	}
	publisher.logger.Debug("Start persistent receiver start")
	defer func() {
		if err == nil {
			publisher.started(err)
			publisher.logger.Debug("Start publisher complete")
		} else {
			publisher.logger.Debug("Start complete with error: " + err.Error())
			publisher.internalPublisher.Events().RemoveEventHandler(publisher.downEventHandlerID)
			publisher.internalPublisher.Events().RemoveEventHandler(publisher.canSendEventHandlerID)
			publisher.terminated(nil)
			publisher.startFuture.Complete(err)
		}
	}()
	publisher.downEventHandlerID = publisher.internalPublisher.Events().AddEventHandler(core.SolClientEventDown, publisher.onDownEvent)
	publisher.acknowledgementHandlerID, publisher.generateCorrelationTag = publisher.internalPublisher.Acknowledgements().AddAcknowledgementHandler(publisher.ackHandler)
	// startup functionality
	if publisher.backpressureConfiguration != backpressureConfigurationDirect {
		go publisher.taskBuffer.Run()
	} else {
		// if we are persistent, we want to register to receive can send events
		publisher.canSendEventHandlerID = publisher.internalPublisher.Events().AddEventHandler(core.SolClientEventCanSend, publisher.onCanSend)
	}
	go publisher.eventExecutor.Run()
	return nil
}

// StartAsync will start the service asynchronously.
// Before this function is called, the service is considered
// off-duty. To operate normally, this function must be called on
// a receiver or publisher instance. This function is idempotent.
// Returns a channel that will receive an error if one occurred or
// nil if successful. Subsequent calls will return additional
// channels that can await an error, or nil if already started.
func (publisher *persistentMessagePublisherImpl) StartAsync() <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- publisher.Start()
		close(result)
	}()
	return result
}

// StartAsyncCallback will start the PersistentMessagePublisher asynchronously.
// Calls the callback when started with an error if one occurred or nil
// if successful.
func (publisher *persistentMessagePublisherImpl) StartAsyncCallback(callback func(solace.PersistentMessagePublisher, error)) {
	go func() {
		callback(publisher, publisher.Start())
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
func (publisher *persistentMessagePublisherImpl) Terminate(gracePeriod time.Duration) (err error) {
	if proceed, err := publisher.terminate(); !proceed {
		return err
	}
	publisher.logger.Debug("Terminate persistent receiver start")
	// make sure the service is marked as terminated
	defer func() {
		publisher.terminated(err)
		if err != nil {
			publisher.logger.Debug("Terminate complete with error: " + err.Error())
		} else {
			publisher.logger.Debug("Terminate complete")
		}
	}()

	// We're terminating, we do not care about the down event handler anymore
	publisher.internalPublisher.Events().RemoveEventHandler(publisher.downEventHandlerID)

	defer func() {
		publisher.logger.Debug("Awaiting termination of event executor")
		// Allow the event executor to terminate, blocking until it does
		publisher.eventExecutor.AwaitTermination()
	}()

	// We first want to gracefully shut down the task buffer, and keep track of how long that takes
	// signal that we should try and terminate gracefully
	var timer *time.Timer
	if gracePeriod >= 0 {
		timer = time.NewTimer(gracePeriod)
	}
	// start by shutting down the task buffer
	var graceful = true

	publisher.logger.Debug("Have buffered backpressure, terminating the task buffer gracefully")
	if publisher.backpressureConfiguration != backpressureConfigurationDirect {
		// First we interrupt all backpressure wait functions
		close(publisher.terminateWaitInterrupt)
		graceful = publisher.taskBuffer.Terminate(timer)
	} else {
		publisher.internalPublisher.Events().RemoveEventHandler(publisher.canSendEventHandlerID)
	}

	// wait for the task buffer to drain
	if !graceful {
		publisher.logger.Debug("Task buffer terminated ungracefully, will not wait for acknowledgements to be processed")
	} else {
		publisher.logger.Debug("Waiting for correlation table to drain")
		// wait for acks
		publisher.correlationLock.Lock()
		remainingAcks := len(publisher.correlationMap)
		close(publisher.requestCorrelateComplete)
		publisher.correlationLock.Unlock()
		if remainingAcks > 0 {
			if timer != nil {
				select {
				case <-publisher.correlationComplete:
					// success
				case <-timer.C:
					graceful = false
				}
			} else {
				// Block forever as our grace period is negative
				<-publisher.correlationComplete
			}
		}
	}

	// Make sure we stop receiving acknowledgements
	publisher.internalPublisher.Acknowledgements().RemoveAcknowledgementHandler(publisher.acknowledgementHandlerID)

	// Next we close the buffer, failing any racing publishes
	// This must happen before we count the number of messages as we cannot allow any more messages
	// in before counting.
	// check that all messages have been delivered, and return an error if they have not been

	publisherTerminatedError := solace.NewError(&solace.IncompleteMessageDeliveryError{}, constants.UnableToPublishAlreadyTerminated, nil)
	var undeliveredCount uint64 = 0
	if publisher.backpressureConfiguration != backpressureConfigurationDirect {
		// only drain the queue if we are in buffered backpressure scenarios
		undeliveredCount = publisher.drainQueue(publisherTerminatedError)
	}
	unackedCount := publisher.drainAckTable(publisherTerminatedError)
	if undeliveredCount > 0 || unackedCount > 0 {
		if publisher.logger.IsDebugEnabled() {
			// we should not have any messages, but if somehow one snuck in and was not published, we should log it.
			if graceful {
				publisher.logger.Debug(fmt.Sprintf("Expected graceful shutdown but had %d undelivered messages and %d unacknowledged messages", undeliveredCount, unackedCount))
			} else {
				publisher.logger.Debug(fmt.Sprintf("Terminating with %d undelivered messages, %d unacked messages", undeliveredCount, unackedCount))
			}
		}
		// return an error if we have one
		publisher.internalPublisher.IncrementMetric(core.MetricPublishMessagesTerminationDiscarded, undeliveredCount)
		err := solace.NewError(&solace.IncompleteMessageDeliveryError{}, fmt.Sprintf(constants.IncompleteMessageDeliveryMessageWithUnacked, undeliveredCount, unackedCount), nil)
		return err
	}
	// finish cleanup successfully
	return nil
}

func (publisher *persistentMessagePublisherImpl) unsolicitedTermination(errorInfo core.SessionEventInfo) {
	if proceed, _ := publisher.terminate(); !proceed {
		return
	}
	if publisher.logger.IsDebugEnabled() {
		publisher.logger.Debug("Received unsolicited termination with event info " + errorInfo.GetInfoString())
		defer publisher.logger.Debug("Unsolicited termination complete")
	}
	timestamp := time.Now()
	publisher.internalPublisher.Events().RemoveEventHandler(publisher.downEventHandlerID)
	publisher.internalPublisher.Acknowledgements().RemoveAcknowledgementHandler(publisher.acknowledgementHandlerID)

	var err error = nil
	var undeliveredCount uint64 = 0
	// Start shutdown, some behaviour depends on buffered vs unbuffered
	if publisher.backpressureConfiguration != backpressureConfigurationDirect {
		// Interrupt publish with wait
		close(publisher.terminateWaitInterrupt)
		// Stop processing new messages
		publisher.taskBuffer.TerminateNow()
		// Drain the queue and count messages
		undeliveredCount = publisher.drainQueue(errorInfo.GetError())
	} else {
		// In unbuffered backpressure all we need to do is remove the can send handler id
		publisher.internalPublisher.Events().RemoveEventHandler(publisher.canSendEventHandlerID)
	}

	// check that all messages have been delivered, and return an error if they have not been
	unackedCount := publisher.drainAckTable(errorInfo.GetError())
	if undeliveredCount > 0 || unackedCount > 0 {
		if publisher.logger.IsDebugEnabled() {
			publisher.logger.Debug(fmt.Sprintf("Terminated with %d undelivered messages and %d unacked messages", undeliveredCount, unackedCount))
		}
		// return an error if we have one
		err = solace.NewError(&solace.IncompleteMessageDeliveryError{}, fmt.Sprintf(constants.IncompleteMessageDeliveryMessageWithUnacked, undeliveredCount, unackedCount), nil)
		publisher.internalPublisher.IncrementMetric(core.MetricPublishMessagesTerminationDiscarded, undeliveredCount)
	}
	// Stop processing events
	publisher.eventExecutor.Terminate()
	publisher.terminated(err)
	// Call the callback
	if publisher.terminationListener != nil {
		publisher.terminationListener(&publisherTerminationEvent{
			timestamp,
			errorInfo.GetError(),
		})
	}
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
func (publisher *persistentMessagePublisherImpl) TerminateAsync(gracePeriod time.Duration) <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- publisher.Terminate(gracePeriod)
		close(result)
	}()
	return result
}

// TerminateAsyncCallback will terminate the PersistentMessagePublisher asynchronously.
// Calls the callback when terminated with nil if successful or an error if
// one occurred. If gracePeriod is less than 0, the function will wait indefinitely.
func (publisher *persistentMessagePublisherImpl) TerminateAsyncCallback(gracePeriod time.Duration, callback func(error)) {
	go func() {
		callback(publisher.Terminate(gracePeriod))
	}()
}

// SetPublishReceiptListener sets the listener to receive delivery receipts.
// PublishReceipt events are triggered once the API receives an acknowledgement
// of message receival from the Broker.
// This should be set before the publisher is started to avoid dropping acknowledgements.
// The listener does not receive events from PublishAwaitAcknowledgement calls.
func (publisher *persistentMessagePublisherImpl) SetMessagePublishReceiptListener(listener solace.MessagePublishReceiptListener) {
	if listener == nil {
		atomic.StorePointer(&publisher.publishReceiptListener, nil)
	} else {
		atomic.StorePointer(&publisher.publishReceiptListener, unsafe.Pointer(&listener))
	}
}

// IsReady checks if the publisher can publish messages. Returns true if the
// publisher can publish messages, false if the publisher is presvented from
// sending messages (e.g., full buffer or I/O problems)
func (publisher *persistentMessagePublisherImpl) IsReady() bool {
	return publisher.IsRunning() && (publisher.backpressureConfiguration != backpressureConfigurationReject || len(publisher.buffer) != cap(publisher.buffer))
}

// NotifyWhenReady makes a request to notify the application when the
// publisher is ready. This function will block until the publisher
// is ready.
func (publisher *persistentMessagePublisherImpl) NotifyWhenReady() {
	if publisher.IsReady() {
		publisher.notifyReady()
	}
}

// queues a new ready event on the event executor
func (publisher *persistentMessagePublisherImpl) notifyReady() {
	readinessListener := publisher.readinessListener
	if readinessListener != nil {
		publisher.eventExecutor.Submit(executor.Task(readinessListener))
	}
}

// drainQueue will drain the message buffer and return the number of undelivered messages. If given a publish failure listener, it will be
// called with every undelivered message and the error if present
func (publisher *persistentMessagePublisherImpl) drainQueue(err error) uint64 {
	close(publisher.buffer)
	undeliveredCount := uint64(0)
	for undelivered := range publisher.buffer {
		undeliveredCount++
		if undelivered != nil && undelivered.corr != nil {
			// the resolve function should dispose of the message OR pass it back to the application
			undelivered.corr.resolve(false, err)
		}
	}
	return undeliveredCount
}

// drainAckTable will remove all entries from the acknowledgement table
func (publisher *persistentMessagePublisherImpl) drainAckTable(err error) uint64 {
	publisher.correlationLock.Lock()
	defer publisher.correlationLock.Unlock()
	unackedCount := uint64(0)
	for key, value := range publisher.correlationMap {
		unackedCount++
		if value != nil {
			// the resolve function should dispose of the message OR pass it back to the application
			value.resolve(false, err)
		}
		delete(publisher.correlationMap, key)
	}
	return unackedCount
}

// PublishBytes will publish a message of type byte array to the given destination.
// Returns an error if one occurred while attempting to publish or if the publisher
// is not started/terminated. Returns an error if one occurred. Possible errors include
// - solace/solace.*PubSubPlusClientError if the message could not be sent and all retry attempts failed.
// - solace/solace.*PublisherOverflowError if publishing messages faster than publisher's I/O
// capabilities allow. When publishing can be resumed, registered PublisherReadinessListeners
// will be called.
func (publisher *persistentMessagePublisherImpl) PublishBytes(bytes []byte, dest *resource.Topic) error {
	if err := publisher.checkStartedStateForPublish(); err != nil {
		return err
	}
	msg, err := publisher.messageBuilder.BuildWithByteArrayPayload(bytes)
	if err != nil {
		return err
	}
	// we built the message so it is safe to cast
	return publisher.publishWithCallbackContext(msg.(*message.OutboundMessageImpl), dest, nil)
}

// PublishString will publish a message of type string to the given destination.
// Returns an error if one occurred. Possible errors include:
// - solace/solace.*PubSubPlusClientError if the message could not be sent and all retry attempts failed.
// - solace/solace.*PublisherOverflowError if publishing messages faster than publisher's I/O
// capabilities allow. When publishing can be resumed, registered PublisherReadinessListeners
// will be called.
func (publisher *persistentMessagePublisherImpl) PublishString(str string, dest *resource.Topic) error {
	if err := publisher.checkStartedStateForPublish(); err != nil {
		return err
	}
	msg, err := publisher.messageBuilder.BuildWithStringPayload(str)
	if err != nil {
		return err
	}
	// we built the message so it is safe to cast
	return publisher.publishWithCallbackContext(msg.(*message.OutboundMessageImpl), dest, nil)
}

// PublishWithProperties will publish the given message of type OutboundMessage
// with the given properties. These properties will override the properties on
// the OutboundMessage instance if present. Possible errors include:
// - solace/solace.*PubSubPlusClientError if the message could not be sent and all retry attempts failed.
// - solace/solace.*PublisherOverflowError if publishing messages faster than publisher's I/O
// capabilities allow. When publishing can be resumed, registered PublisherReadinessListeners
// will be called.
func (publisher *persistentMessagePublisherImpl) Publish(msg apimessage.OutboundMessage, dest *resource.Topic, properties config.MessagePropertiesConfigurationProvider, userContext interface{}) error {
	if err := publisher.checkStartedStateForPublish(); err != nil {
		return err
	}
	msgDup, err := duplicateMessageAndSetProperties(msg, properties)
	if err != nil {
		return err
	}
	return publisher.publishWithCallbackContext(msgDup, dest, userContext)
}

// PublishAwaitAcknowledgement will publish the given message of type OutboundMessage
// and await a publish acknowledgement.
// Optionally provide properties in the form of OutboundMessageProperties to override
// any properties set on OutboundMessage. The properties argument can be nil to
// not set any properties.
// If timeout is less than 0, the function will wait indefinitely.
// Possible errors include:
// - solace/errors.*PubSubPlusClientError if the message could not be sent and all retry attempts failed.
// - solace/errors.*PublisherOverflowError if publishing messages faster than publisher's I/O
// capabilities allow. When publishing can be resumed, registered PublisherReadinessListeners
// will be called.
func (publisher *persistentMessagePublisherImpl) PublishAwaitAcknowledgement(msg apimessage.OutboundMessage, dest *resource.Topic, timeout time.Duration, properties config.MessagePropertiesConfigurationProvider) error {
	if err := publisher.checkStartedStateForPublish(); err != nil {
		return err
	}
	msgDup, err := duplicateMessageAndSetProperties(msg, properties)
	if err != nil {
		return err
	}
	if err = message.SetAckImmediately(msgDup); err != nil {
		return err
	}
	acknowledgementChannel := make(chan error, 1)
	err = publisher.publishWithBlockingContext(msgDup, dest, acknowledgementChannel)
	if err != nil {
		return err
	}
	if timeout >= 0 {
		timer := time.NewTimer(timeout)
		select {
		case err := <-acknowledgementChannel:
			timer.Stop()
			return err
		case <-timer.C:
			return solace.NewError(&solace.TimeoutError{}, "timed out waiting for message to be acknowledged", nil)
		}
	} else {
		// block forever
		return <-acknowledgementChannel
	}
}

func duplicateMessageAndSetProperties(msg apimessage.OutboundMessage, properties config.MessagePropertiesConfigurationProvider) (*message.OutboundMessageImpl, error) {
	msgImpl, ok := msg.(*message.OutboundMessageImpl)
	if !ok {
		return nil, solace.NewError(&solace.IllegalArgumentError{}, fmt.Sprintf(constants.InvalidOutboundMessageType, msg), nil)
	}
	msgDup, err := message.DuplicateOutboundMessage(msgImpl)
	if err != nil {
		return nil, err
	}
	if properties != nil {
		err := message.SetProperties(msgDup, properties.GetConfiguration())
		if err != nil {
			msgDup.Dispose()
			return nil, err
		}
	}
	return msgDup, nil
}

func (publisher *persistentMessagePublisherImpl) publishWithCallbackContext(msg *message.OutboundMessageImpl, dest *resource.Topic, userContext interface{}) (ret error) {
	ctx := &callbackCorrelationContext{
		callbackPtr:   &publisher.publishReceiptListener,
		eventExecutor: publisher.eventExecutor,
		message:       msg,
		userContext:   userContext,
		logger:        publisher.logger,
	}
	return publisher.publish(msg, dest, ctx, userContext)
}

func (publisher *persistentMessagePublisherImpl) publishWithBlockingContext(msg *message.OutboundMessageImpl, dest *resource.Topic, blocker chan error) (ret error) {
	ctx := &blockingCorrelationContext{
		message: msg,
		blocker: blocker,
	}
	return publisher.publish(msg, dest, ctx, nil)
}

// publish impl taking a dup'd message, assuming state has been checked and we are running
func (publisher *persistentMessagePublisherImpl) publish(msg *message.OutboundMessageImpl, dest *resource.Topic, ctx correlationContext, userContext interface{}) (ret error) {

	// Set the destination for the message which is assumed to be a dup'd message.
	err := message.SetDestination(msg, dest.GetName())
	if err != nil {
		msg.Dispose()
		return err
	}
	err = message.SetDeliveryMode(msg, message.DeliveryModePersistent)
	if err != nil {
		msg.Dispose()
		return err
	}

	var messageID uint64
	if ctx != nil {
		var correlationTag []byte
		messageID, correlationTag = publisher.generateCorrelationTag()
		err = message.AttachCorrelationTag(msg, correlationTag)
		if err != nil {
			// message will always be duplicated, we should free it before returning the error
			msg.Dispose()
			return err
		}
	}

	// check the state once more before moving into the publish paths, this closes the race condition windows a little more
	if err = publisher.checkStartedStateForPublish(); err != nil {
		return err
	}
	if publisher.backpressureConfiguration == backpressureConfigurationDirect {
		// publish persistently with CCSMP
		if !publisher.addCorrelationContext(messageID, ctx) {
			// we can only dispose if we have no correlation context
			defer msg.Dispose()
		}
		errorInfo := publisher.internalPublisher.Publish(message.GetOutboundMessagePointer(msg))
		if errorInfo != nil {
			// if we encountered an error publishing, we should not attach the correlation
			publisher.removeCorrelationContext(messageID)
			if errorInfo.ReturnCode == ccsmp.SolClientReturnCodeWouldBlock {
				return solace.NewError(&solace.PublisherOverflowError{}, constants.WouldBlock, nil)
			}
			return core.ToNativeError(errorInfo)
		}
	} else {
		// buffered backpressure scenarios

		// this section is to handle the case where a publish proceeds after we have moved to terminating, specifically
		// in the ungraceful termination case, and we have decided that no more messages should be published, thus the
		// message queue is closed. The window for this race is very small, but it is still worth handling.
		channelWrite := false
		defer func() {
			if !channelWrite {
				// we have not written to the channel yet, we may or may not have received panic, so check
				if r := recover(); r != nil {
					// if we have a panic, and that panic is send on closed channel, we can return an error by setting "ret", otherwise repanic
					if err, ok := r.(error); ok && err.Error() == "send on closed channel" {
						publisher.logger.Debug("Caught a channel closed panic when trying to write to the message buffer, publisher must be terminated.")
						ret = solace.NewError(&solace.IllegalStateError{}, constants.UnableToPublishAlreadyTerminated, nil)
					} else {
						// this shouldn't ever happen, but panics are unpredictable. We want this message to make it into the logs
						publisher.logger.Error(fmt.Sprintf("Experienced panic while attempting to publish a message: %s", err))
						panic(r)
					}
				}
			}
		}()
		pub := &persistentPublishable{msg, dest, ctx}
		if publisher.backpressureConfiguration == backpressureConfigurationReject {
			select {
			case publisher.buffer <- pub:
				channelWrite = true // we successfully wrote the message to the channel
			default:
				return solace.NewError(&solace.PublisherOverflowError{}, constants.WouldBlock, nil)
			}
		} else {
			// wait forever
			select {
			case publisher.buffer <- pub:
				channelWrite = true // we successfully wrote the message to the channel
			case <-publisher.terminateWaitInterrupt:
				return solace.NewError(&solace.IllegalStateError{}, constants.UnableToPublishAlreadyTerminated, nil)
			}
		}
		// if we successfully wrote to the channel (which should always be true at this point), submit the task and terminate.
		if !publisher.taskBuffer.Submit(publisher.sendTask(msg, dest, messageID, ctx)) {
			// if we couldn't submit the task, log. This may happen on shutdown in a race between the task buffer shutting down
			// and the message buffer being drained, at which point we are terminating ungracefully.
			publisher.logger.Debug("Attempted to submit the message for publishing, but the task buffer rejected the task! Has the service been terminated?")
			// At this point, we have a message that made it into the buffer but the task did not get submitted.
			// This message will be counted as "not delivered" when terminate completes.
			// This is very unlikely as the message buffer is closed much earlier than the task buffer,
			// so this window is very small. It is best to handle this when we can though.
		}
	}
	return nil
}

// if given a correlation context, acquire the correlation lock and add a new entry to the map returning true if the context was successfully added
func (publisher *persistentMessagePublisherImpl) addCorrelationContext(messageID uint64, ctx correlationContext) bool {
	if ctx != nil {
		publisher.correlationLock.Lock()
		publisher.correlationMap[messageID] = ctx
		publisher.correlationLock.Unlock()
		return true
	}
	return false
}

// removes the given message id under the correlation lock from the correlation map. this does not call the correlation context stored at the messageId
func (publisher *persistentMessagePublisherImpl) removeCorrelationContext(messageID uint64) {
	publisher.correlationLock.Lock()
	delete(publisher.correlationMap, messageID)
	publisher.correlationLock.Unlock()
}

// sendTask represents the task that is submitted to the internal task buffer and ultimately the shared serialized publisher instance
// returned closure accepts a channel that will receive a notification when any waits should be interrupted
func (publisher *persistentMessagePublisherImpl) sendTask(msg *message.OutboundMessageImpl, dest resource.Destination, messageID uint64, ctx correlationContext) buffer.PublisherTask {
	return func(terminateChannel chan struct{}) {
		var errorInfo core.ErrorInfo
		// save the context to the correlation map
		publisher.addCorrelationContext(messageID, ctx)
		// main publish loop
		for {
			// attempt a publish
			errorInfo = publisher.internalPublisher.Publish(message.GetOutboundMessagePointer(msg))
			if errorInfo != nil {
				// if we got a would block, wait for ready and retry
				if errorInfo.ReturnCode == ccsmp.SolClientReturnCodeWouldBlock {
					err := publisher.internalPublisher.AwaitWritable(terminateChannel)
					if err != nil {
						// if we encountered an error while waiting for writable, the publisher will shut down
						// and this task will not complete. The message queue will be drained by the caller of
						// terminate, so we should not deal with the message.

						// we should also clear the correlation context so we don't double count the message
						// or double call the correlation context
						publisher.removeCorrelationContext(messageID)
						return
					}
					continue
					// otherwise we got another error, should deal with it accordingly
				}
			}
			// exit out of the loop if we succeeded or got an error
			// we will only continue on would_block + AwaitWritable
			break
		}
		isFull := len(publisher.buffer) == cap(publisher.buffer)
		// remove msg from buffer, should be guaranteed to be there, but we don't want to deadlock in case something went wonky.
		// shutdown is contingent on all active tasks completing.
		select {
		case pub, ok := <-publisher.buffer:
			if ok {
				// if we got an error, first we must remove the context correlation if present
				if errorInfo != nil && ctx != nil {
					// make sure we clean up the correlation entry since it will never be dealt with
					publisher.correlationLock.Lock()
					delete(publisher.correlationMap, messageID)
					publisher.correlationLock.Unlock()
					// then we must resolve the context
					ctx.resolve(false, core.ToNativeError(errorInfo, "encountered error while publishing message: "))
				} else if ctx == nil {
					// if we got no error and there is no context associated with this message, dispose of the message
					pub.message.Dispose()
				}
				// check if we should signal that the buffer has space
				// we only have to call the publisher notification of being ready when we
				// have successfully popped a message off the buffer
				if isFull && publisher.backpressureConfiguration == backpressureConfigurationReject {
					publisher.notifyReady()
				}
			}
			// We must have a closed buffer with no more messages. Since the buffer was closed, we can safely ignore the message.
			// This is because the
		default:
			// should never happen as the message queue should always be drained after
			publisher.logger.Error("published a message after publisher buffer was drained, this is unexpected")
		}
	}
}

func (publisher *persistentMessagePublisherImpl) String() string {
	return fmt.Sprintf("solace.PersistentMessagePublisher at %p", publisher)
}

type persistentMessagePublisherBuilderImpl struct {
	internalPublisher core.Publisher
	properties        map[config.PublisherProperty]interface{}
}

// NewPersistentMessagePublisherBuilderImpl function
func NewPersistentMessagePublisherBuilderImpl(internalPublisher core.Publisher) solace.PersistentMessagePublisherBuilder {
	return &persistentMessagePublisherBuilderImpl{
		internalPublisher: internalPublisher,
		// default properties
		properties: constants.DefaultPersistentPublisherProperties.GetConfiguration(),
	}
}

// Build will build a new PersistentMessagePublisher instance based on the configured properties.
// Returns solace/solace.*InvalidConfigurationError if an invalid configuration is provided.
func (builder *persistentMessagePublisherBuilderImpl) Build() (messagePublisher solace.PersistentMessagePublisher, err error) {
	var publisherBackpressureBufferSize int
	var backpressureConfig backpressureConfiguration
	backpressureConfig, publisherBackpressureBufferSize, err = validateBackpressureConfig(builder.properties)
	if err != nil {
		return nil, err
	}
	publisher := &persistentMessagePublisherImpl{}
	publisher.construct(builder.internalPublisher, backpressureConfig, publisherBackpressureBufferSize)
	return publisher, nil
}

// OnBackPressureReject will set the publisher backpressure strategy to reject
// where publish attempts will be rejected once the bufferSize, in number of messages, is reached.
// If bufferSize is 0, an error will be thrown when the transport is full when publishing.
// Valid bufferSize is >= 0.
func (builder *persistentMessagePublisherBuilderImpl) OnBackPressureReject(bufferSize uint) solace.PersistentMessagePublisherBuilder {
	builder.properties[config.PublisherPropertyBackPressureStrategy] = config.PublisherPropertyBackPressureStrategyBufferRejectWhenFull
	builder.properties[config.PublisherPropertyBackPressureBufferCapacity] = bufferSize
	return builder
}

// OnBackPressureWait will set the publisher backpressure strategy to wait where publish
// attempts will block until there is space in the buffer of size bufferSize in number of messages.
// Valid bufferSize is >= 1.
func (builder *persistentMessagePublisherBuilderImpl) OnBackPressureWait(bufferSize uint) solace.PersistentMessagePublisherBuilder {
	builder.properties[config.PublisherPropertyBackPressureStrategy] = config.PublisherPropertyBackPressureStrategyBufferWaitWhenFull
	builder.properties[config.PublisherPropertyBackPressureBufferCapacity] = bufferSize
	return builder
}

// FromConfigurationProvider will configure the persistent publisher with the given properties.
// Built in PublisherPropertiesConfigurationProvider implementations include:
//   PublisherPropertyMap, a map of PublisherProperty keys to values
//   for loading of properties from a string configuration (files or other configuration source)
func (builder *persistentMessagePublisherBuilderImpl) FromConfigurationProvider(provider config.PublisherPropertiesConfigurationProvider) solace.PersistentMessagePublisherBuilder {
	if provider == nil {
		return builder
	}
	for key, value := range provider.GetConfiguration() {
		builder.properties[key] = value
	}
	return builder
}

func (builder *persistentMessagePublisherBuilderImpl) String() string {
	return fmt.Sprintf("solace.PersistentMessagePublisherBuilder at %p", builder)
}

type publishReceipt struct {
	userContext interface{}
	message     apimessage.OutboundMessage
	timestamp   time.Time
	err         error
	persisted   bool
}

// GetUserContext gets the context associated with the publish if provided.
// Returns nil if no user context is set.
func (receipt *publishReceipt) GetUserContext() interface{} {
	return receipt.userContext
}

// GetTimeStamp gets the time.Time that the event occurred, specifically the time when the
// acknowledgement was received by the API from the broker, or when the GetError error occurred
// if present.
func (receipt *publishReceipt) GetTimeStamp() time.Time {
	return receipt.timestamp
}

// GetMessage will return the OutboundMessage instance that was successfully published.
func (receipt *publishReceipt) GetMessage() apimessage.OutboundMessage {
	return receipt.message
}

// GetError will get the error that occurred, if any, indicating a failure to publish.
// GetError will return nil on a successful publish, or an error if a failure occurred
// while delivering the message.
func (receipt *publishReceipt) GetError() error {
	return receipt.err
}

// IsPersisted returns true if the broker confirmed that the message was received and persisted,
// false otherwise.
func (receipt *publishReceipt) IsPersisted() bool {
	return receipt.persisted
}

type correlationContext interface {
	resolve(bool, error)
}

type callbackCorrelationContext struct {
	callbackPtr   *unsafe.Pointer
	message       apimessage.OutboundMessage
	userContext   interface{}
	eventExecutor executor.Executor
	logger        logging.LogLevelLogger
}

func (context *callbackCorrelationContext) resolve(persisted bool, err error) {
	callbackPtr := (*solace.MessagePublishReceiptListener)(atomic.LoadPointer(context.callbackPtr))
	if callbackPtr != nil {
		callback := *callbackPtr
		receipt := context.createReceipt(persisted, err)
		if !context.eventExecutor.Submit(func() { callback(receipt) }) && context.logger.IsInfoEnabled() {
			context.logger.Info(fmt.Sprintf("Failed to submit callback for publish receipt %v. Is the publisher terminated?", receipt))
		}
	} else {
		// if we don't have a callback, we should free the dup'd message
		context.message.Dispose()
	}
}

func (context *callbackCorrelationContext) createReceipt(persisted bool, err error) solace.PublishReceipt {
	return &publishReceipt{
		userContext: context.userContext,
		message:     context.message,
		timestamp:   time.Now(),
		err:         err,
		persisted:   persisted,
	}
}

type blockingCorrelationContext struct {
	blocker chan error
	message apimessage.OutboundMessage
}

func (context *blockingCorrelationContext) resolve(persisted bool, err error) {
	context.blocker <- err
	context.message.Dispose()
}
