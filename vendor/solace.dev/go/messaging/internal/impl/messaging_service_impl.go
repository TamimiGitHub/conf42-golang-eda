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

package impl

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"solace.dev/go/messaging/internal/impl/executor"
	"solace.dev/go/messaging/internal/impl/future"
	"solace.dev/go/messaging/internal/impl/receiver"

	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/internal/impl/logging"
	"solace.dev/go/messaging/internal/impl/message"
	"solace.dev/go/messaging/internal/impl/publisher"
	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/metrics"
)

// the main service lifecycle state
type messagingServiceState = int32

const (
	messagingServiceStateNotConnected  messagingServiceState = iota
	messagingServiceStateConnecting    messagingServiceState = iota
	messagingServiceStateConnected     messagingServiceState = iota
	messagingServiceStateDisconnecting messagingServiceState = iota
	messagingServiceStateDisconnected  messagingServiceState = iota
)

var messagingServiceStateNames = map[messagingServiceState]string{
	messagingServiceStateNotConnected:  "Not Connected",
	messagingServiceStateConnecting:    "Connecting",
	messagingServiceStateConnected:     "Connected",
	messagingServiceStateDisconnecting: "Disconnecting",
	messagingServiceStateDisconnected:  "Disconnected",
}

// the state
type messagingServiceSubState = int32

const (
	messagingServiceSubStateUp           messagingServiceSubState = iota
	messagingServiceSubStateReconnecting messagingServiceSubState = iota
	messagingServiceSubStateReconnected  messagingServiceSubState = iota
	messagingServiceSubStateDown         messagingServiceSubState = iota
)

// var mmessagingServiceSubStateNames = map[messagingServiceSubState]string{
// 	messagingServiceSubStateUp:           "Up",
// 	messagingServiceSubStateReconnecting: "Reconnecting",
// 	messagingServiceSubStateDown:         "Down",
// }

type messagingServiceImpl struct {
	transport        core.Transport
	logger           logging.LogLevelLogger
	state            messagingServiceState
	activeSubState   messagingServiceSubState
	connectFuture    future.FutureError
	disconnectFuture future.FutureError

	// the id used to register with core.Events for unsolicited terminations
	downEventHandlerRegisteredEventID uint
	// the id used to register with core.Events for reconnect events
	reconnectEventHandlerRegisteredEventID uint
	// the id used to register with core.Events for reconnect attempt events
	reconnectAttemptEventHandlerRegisteredEventID uint

	eventExecutor executor.Executor

	downEventHandlerID     uint64
	downEventHandlers      map[uint64]solace.ServiceInterruptionListener
	downEventHandlersMutex sync.Mutex

	reconnectEventHandlerID     uint64
	reconnectEventHandlers      map[uint64]solace.ReconnectionListener
	reconnectEventHandlersMutex sync.Mutex

	reconnectAttemptEventHandlerID     uint64
	reconnectAttemptEventHandlers      map[uint64]solace.ReconnectionAttemptListener
	reconnectAttemptEventHandlersMutex sync.Mutex
}

func newMessagingServiceImpl(logger logging.LogLevelLogger) *messagingServiceImpl {
	messagingService := &messagingServiceImpl{
		state:                         messagingServiceStateNotConnected,
		activeSubState:                messagingServiceSubStateUp,
		connectFuture:                 future.NewFutureError(),
		disconnectFuture:              future.NewFutureError(),
		eventExecutor:                 executor.NewExecutor(),
		downEventHandlers:             make(map[uint64]solace.ServiceInterruptionListener),
		reconnectEventHandlers:        make(map[uint64]solace.ReconnectionListener),
		reconnectAttemptEventHandlers: make(map[uint64]solace.ReconnectionAttemptListener),
	}
	messagingService.logger = logger.For(messagingService)
	return messagingService
}

