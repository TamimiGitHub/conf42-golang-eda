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

package constants

import (
	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/pkg/solace/config"
)

// This file defines the set of default properties used in various services
// such as the messaging service and publishers/receivers

// DefaultAuthenticationScheme is the default auth strategy to use
const DefaultAuthenticationScheme = config.AuthenticationSchemeBasic

// DefaultConfiguration refers to any default configuration values that should override the default CCSMP values here
var DefaultConfiguration = config.ServicePropertyMap{}

// DefaultCcsmpProperties contains the default properties to use when building a new session
var DefaultCcsmpProperties = map[string]string{
	ccsmp.SolClientSessionPropTopicDispatch:              ccsmp.SolClientPropEnableVal,
	ccsmp.SolClientSessionPropSendBlocking:               ccsmp.SolClientPropDisableVal,
	ccsmp.SolClientSessionPropReapplySubscriptions:       ccsmp.SolClientPropEnableVal,
	ccsmp.SolClientSessionPropIgnoreDupSubscriptionError: ccsmp.SolClientPropEnableVal,
	ccsmp.SolClientSessionPropReconnectRetries:           "3",
	ccsmp.SolClientSessionPropGuaranteedWithWebTransport: ccsmp.SolClientPropEnableVal,
	ccsmp.SolClientSessionPropPubWindowSize:              "255",
}

// Default properties for the various sub service

// DefaultDirectPublisherProperties contains the default properties for a DirectPublisher
var DefaultDirectPublisherProperties = config.PublisherPropertyMap{
	config.PublisherPropertyBackPressureBufferCapacity: 50,
	config.PublisherPropertyBackPressureStrategy:       config.PublisherPropertyBackPressureStrategyBufferWaitWhenFull,
}

// DefaultPersistentPublisherProperties contains the default properties for a PersistentPublisher
var DefaultPersistentPublisherProperties = config.PublisherPropertyMap{
	config.PublisherPropertyBackPressureBufferCapacity: 50,
	config.PublisherPropertyBackPressureStrategy:       config.PublisherPropertyBackPressureStrategyBufferWaitWhenFull,
}

// DefaultDirectReceiverProperties contains the default properties for a DirectReceiver
var DefaultDirectReceiverProperties = config.ReceiverPropertyMap{
	config.ReceiverPropertyDirectBackPressureStrategy:       config.ReceiverBackPressureStrategyDropLatest,
	config.ReceiverPropertyDirectBackPressureBufferCapacity: 50,
}

// DefaultPersistentReceiverProperties contains the default properties for a PersistentReceiver
var DefaultPersistentReceiverProperties = config.ReceiverPropertyMap{}
