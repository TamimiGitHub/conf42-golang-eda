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
	"sync/atomic"

	"solace.dev/go/messaging/internal/impl/constants"

	"solace.dev/go/messaging/internal/impl/future"

	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/pkg/solace"
)

// alias int32 for both receiver state and sub state
type messageReceiverState = int32

const (
	messageReceiverStateNotStarted  messageReceiverState = iota
	messageReceiverStateStarting    messageReceiverState = iota
	messageReceiverStateStarted     messageReceiverState = iota
	messageReceiverStateTerminating messageReceiverState = iota
	messageReceiverStateTerminated  messageReceiverState = iota
)

var messageReceiverStateNames = map[messageReceiverState]string{
	messageReceiverStateNotStarted:  "NotStarted",
	messageReceiverStateStarting:    "Starting",
	messageReceiverStateStarted:     "Started",
	messageReceiverStateTerminating: "Terminating",
	messageReceiverStateTerminated:  "Terminated",
}

type basicMessageReceiver struct {
	internalReceiver core.Receiver
	state            messageReceiverState

	terminationListener solace.TerminationNotificationListener

	startFuture, terminateFuture future.FutureError
}

func (receiver *basicMessageReceiver) construct(internalReceiver core.Receiver) {
	receiver.internalReceiver = internalReceiver
	receiver.state = messageReceiverStateNotStarted

	receiver.startFuture = future.NewFutureError()
	receiver.terminateFuture = future.NewFutureError()
}

// returns nil if successfully moved to starting state, false otherwise
func (receiver *basicMessageReceiver) starting() (proceed bool, errOrNil error) {
	currentState := receiver.getState()
	// check the current state first, we many block on startup
	if currentState != messageReceiverStateNotStarted {
		return false, receiver.checkStartupStateAndWait(currentState)
	}
	// so far so good, looks like we can proceed with startup
	if !receiver.internalReceiver.IsRunning() {
		// we error if the receiver is not running
		return false, solace.NewError(&solace.IllegalStateError{}, constants.UnableToStartReceiverParentServiceNotStarted, nil)
	}
	// atomically move to starting, and check response to see if we should proceed, if we can we are good to go with start
	success := atomic.CompareAndSwapInt32(&receiver.state, messageReceiverStateNotStarted, messageReceiverStateStarting)
	if !success {
		// looks like we failed to swap to the new state
		currentState = receiver.getState()
		return false, receiver.checkStartupStateAndWait(currentState)
	}
	// we can continue with execution of start
	return true, nil
}

// checks the state of startup and waits on the future if we are starting/started
func (receiver *basicMessageReceiver) checkStartupStateAndWait(currentState messageReceiverState) error {
	if currentState != messageReceiverStateTerminating && currentState != messageReceiverStateTerminated {
		// wait for completion of startup, but don't continue with start
		return receiver.startFuture.Get()
	}
	return solace.NewError(&solace.IllegalStateError{}, constants.UnableToStartReceiver, nil)
}

// sets the state to started
func (receiver *basicMessageReceiver) started(err error) {
	atomic.StoreInt32(&receiver.state, messageReceiverStateStarted)
	// success
	receiver.startFuture.Complete(err)
}

func (receiver *basicMessageReceiver) terminate() (proceed bool, err error) {
	success := atomic.CompareAndSwapInt32(&receiver.state, messageReceiverStateStarted, messageReceiverStateTerminating)
	if !success {
		// looks like we failed to swap to the new state, someone beat us to terminate
		currentState := receiver.getState()
		if currentState == messageReceiverStateNotStarted && atomic.CompareAndSwapInt32(&receiver.state,
			messageReceiverStateNotStarted, messageReceiverStateTerminated) {
			// don't proceed, just terminate immediately
			receiver.terminateFuture.Complete(nil)
			return false, nil
		}
		return false, receiver.checkTerminateStateAndWait(currentState)
	}
	// we can proceed with termination
	return true, nil
}

func (receiver *basicMessageReceiver) checkTerminateStateAndWait(currentState messageReceiverState) error {
	if currentState == messageReceiverStateTerminating || currentState == messageReceiverStateTerminated {
		return receiver.terminateFuture.Get()
	}
	return solace.NewError(&solace.IllegalStateError{}, constants.UnableToTerminateReceiver, nil)
}

func (receiver *basicMessageReceiver) terminated(err error) {
	atomic.StoreInt32(&receiver.state, messageReceiverStateTerminated)
	// success
	receiver.terminateFuture.Complete(err)
}

func (receiver *basicMessageReceiver) getState() messageReceiverState {
	return atomic.LoadInt32(&receiver.state)
}

// Implementation of common functionality

// IsRunning checks if the process was successfully started and not yet stopped.
// Returns true if running, false otherwise.
func (receiver *basicMessageReceiver) IsRunning() bool {
	return atomic.LoadInt32(&receiver.state) == messageReceiverStateStarted
}

// IsTerminates checks if message delivery process is terminated.
// Returns true if terminated, false otherwise.
func (receiver *basicMessageReceiver) IsTerminated() bool {
	return atomic.LoadInt32(&receiver.state) == messageReceiverStateTerminated
}

// IsTerminating checks if the delivery process termination is ongoing.
// Returns true if the message delivery process is being terminated,
// but termination is not yet complete, otherwise false.
func (receiver *basicMessageReceiver) IsTerminating() bool {
	return atomic.LoadInt32(&receiver.state) == messageReceiverStateTerminating
}

// SetTerminationNotificationListener adds a callback to listen for
// non-recoverable interruption events.
func (receiver *basicMessageReceiver) SetTerminationNotificationListener(listener solace.TerminationNotificationListener) {
	receiver.terminationListener = listener
}
