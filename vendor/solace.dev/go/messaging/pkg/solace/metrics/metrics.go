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

// Package metrics contains the various metrics that can be retrieved as well as the
// interface for retrieving the metrics.
package metrics

// Metric represents the various metrics retrievable from a MessagingService's APIMetrics instance.
type Metric int
// The various metrics available.
const (
	// BrokerDiscardNotificationsReceived is the number of received messages
	// with discard indication set.
	BrokerDiscardNotificationsReceived Metric = iota

	// CompressedBytesReceived is the number of bytes received before decompression.
	CompressedBytesReceived

	// ConnectionAttempts is the total number of TCP connection attempts.
	ConnectionAttempts

	// ControlBytesReceived is the number of control (non-data) messages received
	// by the MessagingService.
	ControlBytesReceived

	// ControlBytesSent is the total number of control (non-data) bytes transmitted
	// by the MessagingService.
	ControlBytesSent

	// ControlMessagesReceived is the total number of control (non-data) messages
	// received by MessagingService.
	ControlMessagesReceived

	// ControlMessagesSent is the total number of control (non-data) messages
	// transmitted by MessagingService.
	ControlMessagesSent

	// DirectBytesReceived is the number of direct messaging bytes received
	// across all direct message publishers on the MessagingService.
	DirectBytesReceived

	// DirectBytesSent is the number of direct messaging bytes sent
	// across all direct message publishers on the MessagingService.
	DirectBytesSent

	// DirectMessagesReceived is the number of direct messages received
	// across all direct message publishers on the MessagingService.
	DirectMessagesReceived

	// DirectMessagesSent is the number of direct messages sent
	// across all direct message publishers on the MessagingService.
	DirectMessagesSent

	// InternalDiscardNotifications is the number of messages received with
	// internal discard notifications set.
	InternalDiscardNotifications

	// PersistentAcknowledgeSent is the number of acknowledgements
	// sent for guaranteed messaging across all persistent message receivers
	// on the MessagingService.
	PersistentAcknowledgeSent

	// PersistentBytesReceived is the number of persistent bytes received
	// across all persistent message receivers on the Messaging Service.
	PersistentBytesReceived

	// PersistentBytesRedelivered is the number of persistent bytes
	// redelivered across all persistent message publishers on the MessagingService.
	PersistentBytesRedelivered

	// PersistentBytesSent is the number of persistent bytes sent
	// across all persistent message publishers on the MessagingService.
	PersistentBytesSent

	// The number of guaranteed messages dropped for being duplicates.
	PersistentDuplicateMessagesDiscarded

	// PersistentMessagesReceived is the number of persistent messages
	// received across all persistent message receivers on the MessagingService.
	PersistentMessagesReceived

	// PersistentMessagesRedelivered is the number of persistent messages
	// redelivered across all persistent message publishers on the MessagingService.
	PersistentMessagesRedelivered

	// PersistentMessagesSent is the number of persistent messages
	// sent across all persistent message publishers on the MessagingService.
	PersistentMessagesSent

	// PersistentNoMatchingFlowMessagesDiscarded is the number of persistent
	// messages discarded for not having a matching flow on the MessagingService.
	PersistentNoMatchingFlowMessagesDiscarded

	// PersistentOutOfOrderMessagesDiscarded is the number of persistent
	// messages discarded for being received out of order across all
	// persistent message receivers on the MessagingService.
	PersistentOutOfOrderMessagesDiscarded

	// PublishMessagesDiscarded is the number of messages discarded due to
	// channel failure.
	PublishMessagesDiscarded

	// PublishedMessagesAcknowledged is the number of guaranteed messages that have
	// been published and acknowledged across all persistent message receivers
	// on the MessagingService
	PublishedMessagesAcknowledged

	// PublisherAcknowledgementReceived is the number of publisher acknowledgements
	// received by all persistent message publishers on the MessagingService.
	PublisherAcknowledgementReceived

	// PublisherAcknowledgementTimeouts is the number of expired acknowledgement timers
	// across all persistent message publishers on the MessagingService.
	PublisherAcknowledgementTimeouts

	// PublisherWindowClosed is the nunber of times the transmit window closed across
	// all message publishers on the MessagingService.
	PublisherWindowClosed

	// The number of messages not accepted due to would block (non-blocking publish only).
	PublisherWouldBlock

	// TotalBytesReceived is the total number of bytes received by the MessagingService
	// and all of its message receivers.
	TotalBytesReceived

	// TotalBytesSent is the total number of bytes sent by the MessagingService
	// and all of its message publishers.
	TotalBytesSent

	// TotalMessagesReceived is the total number of messages received by the
	// MessagingService and all of its receivers.
	TotalMessagesReceived

	// TotalMessagesSent is the total number of messages sent by the MessagingService
	// and all of its publishers.
	TotalMessagesSent

	// TooBigMessagesDiscarded is the number of messages discarded due to being too large.
	TooBigMessagesDiscarded

	// UnknownParameterMessagesDiscarded is the number of messages discarded due to the
	// presence of an unknown element or unknown protocol in the Solace Message Format
	// (SMF) header.
	UnknownParameterMessagesDiscarded

	// ReceivedMessagesTerminationDiscarded is the number of messages discarded due to
	// a receiver being terminated either by application initiated termination or
	// failure event termination.
	ReceivedMessagesTerminationDiscarded

	// ReceivedMessagesBackpressureDiscarded is the number of messages discarded due to
	// a receiver not having buffer space to queue a message
	ReceivedMessagesBackpressureDiscarded

	// PublishMessagesTerminationDiscarded is the number of messages discarded due to
	// a publisher being terminated either by application initiated termination or
	// failure event termination.
	PublishMessagesTerminationDiscarded

	// PublishMessagesBackpressureDiscarded is the number of messages discarded due to
	// a publisher not having buffer space to queue a message when in a buffered
	// backpressure configuration.
	PublishMessagesBackpressureDiscarded

	// MetricCount is the number of metrics defined by this package.
	MetricCount int = iota
)

// APIInfo allows for retrieval of various API properties transmitted to the Broker on connect.
type APIInfo interface {
	// GetAPIBuildDate returns the build date of the current API in use.
	GetAPIBuildDate() string
	// GetAPIVersion returns the version of the current API in use.
	GetAPIVersion() string
	// GetAPIUserID returns the user ID transmitted to the broker.
	GetAPIUserID() string
	// GetAPIImplementationVendor returns the API implementation vendor transmitted to broker.
	GetAPIImplementationVendor() string
}

// APIMetrics allows for retrieval of various metrics stored by the API.
type APIMetrics interface {
	// GetValue will retrieve the value/count of the specified Metric.
	GetValue(metric Metric) uint64
	// Reset resets all metrics.
	Reset()
}
