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
	"sync/atomic"
	"time"

	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/internal/impl/executor"
	"solace.dev/go/messaging/internal/impl/message"
	"solace.dev/go/messaging/internal/impl/validation"

	"solace.dev/go/messaging/internal/impl/future"

	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/resource"
)

// alias int32 for both publisher state and sub state
type messagePublisherState = int32

const (
	messagePublisherStateNotStarted  messagePublisherState = iota
	messagePublisherStateStarting    messagePublisherState = iota
	messagePublisherStateStarted     messagePublisherState = iota
	messagePublisherStateTerminating messagePublisherState = iota
	messagePublisherStateTerminated  messagePublisherState = iota
)

var messagePublisherStateNames = map[messagePublisherState]string{
	messagePublisherStateNotStarted:  "NotStarted",
	messagePublisherStateStarting:    "Starting",
	messagePublisherStateStarted:     "Started",
	messagePublisherStateTerminating: "Terminating",
	messagePublisherStateTerminated:  "Terminated",
}

type backpressureConfiguration byte

const (
	backpressureConfigurationDirect backpressureConfiguration = iota
	backpressureConfigurationReject backpressureConfiguration = iota
	backpressureConfigurationWait   backpressureConfiguration = iota
)

type publishable struct {
	message     *message.OutboundMessageImpl
	destination resource.Destination
}

type basicMessagePublisher struct {
	internalPublisher core.Publisher
	state             messagePublisherState

	readinessListener   solace.PublisherReadinessListener
	terminationListener solace.TerminationNotificationListener

	startFuture, terminateFuture future.FutureError

	messageBuilder solace.OutboundMessageBuilder
	eventExecutor  executor.Executor
}

func (publisher *basicMessagePublisher) construct(internalPublisher core.Publisher) {
	publisher.internalPublisher = internalPublisher
	publisher.state = messagePublisherStateNotStarted

	publisher.startFuture = future.NewFutureError()
	publisher.terminateFuture = future.NewFutureError()

	publisher.messageBuilder = message.NewOutboundMessageBuilder()
	publisher.eventExecutor = executor.NewExecutor()
}

// returns nil if successfully moved to starting state, false otherwise
func (publisher *basicMessagePublisher) starting() (proceed bool, errOrNil error) {
	currentState := publisher.getState()
	// check the current state first, we many block on startup
	if currentState != messagePublisherStateNotStarted {
		return false, publisher.checkStartupStateAndWait(currentState)
	}
	// so far so good, looks like we can proceed with startup
	if !publisher.internalPublisher.IsRunning() {
		// we error if the publisher is not running
		return false, solace.NewError(&solace.IllegalStateError{}, constants.UnableToStartPublisherParentServiceNotStarted, nil)
	}
	// atomically move to starting, and check response to see if we should proceed, if we can we are good to go with start
	success := atomic.CompareAndSwapInt32(&publisher.state, messagePublisherStateNotStarted, messagePublisherStateStarting)
	if !success {
		// looks like we failed to swap to the new state
		currentState = publisher.getState()
		return false, publisher.checkStartupStateAndWait(currentState)
	}
	// we can continue with execution of start
	return true, nil
}

// checks the state of startup and waits on the future if we are starting/started
func (publisher *basicMessagePublisher) checkStartupStateAndWait(currentState messagePublisherState) error {
	if currentState != messagePublisherStateTerminating && currentState != messagePublisherStateTerminated {
		// wait for completion of startup, but don't continue with start
		return publisher.startFuture.Get()
	}
	return solace.NewError(&solace.IllegalStateError{}, constants.UnableToStartPublisher, nil)
}

// sets the state to started
func (publisher *basicMessagePublisher) started(err error) {
	atomic.StoreInt32(&publisher.state, messagePublisherStateStarted)
	// success
	publisher.startFuture.Complete(err)
}

func (publisher *basicMessagePublisher) terminate() (proceed bool, err error) {
	success := atomic.CompareAndSwapInt32(&publisher.state, messagePublisherStateStarted, messagePublisherStateTerminating)
	if !success {
		// looks like we failed to swap to the new state, someone beat us to terminate
		currentState := publisher.getState()
		if currentState == messagePublisherStateNotStarted && atomic.CompareAndSwapInt32(&publisher.state,
			messagePublisherStateNotStarted, messagePublisherStateTerminated) {
			// don't proceed, just terminate immediately
			publisher.terminateFuture.Complete(nil)
			return false, nil
		}
		return false, publisher.checkTerminateStateAndWait(currentState)
	}
	// we can proceed with termination
	return true, nil
}

func (publisher *basicMessagePublisher) checkTerminateStateAndWait(currentState messagePublisherState) error {
	if currentState == messagePublisherStateTerminating || currentState == messagePublisherStateTerminated {
		return publisher.terminateFuture.Get()
	}
	return solace.NewError(&solace.IllegalStateError{}, constants.UnableToTerminatePublisher, nil)
}