// Connect connects the messaging service.
// Will block until the connection attempt is completed.
// Returns nil if successful or an error containing failure details.
// May return a solace/errors.*PubSubPlusClientError if a connection error occurs,
// or solace/errors.*IllegalStateError if the MessagingService is already connected,
// or has already been terminated.
func (service *messagingServiceImpl) Connect() (ret error) {
	proceed := atomic.CompareAndSwapInt32(&service.state, messagingServiceStateNotConnected, messagingServiceStateConnecting)
	if !proceed {
		currentState := service.getState()
		if currentState != messagingServiceStateDisconnecting && currentState != messagingServiceStateDisconnected {
			return service.connectFuture.Get()
		}
		return solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToConnectAlreadyDisconnectedService, messagingServiceStateNames[service.state]), nil)
	}
	if service.logger.IsDebugEnabled() {
		service.logger.Debug("Connect start")
		defer func() {
			if ret != nil {
				service.logger.Debug("Connect complete with error: " + ret.Error())
			} else {
				service.logger.Debug("Connect complete")
			}
		}()
	}
	// attempt to connect the transport synchronously
	err := service.transport.Connect()
	if err != nil {
		atomic.StoreInt32(&service.activeSubState, messagingServiceSubStateDown)
		atomic.StoreInt32(&service.state, messagingServiceStateConnected)
		service.connectFuture.Complete(err)
		return err
	}
	go service.eventExecutor.Run()
	service.downEventHandlerRegisteredEventID = service.transport.Events().AddEventHandler(core.SolClientEventDown, service.downEventHandler)
	service.reconnectEventHandlerRegisteredEventID = service.transport.Events().AddEventHandler(core.SolClientEventReconnect, service.reconnectEventHandler)
	service.reconnectAttemptEventHandlerRegisteredEventID = service.transport.Events().AddEventHandler(core.SolClientEventReconnectAttempt, service.reconnectAttempEventHandler)
	atomic.StoreInt32(&service.state, messagingServiceStateConnected)
	service.connectFuture.Complete(nil)
	return nil
}

func (service *messagingServiceImpl) downEventHandler(sessionEventInfo core.SessionEventInfo) {
	// atomically swap to disconnected and proceed with disconnect if successful, otherwise return immediately
	proceed := atomic.CompareAndSwapInt32(&service.state, messagingServiceStateConnected, messagingServiceStateDisconnected)
	if !proceed {
		return
	}
	atomic.StoreInt32(&service.activeSubState, messagingServiceSubStateDown)
	// Stop processing any events, don't await termination
	service.eventExecutor.Terminate()
	// we must offload removal of the listeners to another goroutine to run at some point in the future.
	// this is really just needed for garbage collection such that we don't have cyclic references.
	go func() {
		service.unregisterEventHandlers()
		// offload closing of the transport to another thread in order to shutdown
		service.transport.Close()
		// Only complete the future at the end so we can block on Disconnect
		service.disconnectFuture.Complete(nil)
	}()
	// Offload synchronous looping over the interruption listeners
	go service.notifyInterruptionListeners(sessionEventInfo)
}

// Loop over the service interruption listeners synchronously
func (service *messagingServiceImpl) notifyInterruptionListeners(sessionEventInfo core.SessionEventInfo) {
	service.downEventHandlersMutex.Lock()
	serviceEvent := &serviceEventInfo{
		timestamp: time.Now(),
		brokerURI: service.transport.Host(),
		message:   sessionEventInfo.GetInfoString(),
		cause:     sessionEventInfo.GetError(),
	}
	for _, eventHandler := range service.downEventHandlers {
		eventHandler(serviceEvent)
	}
	service.downEventHandlersMutex.Unlock()
	service.cleanupListeners()
}

func (service *messagingServiceImpl) reconnectEventHandler(sessionEventInfo core.SessionEventInfo) {
	service.reconnectEventHandlersMutex.Lock()
	defer service.reconnectEventHandlersMutex.Unlock()
	serviceEvent := &serviceEventInfo{
		timestamp: time.Now(),
		brokerURI: service.transport.Host(),
		message:   sessionEventInfo.GetInfoString(),
		cause:     sessionEventInfo.GetError(),
	}
	for _, eventHandler := range service.reconnectEventHandlers {
		eventHandlerRef := eventHandler
		if !service.eventExecutor.Submit(func() {
			eventHandlerRef(serviceEvent)
		}) {
			service.logger.Info("Failed to submit reconnection event due to service termination")
		}
	}
}

