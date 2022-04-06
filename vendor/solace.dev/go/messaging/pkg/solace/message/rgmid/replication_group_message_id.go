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

// Package rgmid contains the ReplicationGroupMessageID interface.
package rgmid

// ReplicationGroupMessageID represents a ReplicationGroupMessageID of a message.
type ReplicationGroupMessageID interface {
	// Compare compares the ReplicationGroupMessageID to another.
	// Not all valid ReplicationGroupMessageID
	// instances can be compared. If the messages identified were not published
	// to the same broker or HA pair, then they are not comparable and a
	// *solace.IllegalArgumentError is returned.
	Compare(ReplicationGroupMessageID) (result int, err error)
	// String returns the ReplicationGroupMessageID as a string value.
	String() string
}