func (publisher *basicMessagePublisher) terminated(err error) {
	atomic.StoreInt32(&publisher.state, messagePublisherStateTerminated)
	// success
	publisher.terminateFuture.Complete(err)
}

func (publisher *basicMessagePublisher) getState() messagePublisherState {
	return atomic.LoadInt32(&publisher.state)
}

func (publisher *basicMessagePublisher) checkStartedStateForPublish() error {
	if currentState := publisher.getState(); currentState != messagePublisherStateStarted {
		if currentState == messagePublisherStateTerminating || currentState == messagePublisherStateTerminated {
			return solace.NewError(&solace.IllegalStateError{}, constants.UnableToPublishAlreadyTerminated, nil)
		}
		return solace.NewError(&solace.IllegalStateError{}, constants.UnableToPublishNotStarted+messagePublisherStateNames[currentState], nil)
	}
	return nil
}

// Implementation of common functionality

// IsRunning checks if the process was successfully started and not yet stopped.
// Returns true if running, false otherwise.
func (publisher *basicMessagePublisher) IsRunning() bool {
	return atomic.LoadInt32(&publisher.state) == messagePublisherStateStarted
}

// IsTerminates checks if message delivery process is terminated.
// Returns true if terminated, false otherwise.
func (publisher *basicMessagePublisher) IsTerminated() bool {
	return atomic.LoadInt32(&publisher.state) == messagePublisherStateTerminated
}

// IsTerminating checks if the delivery process termination is ongoing.
// Returns true if the message delivery process is being terminated,
// but termination is not yet complete, otherwise false.
func (publisher *basicMessagePublisher) IsTerminating() bool {
	return atomic.LoadInt32(&publisher.state) == messagePublisherStateTerminating
}

// SetTerminationNotificationListener adds a callback to listen for
// non-recoverable interruption events.
func (publisher *basicMessagePublisher) SetTerminationNotificationListener(listener solace.TerminationNotificationListener) {
	publisher.terminationListener = listener
}

// SetPublisherReadinessListener registers a listener to be called when the
// publisher can send messages. Typically, the listener is notified after a
// Publisher instance raises an error indicating that the outbound message
// buffer is full.
func (publisher *basicMessagePublisher) SetPublisherReadinessListener(listener solace.PublisherReadinessListener) {
	publisher.readinessListener = listener
}

type publisherTerminationEvent struct {
	eventTime time.Time
	cause     error
}

// GetTimestamp retrieves the timestamp of the event.
func (event *publisherTerminationEvent) GetTimestamp() time.Time {
	return event.eventTime
}

// GetMessage retrieves the event message.
func (event *publisherTerminationEvent) GetMessage() string {
	return fmt.Sprintf("Publisher Termination Event - timestamp: %s, cause: %s", event.eventTime, event.cause)
}

// GetCause retrieves the cause of the client exception if any.
// Returns the error event or nil if no cause is present.
func (event *publisherTerminationEvent) GetCause() error {
	return event.cause
}

// Common functionality to validate backpressure config, can be used by direct and persistent publishers
func validateBackpressureConfig(properties config.PublisherPropertyMap) (backpressureConfig backpressureConfiguration, publisherBackpressureBufferSize int, err error) {
	var publisherBackpressureStrategy string
	// fetch all required properties as their desired types
	if publisherBackpressureStrategy, _, err = validation.StringPropertyValidation(
		string(config.PublisherPropertyBackPressureStrategy),
		properties[config.PublisherPropertyBackPressureStrategy],
		config.PublisherPropertyBackPressureStrategyBufferRejectWhenFull,
		config.PublisherPropertyBackPressureStrategyBufferWaitWhenFull,
	); err != nil {
		return
	}
	if publisherBackpressureBufferSize, _, err = validation.IntegerPropertyValidation(
		string(config.PublisherPropertyBackPressureBufferCapacity),
		properties[config.PublisherPropertyBackPressureBufferCapacity],
	); err != nil {
		return
	}
	// validate backpressure configuration
	if publisherBackpressureStrategy == config.PublisherPropertyBackPressureStrategyBufferRejectWhenFull {
		if publisherBackpressureBufferSize < 0 {
			err = solace.NewError(&solace.InvalidConfigurationError{}, fmt.Sprintf("buffer size must be >= 0 with backpressure strategy %s, got %d", publisherBackpressureStrategy, publisherBackpressureBufferSize), nil)
			return
		}
		if publisherBackpressureBufferSize == 0 {
			backpressureConfig = backpressureConfigurationDirect
		} else {
			backpressureConfig = backpressureConfigurationReject
		}
	} else if publisherBackpressureStrategy == config.PublisherPropertyBackPressureStrategyBufferWaitWhenFull {
		if publisherBackpressureBufferSize < 1 {
			err = solace.NewError(&solace.InvalidConfigurationError{}, fmt.Sprintf("buffer size must be > 0 with backpressure strategy %s, got %d", publisherBackpressureStrategy, publisherBackpressureBufferSize), nil)
			return
		}
		backpressureConfig = backpressureConfigurationWait
	}
	return
}