func (service *messagingServiceImpl) reconnectAttempEventHandler(sessionEventInfo core.SessionEventInfo) {
	service.reconnectAttemptEventHandlersMutex.Lock()
	defer service.reconnectAttemptEventHandlersMutex.Unlock()
	serviceEvent := &serviceEventInfo{
		timestamp: time.Now(),
		brokerURI: service.transport.Host(),
		message:   sessionEventInfo.GetInfoString(),
		cause:     sessionEventInfo.GetError(),
	}
	for _, eventHandler := range service.reconnectAttemptEventHandlers {
		eventHandlerRef := eventHandler
		if !service.eventExecutor.Submit(func() {
			eventHandlerRef(serviceEvent)
		}) {
			service.logger.Info("Failed to submit reconnection attempt event due to service termination")
		}
	}
}

// ConnectAsync connects the messaging service asynchronously.
// Returns a channel that will receive an event when completed.
// Channel will receive nil if successful or an error containing failure details.
// For more information, see MessagingService.Connect.
func (service *messagingServiceImpl) ConnectAsync() <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- service.Connect()
		close(result)
	}()
	return result
}

// ConnectAsyncWithCallback connects the messaging service asynchonously.
// When complete, the given callback will be called with nil if successful
// or an error if not successful. In both cases, the messaging service
// will be passed as well.
func (service *messagingServiceImpl) ConnectAsyncWithCallback(callback func(solace.MessagingService, error)) {
	go func() {
		callback(service, service.Connect())
	}()
}

// CreateDirectMessagePublisherBuilder creates a new direct message publisher builder
// that can be used to configure direct message publisher instances.
func (service *messagingServiceImpl) CreateDirectMessagePublisherBuilder() solace.DirectMessagePublisherBuilder {
	return publisher.NewDirectMessagePublisherBuilderImpl(service.transport.Publisher())
}

// CreateDirectMessageReceiverBuilder creates a new direct message receiver builder
// that can be used to configure direct message receiver instances.
func (service *messagingServiceImpl) CreateDirectMessageReceiverBuilder() solace.DirectMessageReceiverBuilder {
	return receiver.NewDirectMessageReceiverBuilderImpl(service.transport.Receiver())
}

// CreatePersistentMessagePublisherBuilder creates a new persistent message publisher builder
// that can be used to configure persistent message publisher instances.
func (service *messagingServiceImpl) CreatePersistentMessagePublisherBuilder() solace.PersistentMessagePublisherBuilder {
	return publisher.NewPersistentMessagePublisherBuilderImpl(service.transport.Publisher())
}

// CreatePersistentMessageReceiverBuilder creates a new persistent message receiver builder
// that can be used to configure persistent message receiver instances.
func (service *messagingServiceImpl) CreatePersistentMessageReceiverBuilder() solace.PersistentMessageReceiverBuilder {
	return receiver.NewPersistentMessageReceiverBuilderImpl(service.transport.Receiver())
}

// MessageBuilder creates a new outbound message builder that can be
// used to build messages to send via a message publisher.
// Should this just be called MessageBuilder?
func (service *messagingServiceImpl) MessageBuilder() solace.OutboundMessageBuilder {
	return message.NewOutboundMessageBuilder()
}

