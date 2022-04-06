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
	"unsafe"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/logging"
)

// EventHandler type defined
type EventHandler func(SessionEventInfo)

// Events interface defined
type Events interface {
	AddEventHandler(sessionEvent Event, responseCode EventHandler) uint
	RemoveEventHandler(id uint)
}

// EventInfo interface defined
type EventInfo interface {
	GetError() error
	GetInfoString() string
}

// SessionEventInfo interface defined
type SessionEventInfo interface {
	EventInfo
	GetCorrelationPointer() unsafe.Pointer
}

// Event type defined
type Event int

const (

	// SolClientEventUp represents ccsmp.SolClientSessionEventUpNotice in sessionEventMapping
	SolClientEventUp Event = iota

	// SolClientEventDown represents SolClientSessionEventDownError in sessionEventMapping
	SolClientEventDown

	// SolClientEventCanSend represents ccsmp.SolClientSessionEventCanSend in sessionEventMapping
	SolClientEventCanSend

	// SolClientEventAcknowledgement represents ccsmp.SolClientSessionEventAcknowledgement in sessionEventMapping
	SolClientEventAcknowledgement

	// SolClientEventRejected represents ccsmp.SolClientSessionEventRejectedMsgError in sessionEventMapping
	SolClientEventRejected

	// SolClientEventReconnect represents ccsmp.SolClientSessionEventReconnectedNotice in sessionEventMapping
	SolClientEventReconnect

	// SolClientEventReconnectAttempt represents ccsmp.SolClientSessionEventReconnectingNotice in sessionEventMapping
	SolClientEventReconnectAttempt

	// SolClientSubscriptionOk represents ccsmp.SolClientSessionEventSubscriptionOk in sessionEventMapping
	SolClientSubscriptionOk

	// SolClientSubscriptionError represents ccsmp.SolClientSessionEventSubscriptionError in sessionEventMapping
	SolClientSubscriptionError
)

var sessionEventMapping = map[ccsmp.SolClientSessionEvent]Event{
	ccsmp.SolClientSessionEventUpNotice:           SolClientEventUp,
	ccsmp.SolClientSessionEventDownError:          SolClientEventDown,
	ccsmp.SolClientSessionEventCanSend:            SolClientEventCanSend,
	ccsmp.SolClientSessionEventAcknowledgement:    SolClientEventAcknowledgement,
	ccsmp.SolClientSessionEventRejectedMsgError:   SolClientEventRejected,
	ccsmp.SolClientSessionEventReconnectedNotice:  SolClientEventReconnect,
	ccsmp.SolClientSessionEventReconnectingNotice: SolClientEventReconnectAttempt,
	ccsmp.SolClientSessionEventSubscriptionError:  SolClientSubscriptionError,
	ccsmp.SolClientSessionEventSubscriptionOk:     SolClientSubscriptionOk,
}

// Implementation
type ccsmpBackedEvents struct {
	eventHandlerLock sync.RWMutex
	eventHandlers    map[Event](map[uint]EventHandler)

	session *ccsmp.SolClientSession

	handlerID uint
}

type sessionEventInfo struct {
	err          error
	infoString   string
	correlationP unsafe.Pointer
	userP        unsafe.Pointer
}

func (info *sessionEventInfo) GetError() error {
	return info.err
}

func (info *sessionEventInfo) GetInfoString() string {
	return info.infoString
}

func (info *sessionEventInfo) GetCorrelationPointer() unsafe.Pointer {
	return info.correlationP
}

func newCcsmpEvents(session *ccsmp.SolClientSession) *ccsmpBackedEvents {
	return &ccsmpBackedEvents{
		eventHandlers: make(map[Event](map[uint]EventHandler)),
		session:       session,
	}
}

func (events *ccsmpBackedEvents) eventCallback(sessionEvent ccsmp.SolClientSessionEvent, responseCode ccsmp.SolClientResponseCode,
	info string, correlationP unsafe.Pointer, userP unsafe.Pointer) {
	events.eventHandlerLock.RLock()
	defer events.eventHandlerLock.RUnlock()

	if mappedEvent, ok := sessionEventMapping[sessionEvent]; ok {
		if logging.Default.IsDebugEnabled() {
			logging.Default.Debug(fmt.Sprintf("Received event %d from C, mapped to event %d", sessionEvent, mappedEvent))
		}
		eventHandlers := events.eventHandlers[mappedEvent]
		lastErrorInfo := ccsmp.GetLastErrorInfo(0)
		if logging.Default.IsDebugEnabled() {
			logging.Default.Debug(fmt.Sprintf("Retrieved Last Error Info: %s", lastErrorInfo))
		}
		var err error
		if lastErrorInfo.SubCode != ccsmp.SolClientSubCodeOK {
			err = ToNativeError(lastErrorInfo)
		}
		for _, eventHandler := range eventHandlers {
			eventHandler(&sessionEventInfo{err: err, infoString: info, correlationP: correlationP, userP: userP})
		}
	}
}

func (events *ccsmpBackedEvents) start() {
	events.session.SetEventCallback(events.eventCallback)
}

func (events *ccsmpBackedEvents) terminate() {
	events.eventHandlerLock.Lock()
	defer events.eventHandlerLock.Unlock()
	for id := range events.eventHandlers {
		delete(events.eventHandlers, id)
	}
	events.session.SetEventCallback(nil)
}

func (events *ccsmpBackedEvents) emitEvent(event Event, eventInfo SessionEventInfo) {
	events.eventHandlerLock.RLock()
	defer events.eventHandlerLock.RUnlock()
	eventHandlers := events.eventHandlers[event]
	for _, eventHandler := range eventHandlers {
		eventHandler(eventInfo)
	}
}

func (events *ccsmpBackedEvents) AddEventHandler(sessionEvent Event, eventHandler EventHandler) uint {
	// acquire full lock since we are setting transport.eventHandlers, we always want adding to be entirely serial.
	events.eventHandlerLock.Lock()
	defer events.eventHandlerLock.Unlock()
	// shouldn't happen
	if events.eventHandlers == nil {
		logging.Default.Debug(fmt.Sprintf("attempted to add event handler %p for event %d to terminated events service", eventHandler, sessionEvent))
		return 0
	}
	eventHandlers, ok := events.eventHandlers[sessionEvent]
	// if we don't have an event handlers map yet for the given map, create a new one
	if !ok {
		eventHandlers = make(map[uint]EventHandler)
		events.eventHandlers[sessionEvent] = eventHandlers
	}
	events.handlerID++
	eventHandlers[events.handlerID] = eventHandler
	return events.handlerID
}

func (events *ccsmpBackedEvents) RemoveEventHandler(id uint) {
	events.eventHandlerLock.Lock()
	defer events.eventHandlerLock.Unlock()
	for _, eventHandlers := range events.eventHandlers {
		if _, ok := eventHandlers[id]; ok {
			delete(eventHandlers, id)
			break
		}
	}
}
