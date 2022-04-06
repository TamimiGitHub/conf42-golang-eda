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

package config

// The various back pressure strategies that can be used to configure a receiver.
const (
	// ReceiverBackPressureStrategyDropLatest drops the newest incoming message when the receiver's
	// buffer is full.
	ReceiverBackPressureStrategyDropLatest = "BUFFER_DROP_LATEST_WHEN_FULL"
	// ReceiverBackPressureStrategyDropOldest drops the oldest buffered message when the receiver's
	// buffer is full.
	ReceiverBackPressureStrategyDropOldest = "BUFFER_DROP_OLDEST_WHEN_FULL"
)

// The various replay strategies available when configuring Replay on a PersistentMessageReceiver.
const (
	// PersistentReplayAll replays all messages stored on a replay log.
	PersistentReplayAll = "REPLAY_ALL"
	// PersistentReplayTimeBased replays all messages stored on a replay log after the specified date.
	PersistentReplayTimeBased = "REPLAY_TIME_BASED"
	// PersistentReplayIDBased replays all the messages stored on a replay log after the given message ID.
	PersistentReplayIDBased = "REPLAY_ID_BASED"
)

// MissingResourcesCreationStrategy represents the various missing resource
// creation strategies available to publishers requiring guaranteed resources.
type MissingResourcesCreationStrategy string

const (
	// PersistentReceiverDoNotCreateMissingResources disables any attempt to create the missing
	// resource and is the default missing resource strategy. This value is
	// recommended for production.
	PersistentReceiverDoNotCreateMissingResources MissingResourcesCreationStrategy = "DO_NOT_CREATE"
	// PersistentReceiverCreateOnStartMissingResources attempts to create missing resources
	// once a connection is established, and only for resources known at that
	// time.
	PersistentReceiverCreateOnStartMissingResources MissingResourcesCreationStrategy = "CREATE_ON_START"
)

// The available acknowledgement strategies used on a PersistentMessageReceiver.
const (
	// PersistentReceiverAutoAck acknowledge the message automatically.
	PersistentReceiverAutoAck = "AUTO_ACK"
	// PersistentReceiverClientAck does not acknowledge the message and messages instead needs to be
	// acknowledged with PersistentMessageReceiver::Ack.
	PersistentReceiverClientAck = "CLIENT_ACK"
)