// Disconnect disconnect the messaging service.
// The messaging service must be connected to disconnect.
// Will block until the disconnection attempt is completed.
// Returns nil if successful or an error containing failure details.
// A disconnected messaging service may not be reconnected.
// Returns solace/errors.*IllegalStateError if not connected or already disconnected.
func (service *messagingServiceImpl) Disconnect() (ret error) {
	proceed := atomic.CompareAndSwapInt32(&service.state, messagingServiceStateConnected, messagingServiceStateDisconnecting)
	if !proceed {
		currentState := service.getState()
		if currentState != messagingServiceStateNotConnected && currentState != messagingServiceStateConnecting {
			return service.disconnectFuture.Get()
		}
		return solace.NewError(&solace.IllegalStateError{}, fmt.Sprintf(constants.UnableToDisconnectUnstartedService, messagingServiceStateNames[service.state]), nil)
	}
	if service.logger.IsDebugEnabled() {
		service.logger.Debug("Disconnect start")
		defer func() {
			if ret != nil {
				service.logger.Debug("Disconnect complete with error: " + ret.Error())
			} else {
				service.logger.Debug("Disconnect complete")
			}
		}()
	}
	// make sure we remove the event handler id so we can be GC'd
	service.unregisterEventHandlers()
	service.cleanupListeners()
	err := service.transport.Disconnect()
	service.transport.Close()
	service.eventExecutor.Terminate()
	atomic.StoreInt32(&service.state, messagingServiceStateDisconnected)
	service.disconnectFuture.Complete(err)
	return err
}

func (service *messagingServiceImpl) unregisterEventHandlers() {
	// we also want to remove the event handler before calling disconnect so we don't try and call disconnect again
	service.transport.Events().RemoveEventHandler(service.downEventHandlerRegisteredEventID)
	service.transport.Events().RemoveEventHandler(service.reconnectEventHandlerRegisteredEventID)
	service.transport.Events().RemoveEventHandler(service.reconnectAttemptEventHandlerRegisteredEventID)
}

func (service *messagingServiceImpl) cleanupListeners() {
	// We also want to remove all listeners so we don't keep references around
	service.downEventHandlersMutex.Lock()
	for k := range service.downEventHandlers {
		delete(service.downEventHandlers, k)
	}
	service.downEventHandlersMutex.Unlock()
	// Clear the reconnect events
	service.reconnectEventHandlersMutex.Lock()
	for k := range service.reconnectEventHandlers {
		delete(service.reconnectEventHandlers, k)
	}
	service.reconnectEventHandlersMutex.Unlock()
	// Clear the reconnect attempt events
	service.reconnectAttemptEventHandlersMutex.Lock()
	for k := range service.reconnectAttemptEventHandlers {
		delete(service.reconnectAttemptEventHandlers, k)
	}
	service.reconnectAttemptEventHandlersMutex.Unlock()
}

// DisconnectAsync disconnects the messaging service asynchronously.
// Returns a channel that will receive an event when completed
// Channel will receive nil if successful or an error containing failure details
// For more information, see MessagingService.Disconnect.
func (service *messagingServiceImpl) DisconnectAsync() <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- service.Disconnect()
		close(result)
	}()
	return result
}

// DisconnectAsyncWithCallback disconnects the messaging service asynchronously.
// When complete, the given callback will be called with nil if successful
// or an error if not successful.
func (service *messagingServiceImpl) DisconnectAsyncWithCallback(callback func(error)) {
	go func() {
		callback(service.Disconnect())
	}()
}

// IsConnected determines if service is operational and Connect() was previously
// called successfully.
// Returns true if service is connected to a remote destination, false otherwise
func (service *messagingServiceImpl) IsConnected() bool {
	return service.getState() == messagingServiceStateConnected && service.getSubState() == messagingServiceSubStateUp
}

// AddReconnectionListener adds a new reconnection listener to the messaging service.
// The reconnection listener will be called when reconnection events occur.
func (service *messagingServiceImpl) AddReconnectionListener(listener solace.ReconnectionListener) uint64 {
	id := atomic.AddUint64(&service.reconnectEventHandlerID, 1)
	service.reconnectEventHandlersMutex.Lock()
	defer service.reconnectEventHandlersMutex.Unlock()
	service.reconnectEventHandlers[id] = listener
	return id
}

// RemoveReconnectionListener removes the given listener from the messaging service.
func (service *messagingServiceImpl) RemoveReconnectionListener(listener uint64) {
	service.reconnectEventHandlersMutex.Lock()
	defer service.reconnectEventHandlersMutex.Unlock()
	delete(service.reconnectEventHandlers, listener)
}

