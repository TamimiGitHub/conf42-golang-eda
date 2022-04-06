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

// Package publisher is defined below
package publisher

import (
	"fmt"
	"time"

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

type directMessagePublisherImpl struct {
	basicMessagePublisher
	logger logging.LogLevelLogger

	downEventHandlerID    uint
	canSendEventHandlerID uint

	// callbacks and such
	publishFailureListener solace.PublishFailureListener

	// the parameters for backpressure
	backpressureConfiguration backpressureConfiguration
	// buffers for backpressure
	buffer     chan *publishable
	taskBuffer buffer.PublisherTaskBuffer

	terminateWaitInterrupt chan struct{}
}

func (publisher *directMessagePublisherImpl) construct(internalPublisher core.Publisher, backpressureConfig backpressureConfiguration, bufferSize int) {
	publisher.basicMessagePublisher.construct(internalPublisher)
	publisher.backpressureConfiguration = backpressureConfig
	if publisher.backpressureConfiguration != backpressureConfigurationDirect {
		// allocate buffers
		publisher.buffer = make(chan *publishable, bufferSize)
		publisher.taskBuffer = buffer.NewChannelBasedPublisherTaskBuffer(bufferSize, publisher.internalPublisher.TaskQueue)
	}
	publisher.terminateWaitInterrupt = make(chan struct{})
	publisher.logger = logging.For(publisher)
}

func (publisher *directMessagePublisherImpl) onDownEvent(eventInfo core.SessionEventInfo) {
	go publisher.unsolicitedTermination(eventInfo)
}

func (publisher *directMessagePublisherImpl) onCanSend(eventInfo core.SessionEventInfo) {
	// We want to offload from the context thread whenever possible, thus we will pass this
	// task off to a new goroutine. This should be sufficient as you are guaranteed to get the
	// can send, it is just not immediate.
	go publisher.notifyReady()
}

// Start will start the service synchronously.
// Before this function is called, the service is considered
// off-duty. To operate normally, this function must be called on
// a receiver or publisher instance. This function is idempotent.
// Returns an error if one occurred or nil if successful.
func (publisher *directMessagePublisherImpl) Start() (err error) {
	// this will block until we are started if we are not first
	if proceed, err := publisher.starting(); !proceed {
		return err
	}
	publisher.logger.Debug("Start direct publisher start")
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
	// startup functionality
	if publisher.backpressureConfiguration != backpressureConfigurationDirect {
		go publisher.taskBuffer.Run()
	} else {
		// if we are direct, we want to register to receive can send events
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
func (publisher *directMessagePublisherImpl) StartAsync() <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- publisher.Start()
		close(result)
	}()
	return result
}

// StartAsyncCallback will start the DirectMessagePublisher asynchronously.
// Calls the callback when started with an error if one occurred or nil
// if successful.
func (publisher *directMessagePublisherImpl) StartAsyncCallback(callback func(solace.DirectMessagePublisher, error)) {
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
func (publisher *directMessagePublisherImpl) Terminate(gracePeriod time.Duration) (err error) {
	if proceed, err := publisher.terminate(); !proceed {
		return err
	}
	publisher.logger.Debug("Terminate direct publisher start")
	// make sure the service is marked as terminated
	defer func() {
		publisher.terminated(err)
		if err != nil {
			publisher.logger.Debug("Terminate complete with error: " + err.Error())
		} else {
			publisher.logger.Debug("Terminate complete")
		}
	}()

	defer func() {
		publisher.logger.Debug("Awaiting termination of event executor")
		// Allow the event executor to terminate, blocking until it does
		publisher.eventExecutor.AwaitTermination()
	}()

	// We're terminating, we do not care about the down event handler anymore
	publisher.internalPublisher.Events().RemoveEventHandler(publisher.downEventHandlerID)

	// We first want to gracefully shut down the task buffer, and keep track of how long that takes
	// signal that we should try and terminate gracefully
	if publisher.backpressureConfiguration != backpressureConfigurationDirect {
		publisher.logger.Debug("Have buffered backpressure, terminating the task buffer gracefully")
		// First we interrupt all backpressure wait functions
		close(publisher.terminateWaitInterrupt)

		var timer *time.Timer
		if gracePeriod >= 0 {
			timer = time.NewTimer(gracePeriod)
		}
		// start by shutting down the task buffer
		graceful := publisher.taskBuffer.Terminate(timer)
		// adjust the grace period to the remaining time (approximate)
		if !graceful {
			publisher.logger.Debug("Task buffer terminated ungracefully")
		}

		// Next we close the buffer, failing any racing publishes
		// This must happen before we count the number of messages as we cannot allow any more messages
		// in before counting.
		// check that all messages have been delivered, and return an error if they have not been
		undeliveredCount := publisher.drainQueue(time.Now(), nil, nil)
		if undeliveredCount > 0 {
			if publisher.logger.IsDebugEnabled() {
				// we should not have any messages, but if somehow one snuck in and was not published, we should log it.
				if graceful {
					publisher.logger.Debug(fmt.Sprintf("Expected message buffer to be empty on graceful shutdown, but it had length %d", undeliveredCount))
				} else {
					publisher.logger.Debug(fmt.Sprintf("Terminating with %d undelivered messages", undeliveredCount))
				}
			}
			// return an error if we have one
			publisher.internalPublisher.IncrementMetric(core.MetricPublishMessagesTerminationDiscarded, uint64(undeliveredCount))
			err := solace.NewError(&solace.IncompleteMessageDeliveryError{}, fmt.Sprintf(constants.IncompleteMessageDeliveryMessage, undeliveredCount), nil)
			return err
		}
	} else {
		publisher.internalPublisher.Events().RemoveEventHandler(publisher.canSendEventHandlerID)
	}
	// finish cleanup successfully
	return nil
}

func (publisher *directMessagePublisherImpl) unsolicitedTermination(errorInfo core.SessionEventInfo) {
	if proceed, _ := publisher.terminate(); !proceed {
		return
	}
	if publisher.logger.IsDebugEnabled() {
		publisher.logger.Debug("Received unsolicited termination with event info " + errorInfo.GetInfoString())
		defer publisher.logger.Debug("Unsolicited termination complete")
	}
	timestamp := time.Now()
	publisher.internalPublisher.Events().RemoveEventHandler(publisher.downEventHandlerID)

	var err error = nil
	if publisher.backpressureConfiguration != backpressureConfigurationDirect {
		close(publisher.terminateWaitInterrupt)
		// Close the task buffer without waiting for any more tasks to be processed
		publisher.taskBuffer.TerminateNow()
		// check that all messages have been delivered, and return an error if they have not been
		undeliveredCount := publisher.drainQueue(timestamp, errorInfo.GetError(), publisher.publishFailureListener)
		if undeliveredCount > 0 {
			if publisher.logger.IsDebugEnabled() {
				publisher.logger.Debug(fmt.Sprintf("Terminated with %d undelivered messages", undeliveredCount))
			}
			// return an error if we have one
			err = solace.NewError(&solace.IncompleteMessageDeliveryError{}, fmt.Sprintf(constants.IncompleteMessageDeliveryMessage, undeliveredCount), nil)
			publisher.internalPublisher.IncrementMetric(core.MetricPublishMessagesTerminationDiscarded, undeliveredCount)
		}
	} else {
		publisher.internalPublisher.Events().RemoveEventHandler(publisher.canSendEventHandlerID)
	}
	// Terminate the event executor without waiting for the termination to complete
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
func (publisher *directMessagePublisherImpl) TerminateAsync(gracePeriod time.Duration) <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- publisher.Terminate(gracePeriod)
		close(result)
	}()
	return result
}

// TerminateAsyncCallback will terminate the DirectMessagePublisher asynchronously.
// Calls the callback when terminated with nil if successful or an error if
// one occurred. If gracePeriod is less than 0, the function will wait indefinitely.
func (publisher *directMessagePublisherImpl) TerminateAsyncCallback(gracePeriod time.Duration, callback func(error)) {
	go func() {
		callback(publisher.Terminate(gracePeriod))
	}()
}

// SetPublishFailureListenre will set the listener to call in the case of
// a direct message publish failure.
func (publisher *directMessagePublisherImpl) SetPublishFailureListener(listener solace.PublishFailureListener) {
	publisher.publishFailureListener = listener
}

// IsReady checks if the publisher can publish messages. Returns true if the
// publisher can publish messages, false if the publisher is presvented from
// sending messages (e.g., full buffer or I/O problems)
func (publisher *directMessagePublisherImpl) IsReady() bool {
	return publisher.IsRunning() && (publisher.backpressureConfiguration != backpressureConfigurationReject || len(publisher.buffer) != cap(publisher.buffer))
}

// NotifyWhenReady makes a request to notify the application when the
// publisher is ready. This function will block until the publisher
// is ready.
func (publisher *directMessagePublisherImpl) NotifyWhenReady() {
	if publisher.IsReady() {
		publisher.notifyReady()
	}
}

// queues a new ready event on the event executor
func (publisher *directMessagePublisherImpl) notifyReady() {
	readinessListener := publisher.readinessListener
	if readinessListener != nil {
		publisher.eventExecutor.Submit(executor.Task(readinessListener))
	}
}

// drainQueue will drain the message buffer and return the number of undelivered messages. If given a publish failure listener, it will be
// called with every undelivered message and the error if present
func (publisher *directMessagePublisherImpl) drainQueue(shutdownTime time.Time, err error, listener solace.PublishFailureListener) uint64 {
	close(publisher.buffer)
	undeliveredCount := uint64(0)
	for undelivered := range publisher.buffer {
		underliveredRef := undelivered
		undeliveredCount++
		if listener != nil {
			event := &failedPublishEvent{
				dest:      underliveredRef.destination,
				message:   underliveredRef.message,
				timestamp: shutdownTime,
				err:       err,
			}
			if !publisher.eventExecutor.Submit(func() { listener(event) }) && publisher.logger.IsInfoEnabled() {
				publisher.logger.Info(fmt.Sprintf("Failed to submit failed publish event %v, is the publisher terminated?", event))
			}
		}
	}
	return undeliveredCount
}

// PublishBytes will publish a message of type byte array to the given destination.
// Returns an error if one occurred while attempting to publish or if the publisher
// is not started/terminated. Returns an error if one occurred. Possible errors include
// - solace/solace.*PubSubPlusClientError if the message could not be sent and all retry attempts failed.
// - solace/solace.*PublisherOverflowError if publishing messages faster than publisher's I/O
// capabilities allow. When publishing can be resumed, registered PublisherReadinessListeners
// will be called.
func (publisher *directMessagePublisherImpl) PublishBytes(bytes []byte, dest *resource.Topic) error {
	msg, err := publisher.messageBuilder.BuildWithByteArrayPayload(bytes)
	if err != nil {
		return err
	}
	// we built the message so it is safe to cast
	return publisher.publish(msg.(*message.OutboundMessageImpl), dest)
}

// PublishString will publish a message of type string to the given destination.
// Returns an error if one occurred. Possible errors include:
// - solace/solace.*PubSubPlusClientError if the message could not be sent and all retry attempts failed.
// - solace/solace.*PublisherOverflowError if publishing messages faster than publisher's I/O
// capabilities allow. When publishing can be resumed, registered PublisherReadinessListeners
// will be called.
func (publisher *directMessagePublisherImpl) PublishString(str string, dest *resource.Topic) error {
	msg, err := publisher.messageBuilder.BuildWithStringPayload(str)
	if err != nil {
		return err
	}
	// we built the message so it is safe to cast
	return publisher.publish(msg.(*message.OutboundMessageImpl), dest)
}

// Publish will publish the given message of type OutboundMessage built by a
// OutboundMessageBuilder to the given destination. Possible errors include:
// - solace/solace.*PubSubPlusClientError if the message could not be sent and all retry attempts failed.
// - solace/solace.*PublisherOverflowError if publishing messages faster than publisher's I/O
// capabilities allow. When publishing can be resumed, registered PublisherReadinessListeners
// will be called.
func (publisher *directMessagePublisherImpl) Publish(msg apimessage.OutboundMessage, dest *resource.Topic) error {
	return publisher.PublishWithProperties(msg, dest, nil)
}

// PublishWithProperties will publish the given message of type OutboundMessage
// with the given properties. These properties will override the properties on
// the OutboundMessage instance if present. Possible errors include:
// - solace/solace.*PubSubPlusClientError if the message could not be sent and all retry attempts failed.
// - solace/solace.*PublisherOverflowError if publishing messages faster than publisher's I/O
// capabilities allow. When publishing can be resumed, registered PublisherReadinessListeners
// will be called.
func (publisher *directMessagePublisherImpl) PublishWithProperties(msg apimessage.OutboundMessage, dest *resource.Topic, properties config.MessagePropertiesConfigurationProvider) error {
	if err := publisher.checkStartedStateForPublish(); err != nil {
		return err
	}
	msgImpl, ok := msg.(*message.OutboundMessageImpl)
	if !ok {
		return solace.NewError(&solace.IllegalArgumentError{}, fmt.Sprintf(constants.InvalidOutboundMessageType, msg), nil)
	}
	msgDup, err := message.DuplicateOutboundMessage(msgImpl)
	if err != nil {
		return err
	}
	if properties != nil {
		err := message.SetProperties(msgDup, properties.GetConfiguration())
		if err != nil {
			msgDup.Dispose()
			return err
		}
	}
	return publisher.publish(msgDup, dest)
}

// publish impl taking a dup'd message, assuming state has been checked and we are running
func (publisher *directMessagePublisherImpl) publish(msg *message.OutboundMessageImpl, dest *resource.Topic) (ret error) {
	// There is a potential race condition in this function in buffered scenarios whereby a message is pushed into backpressure
	// after the publisher has moved from Started to Terminated if the routine is interrupted after the state check and not resumed
	// until much much later. Therefore, it may be possible for a message to get into the publisher buffers but not actually
	// be put out to the wire as the publisher's task buffer may shut down immediately after. This would result in an unpublished
	// message that was submitted to publish successfully. In reality, this condition's window is so rediculously tiny that it
	// can be considered a non-problem. Also (at the time of writing) this race condition is present in all other next-gen APIs.

	// Set the destination for the message which is assumed to be a dup'd message.
	err := message.SetDestination(msg, dest.GetName())
	if err != nil {
		msg.Dispose()
		return err
	}

	// check the state once more before moving into the publish paths
	if err := publisher.checkStartedStateForPublish(); err != nil {
		return err
	}
	if publisher.backpressureConfiguration == backpressureConfigurationDirect {
		defer msg.Dispose()
		// publish directly with CCSMP
		errorInfo := publisher.internalPublisher.Publish(message.GetOutboundMessagePointer(msg))
		if errorInfo != nil {
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
		pub := &publishable{msg, dest}
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
		if !publisher.taskBuffer.Submit(publisher.sendTask(msg, dest)) {
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

// sendTask represents the task that is submitted to the internal task buffer and ultimately the shared serialized publisher instance
// returned closure accepts a channel that will receive a notification when any waits should be interrupted
func (publisher *directMessagePublisherImpl) sendTask(msg *message.OutboundMessageImpl, dest resource.Destination) buffer.PublisherTask {
	return func(terminateChannel chan struct{}) {
		var errorInfo core.ErrorInfo
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
				// Only if we were the ones to drain the message from the buffer should we call the publish failure listener
				if errorInfo != nil && publisher.publishFailureListener != nil {
					event := &failedPublishEvent{
						message:   msg,
						dest:      dest,
						timestamp: time.Now(),
						err:       core.ToNativeError(errorInfo, "encountered error while publishing message: "),
					}
					// if we call the publish failure listener, we should not dispose of the message
					if !publisher.eventExecutor.Submit(func() { publisher.publishFailureListener(event) }) &&
						publisher.logger.IsInfoEnabled() {
						publisher.logger.Info(fmt.Sprintf("Failed to submit publish failure event %v. Is the publisher terminated?", event))
					}
				} else {
					// clean up the message, we are finished with it in the direct messaging case
					// slightly more efficient to dispose of the message than let GC clean it up
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

func (publisher *directMessagePublisherImpl) String() string {
	return fmt.Sprintf("solace.DirectMessagePublisher at %p", publisher)
}

type directMessagePublisherBuilderImpl struct {
	internalPublisher core.Publisher
	properties        map[config.PublisherProperty]interface{}
}

// NewDirectMessagePublisherBuilderImpl function
func NewDirectMessagePublisherBuilderImpl(internalPublisher core.Publisher) solace.DirectMessagePublisherBuilder {
	return &directMessagePublisherBuilderImpl{
		internalPublisher: internalPublisher,
		// default properties
		properties: constants.DefaultDirectPublisherProperties.GetConfiguration(),
	}
}

// Build will build a new DirectMessagePublisher instance based on the configured properties.
// Returns solace/solace.*InvalidConfigurationError if an invalid configuration is provided.
func (builder *directMessagePublisherBuilderImpl) Build() (messagePublisher solace.DirectMessagePublisher, err error) {
	backpressureConfig, publisherBackpressureBufferSize, err := validateBackpressureConfig(builder.properties)
	if err != nil {
		return nil, err
	}
	publisher := &directMessagePublisherImpl{}
	publisher.construct(builder.internalPublisher, backpressureConfig, publisherBackpressureBufferSize)
	return publisher, nil
}

// OnBackPressureReject will set the publisher backpressure strategy to reject
// where publish attempts will be rejected once the bufferSize, in number of messages, is reached.
// If bufferSize is 0, an error will be thrown when the transport is full when publishing.
// Valid bufferSize is >= 0.
func (builder *directMessagePublisherBuilderImpl) OnBackPressureReject(bufferSize uint) solace.DirectMessagePublisherBuilder {
	builder.properties[config.PublisherPropertyBackPressureStrategy] = config.PublisherPropertyBackPressureStrategyBufferRejectWhenFull
	builder.properties[config.PublisherPropertyBackPressureBufferCapacity] = bufferSize
	return builder
}

// OnBackPressureWait will set the publisher backpressure strategy to wait where publish
// attempts will block until there is space in the buffer of size bufferSize in number of messages.
// Valid bufferSize is >= 1.
func (builder *directMessagePublisherBuilderImpl) OnBackPressureWait(bufferSize uint) solace.DirectMessagePublisherBuilder {
	builder.properties[config.PublisherPropertyBackPressureStrategy] = config.PublisherPropertyBackPressureStrategyBufferWaitWhenFull
	builder.properties[config.PublisherPropertyBackPressureBufferCapacity] = bufferSize
	return builder
}

// FromConfigurationProvider will configure the direct publisher with the given properties.
// Built in PublisherPropertiesConfigurationProvider implementations include:
//   PublisherPropertyMap, a map of PublisherProperty keys to values
//   for loading of properties from a string configuration (files or other configuration source)
func (builder *directMessagePublisherBuilderImpl) FromConfigurationProvider(provider config.PublisherPropertiesConfigurationProvider) solace.DirectMessagePublisherBuilder {
	if provider == nil {
		return builder
	}
	for key, value := range provider.GetConfiguration() {
		builder.properties[key] = value
	}
	return builder
}

func (builder *directMessagePublisherBuilderImpl) String() string {
	return fmt.Sprintf("solace.DirectMessagePublisherBuilder at %p", builder)
}
