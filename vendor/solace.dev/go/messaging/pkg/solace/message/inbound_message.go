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

package message

import (
	"time"

	"solace.dev/go/messaging/pkg/solace/message/rgmid"
)

// InboundMessage represents a message received by a consumer.
type InboundMessage interface {
	// Extend the Message interface.
	Message

	// GetDestinationName retrieves the destination name on which the message was received.
	// The destination may be either a topic or a queue.
	// An empty string is returned if the information is not available.
	GetDestinationName() string

	// GetTimeStamp retrieves the timestamp as time.Time.
	// This timestamp represents the time that the message was received by the API.
	// This may differ from the time that the message is received by the MessageReceiver.
	GetTimeStamp() (timestamp time.Time, ok bool)

	// GetSenderTimestamp retrieves the timestamp as time.Time.
	// This timestamp is often set automatically when the message is published.
	GetSenderTimestamp() (senderTimestamp time.Time, ok bool)

	// GetSenderID retrieves the sender's ID. This field can be set automatically during
	// message publishing, but existing values are not overwritten if not empty, for example
	// if the message is sent multiple times.
	// Returns the sender's ID and a boolean indicating if it was set.
	GetSenderID() (senderID string, ok bool)

	// GetReplicationGroupMessageID retrieves the Replication Group Message ID.
	// Returns an empty string and false if no Replication Group Message ID is set on the
	// message (for example, direct messages or unsupported broker versions)
	GetReplicationGroupMessageID() (replicationGroupMessageID rgmid.ReplicationGroupMessageID, ok bool)

	// GetMessageDiscardNotification retrieves the message discard notification about
	// previously discarded messages. Returns a MessageDiscardNotification and is  not expected
	// to be nil.
	GetMessageDiscardNotification() MessageDiscardNotification

	// IsRedelivered retrieves the message's redelivery status. Returns true if the message
	// redelivery occurred in the past, otherwise false.
	IsRedelivered() bool
}

// MessageDiscardNotification is used to indicate that there are discarded messages.
type MessageDiscardNotification interface {
	// HasBrokerDiscardIndication determines whether the broker has discarded one
	// or more messages prior to the current message.
	// Returns true if messages (one or more) were previously discarded by the broker,
	// otherwise false.
	HasBrokerDiscardIndication() bool

	// HasInternalDiscardIndication determines if the API has discarded one or more messages
	// prior to the current message (i.e., in a back-pressure situation).
	// Returns true if messages (one or more) were previously discarded by the API, otherwise false.
	HasInternalDiscardIndication() bool
}