// AddReconnectionAttemptListener adds a listener to receive reconnection attempt notification.
// The reconnection listener will be called when a connection is lost and reconnection attempts have begun.
func (service *messagingServiceImpl) AddReconnectionAttemptListener(listener solace.ReconnectionAttemptListener) uint64 {
	id := atomic.AddUint64(&service.reconnectAttemptEventHandlerID, 1)
	service.reconnectAttemptEventHandlersMutex.Lock()
	defer service.reconnectAttemptEventHandlersMutex.Unlock()
	service.reconnectAttemptEventHandlers[id] = listener
	return id
}

// RemoveReconnectionAttemptListener removes the given listener from the messaging service.
func (service *messagingServiceImpl) RemoveReconnectionAttemptListener(listener uint64) {
	service.reconnectAttemptEventHandlersMutex.Lock()
	defer service.reconnectAttemptEventHandlersMutex.Unlock()
	delete(service.reconnectAttemptEventHandlers, listener)
}

// AddServiceInterruptionListener adds a listener to receive non recoverable service interruption events.
func (service *messagingServiceImpl) AddServiceInterruptionListener(listener solace.ServiceInterruptionListener) uint64 {
	id := atomic.AddUint64(&service.downEventHandlerID, 1)
	service.downEventHandlersMutex.Lock()
	defer service.downEventHandlersMutex.Unlock()
	service.downEventHandlers[id] = listener
	return id
}

// RemoveServiceInterruptionListener removes a service listener to receive non
// recoverable service interruption events.
func (service *messagingServiceImpl) RemoveServiceInterruptionListener(listener uint64) {
	service.downEventHandlersMutex.Lock()
	defer service.downEventHandlersMutex.Unlock()
	delete(service.downEventHandlers, listener)
}

// GetApplicationID gets the application identifier.
func (service *messagingServiceImpl) GetApplicationID() string {
	return service.transport.ID()
}

// Metrics will return the metrics for this MessagingService instance.
func (service *messagingServiceImpl) Metrics() metrics.APIMetrics {
	return &metricsImpl{metricsHandle: service.transport.Metrics()}
}

// Info will return the API Info for this MessagingService instance.
func (service *messagingServiceImpl) Info() metrics.APIInfo {
	version, buildDate, variant := core.GetVersion()
	return &apiInfo{
		version:   version,
		buildDate: buildDate,
		vendor:    variant,
		userID:    service.transport.ID(),
	}
}

func (service *messagingServiceImpl) String() string {
	return fmt.Sprintf("solace.MessagingService at %p", service)
}

func (service *messagingServiceImpl) getState() messagingServiceState {
	return atomic.LoadInt32(&service.state)
}

func (service *messagingServiceImpl) getSubState() messagingServiceSubState {
	return atomic.LoadInt32(&service.activeSubState)
}

type apiInfo struct {
	buildDate, version, vendor, userID string
}

// GetAPIBuildDate will return the build date of the Solace PubSub+ Golang API in use.
func (info *apiInfo) GetAPIBuildDate() string {
	return info.buildDate
}

// GetAPIVersion will return the version of the Solace PubSub+ Golang API in use.
func (info *apiInfo) GetAPIVersion() string {
	return info.version
}

// GetAPIUserID will return the user ID transmitted to the event broker.
func (info *apiInfo) GetAPIUserID() string {
	return info.userID
}

// GetAPIImplementationVendor will get the API implementation vendor transmitted to the event broker.
func (info *apiInfo) GetAPIImplementationVendor() string {
	return info.vendor
}

type serviceEventInfo struct {
	timestamp time.Time
	brokerURI string
	message   string
	cause     error
}

// GetTimestamp retrieves the timestamp of the event
func (event *serviceEventInfo) GetTimestamp() time.Time {
	return event.timestamp
}

// GetBrokerURI retrieves the URI of the event broker
func (event *serviceEventInfo) GetBrokerURI() string {
	return event.brokerURI
}

// GetMessage retrieves the message contents
func (event *serviceEventInfo) GetMessage() string {
	return event.message
}

// GetCause retrieves the cause of the client error
func (event *serviceEventInfo) GetCause() error {
	return event.cause
}
