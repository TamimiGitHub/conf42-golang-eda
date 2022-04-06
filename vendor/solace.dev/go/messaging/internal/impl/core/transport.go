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

// Package core abstracts out the inner workings of CCSMP and provides a few key entities used by various other
// services. For example, it contains SolClientTransport which wraps the session and context, as well as SolClientPublisher
// which wraps the session and provides hooks for callbacks.
// SolClient structs are not responsible for state tracking.
package core

import (
	"fmt"
	"runtime"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/internal/impl/logging"
	"solace.dev/go/messaging/pkg/solace"
)

// Transport interface
type Transport interface {
	Connect() error
	Disconnect() error
	Close() error
	Publisher() Publisher
	Receiver() Receiver
	Metrics() Metrics
	Events() Events
	ID() string
	Host() string
}

// NewTransport function
func NewTransport(host string, properties []string) (Transport, error) {
	return newCcsmpTransport(host, properties)
}

// Implementation
func newCcsmpTransport(host string, properties []string) (*ccsmpTransport, error) {
	ccsmpTransport := &ccsmpTransport{}
	// We want to be able to clean up if we have an unstarted service.
	// This finalizer will be removed on a call to Connect, and the context+session will
	// be destroyed on a call to Disconnect
	context, err := ccsmp.SolClientContextCreate()
	if err != nil {
		return nil, ToNativeError(err)
	}
	runtime.SetFinalizer(context, destroyContext)
	ccsmpTransport.context = context
	session, err := context.SolClientSessionCreate(properties)
	if err != nil {
		ret := ToNativeError(err)
		return nil, ret
	}
	runtime.SetFinalizer(session, destroySession)
	id, err := session.SolClientSessionGetClientName()
	if err != nil {
		return nil, ToNativeError(err)
	}
	ccsmpTransport.id = id
	ccsmpTransport.host = host
	ccsmpTransport.session = session
	ccsmpTransport.events = newCcsmpEvents(ccsmpTransport.session)
	ccsmpTransport.metrics = newCcsmpMetrics(ccsmpTransport.session)
	ccsmpTransport.publisher = newCcsmpPublisher(ccsmpTransport.session, ccsmpTransport.events, ccsmpTransport.metrics)
	ccsmpTransport.receiver = newCcsmpReceiver(ccsmpTransport.session, ccsmpTransport.events, ccsmpTransport.metrics)
	return ccsmpTransport, nil
}

// ccsmpTransport implements SolClientTransport and is responsible for handling session/context lifecycle
// as well as handling lifecycle of various sub components. Allows exposing of these subcomponents to
// users of the transport.
type ccsmpTransport struct {
	context *ccsmp.SolClientContext
	session *ccsmp.SolClientSession

	publisher *ccsmpBackedPublisher
	receiver  *ccsmpBackedReceiver
	metrics   *ccsmpBackedMetrics

	events *ccsmpBackedEvents

	host, id string
}

func (transport *ccsmpTransport) Connect() error {
	runtime.SetFinalizer(transport.context, nil)
	runtime.SetFinalizer(transport.session, nil)
	err := transport.session.SolClientSessionConnect()
	if err != nil {
		return ToNativeError(err, "an error occurred while connecting: ")
	}
	// start components
	transport.events.start()
	transport.publisher.start()
	transport.receiver.start()
	return nil
}

func (transport *ccsmpTransport) Disconnect() error {
	// first disconnect
	err := transport.session.SolClientSessionDisconnect()
	if err != nil {
		return ToNativeError(err, "an error occurred while disconnecting: ")
	}
	// signal all listeners that we are down
	transport.events.emitEvent(SolClientEventDown, &sessionEventInfo{
		err:          solace.NewError(&solace.ServiceUnreachableError{}, constants.TerminatedOnMessagingServiceShutdown, nil),
		infoString:   constants.TerminatedOnMessagingServiceShutdown,
		correlationP: nil,
		userP:        nil,
	})
	return nil
}

func (transport *ccsmpTransport) Close() error {
	// notify publisher and receiver that they are down
	transport.metrics.terminate()
	transport.events.terminate()
	transport.publisher.terminate()
	transport.receiver.terminate()
	// then clean up the session and context with calls to destroy.
	// this will invalidate metrics, but we can copy them out into golang memory if needed in the future.
	destroySession(transport.session)
	destroyContext(transport.context)
	return nil
}

func destroySession(session *ccsmp.SolClientSession) {
	err := session.SolClientSessionDestroy()
	if err != nil {
		logging.Default.Error(fmt.Sprintf("an error occurred while cleaning up session: %s (subcode %d)", err.GetMessageAsString(), err.SubCode))
	}
}

func destroyContext(context *ccsmp.SolClientContext) {
	err := context.SolClientContextDestroy()
	if err != nil {
		logging.Default.Error(fmt.Sprintf("an error occurred while cleaning up context: %s (subcode %d)", err.GetMessageAsString(), err.SubCode))
	}
}

func (transport *ccsmpTransport) Publisher() Publisher {
	return transport.publisher
}

func (transport *ccsmpTransport) Receiver() Receiver {
	return transport.receiver
}

func (transport *ccsmpTransport) Metrics() Metrics {
	return transport.metrics
}

func (transport *ccsmpTransport) Events() Events {
	return transport.events
}

func (transport *ccsmpTransport) ID() string {
	return transport.id
}

func (transport *ccsmpTransport) Host() string {
	return transport.host
}
