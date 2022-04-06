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

import (
	"time"

	"solace.dev/go/messaging/pkg/solace/message/rgmid"
)

// ReplayStrategy is an interface for Message Replay strategies.
// Message Replay lets applications retrieve messages from an event
// broker hours or even days after those messages were first received
// by the broker.
type ReplayStrategy struct {
	strategy string
	data     interface{}
}

// GetStrategy returns the strategy type stored in this ReplayStrategy.
func (strategy ReplayStrategy) GetStrategy() string {
	return strategy.strategy
}

// GetData returns the additional data, if applicable, stored with the strategy.
// For example, for time based replay this contains the start date.
func (strategy ReplayStrategy) GetData() interface{} {
	return strategy.data
}

// ReplayStrategyAllMessages sets the strategy to replaying all
// messages stored on the broker's replay log.
func ReplayStrategyAllMessages() ReplayStrategy {
	return ReplayStrategy{
		strategy: PersistentReplayAll,
		data:     nil,
	}
}

// ReplayStrategyTimeBased sets the strategy to replay
// all messages stored on the broker's replay log dated at or
// after the specified replayDate. Note that the replayDate given
// should be set to the correct timezone from which replay is
// meant to begin. See time.Date for the appropriate
// configuration of a timezone aware time.Time value.
func ReplayStrategyTimeBased(replayDate time.Time) ReplayStrategy {
	return ReplayStrategy{
		strategy: PersistentReplayTimeBased,
		data:     replayDate,
	}
}

// ReplayStrategyReplicationGroupMessageID sets the strategy
// to replay all messages stored on the broker's replay log
// received at or after the specified messageID. Returns the constructed
// ReplayStrategy.
// Valid Replication Group Message IDs take the form
//	rmid1:xxxxx-xxxxxxxxxxx-xxxxxxxx-xxxxxxxx
// where x is a valid hexadecimal digit.
func ReplayStrategyReplicationGroupMessageID(replicationGroupMessageID rgmid.ReplicationGroupMessageID) ReplayStrategy {
	return ReplayStrategy{
		strategy: PersistentReplayIDBased,
		data:     replicationGroupMessageID,
	}
}
