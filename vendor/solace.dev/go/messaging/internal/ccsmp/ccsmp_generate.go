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

package ccsmp

// Add any generate directives here so that any of the CCSMP enum definitions can be generated in golang.

//go:generate go run ./generator/ccsmp_return_code_generator.go $SOLCLIENT_H
//go:generate go run ./generator/ccsmp_session_event_generator.go $SOLCLIENT_H
//go:generate go run ./generator/ccsmp_session_prop_generator.go $SOLCLIENT_H
//go:generate go run ./generator/ccsmp_log_level_generator.go $SOLCLIENT_H
//go:generate go run ./generator/ccsmp_session_stats_generator.go $SOLCLIENT_H
//go:generate go run ./generator/ccsmp_flow_prop_generator.go $SOLCLIENT_H
//go:generate go run ./generator/ccsmp_flow_event_generator.go $SOLCLIENT_H
//go:generate go run ./generator/ccsmp_endpoint_prop_generator.go $SOLCLIENT_H
//go:generate go run ./generator/ccsmp_global_prop_generator.go $SOLCLIENT_H

// Type definitions for generated code

// SolClientSessionEvent type defined
type SolClientSessionEvent int

// SolClientFlowEvent type defined
type SolClientFlowEvent int

// SolClientLogLevel type defined
type SolClientLogLevel int

// SolClientStatsRX type defined
type SolClientStatsRX int

// SolClientStatsTX type defined
type SolClientStatsTX int
