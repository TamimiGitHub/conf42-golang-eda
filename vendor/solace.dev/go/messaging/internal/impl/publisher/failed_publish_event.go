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

package publisher

import (
	"time"

	"solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/resource"
)

type failedPublishEvent struct {
	message   message.OutboundMessage
	dest      resource.Destination
	timestamp time.Time
	err       error
}

// GetMessage gets the message that was not delivered
func (event *failedPublishEvent) GetMessage() message.OutboundMessage {
	return event.message
}

// GetDestination gets the destination that the message was published to
func (event *failedPublishEvent) GetDestination() resource.Destination {
	return event.dest
}

// GetTimestamp gets the timestamp of the error
func (event *failedPublishEvent) GetTimeStamp() time.Time {
	return event.timestamp
}

// GetError gets the error that failed the publish attempt
func (event *failedPublishEvent) GetError() error {
	return event.err
}
