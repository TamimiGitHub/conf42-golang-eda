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

// Package config contains the following  constructs used for configuration:
//
// - a set of keys that can be used to configure a variety of messaging services,
// such as a solace.MessagingService, solace.DirectMessageReceiver, etc.
//
// - a set of strategies that can be used to configure various features on
// various builders, such as Kerberos authentication on a MessagingServiceBuilder and
// Replay on a PersistentMessageReceiver.
//
// - a collection of map types that represent a valid configuration that can be passed
// to a builder.
//
// For the various maps, such as a ServicePropertyMap, the configuration may be loaded
// as JSON as shown:
//
//  import "encoding/json"
//
//  ...
//
//  var myJsonConfig []byte := []byte(`{"solace":{"messaging":{"transport":{"host":"10.10.10.10"}}}}`)
//  var myServicePropertyMap config.ServicePropertyMap
//  json.Unmarshal(myJsonConfig, &myServicePropertyMap)
//  messaging.NewMessagingServiceBuilder().FromConfigurationProvider(myServicePropertyMap)
//  ...
//
package config
