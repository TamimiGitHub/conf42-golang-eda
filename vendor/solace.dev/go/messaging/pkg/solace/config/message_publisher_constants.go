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

// The various back pressure strategies that can be used when configuring Publisher back pressure.
const (
	// PublisherPropertyBackPressureStrategyBufferWaitWhenFull waits for space on a call to publish
	// when the publisher's buffer is full.
	PublisherPropertyBackPressureStrategyBufferWaitWhenFull = "BUFFER_WAIT_WHEN_FULL"
	// PublisherPropertyBackPressureStrategyBufferRejectWhenFull rejects the call to publish
	// when the publisher's buffer is full.
	PublisherPropertyBackPressureStrategyBufferRejectWhenFull = "BUFFER_REJECT_WHEN_FULL"
)
