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

// Package subcode contains the subcodes returned from the Solace PubSub+ Messaging API for C.
// The subcodes are generated in subcode_generated.go. This is an advanced feature and should
// be used only when absolutely necessary.
package subcode

import "solace.dev/go/messaging/internal/ccsmp"

//go:generate go run subcode_generator.go $SOLCLIENT_H

// Code represents the various subcodes that can be returned as part of SolClientError.
type Code int

// Is checks if the given Code equals any of the entries.
func Is(subCode Code, entries ...Code) bool {
	for _, entry := range entries {
		if entry == subCode {
			return true
		}
	}
	return false
}

// String returns the subcode as a string value.
func (code Code) String() string {
	return ccsmp.SolClientSubCodeToString(ccsmp.SolClientSubCode(code))
}
