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

// This file gives access from the impl package to the message's ReplicationGroupMessageIDFromString allocator

import (
	"solace.dev/go/messaging/internal/impl/message"
	"solace.dev/go/messaging/pkg/solace/message/rgmid"
)

// ReplicationGroupMessageIDOf function
func ReplicationGroupMessageIDOf(str string) (rgmid.ReplicationGroupMessageID, error) {
	return message.ReplicationGroupMessageIDFromString(str)
}
