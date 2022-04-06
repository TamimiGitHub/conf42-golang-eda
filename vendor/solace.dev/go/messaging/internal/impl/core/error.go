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

package core

import (
	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/subcode"
)

// ErrorInfo reexports *ccsmp.SolClientErrorInfoWrapper.
// In all cases, core.ErrorInfo should be used instead of *ccsmp.SolClientErrorInfoWrapper
type ErrorInfo = *ccsmp.SolClientErrorInfoWrapper

// ToNativeError wrapps the ErrorInfo in a solace.NativeError with a prefix string if provided.
func ToNativeError(err ErrorInfo, args ...string) error {
	var prefix string
	if len(args) > 0 {
		prefix = args[0]
	}
	nativeError := solace.NewNativeError(prefix+err.GetMessageAsString(), subcode.Code(err.SubCode))
	switch nativeError.SubCode() {
	case subcode.LoginFailure:
		return solace.NewError(&solace.AuthenticationError{}, constants.LoginFailure+nativeError.Error(), nativeError)
	case subcode.UnresolvedHost:
		return solace.NewError(&solace.ServiceUnreachableError{}, constants.UnresolvedSession+nativeError.Error(), nativeError)
	case subcode.ParamConflict, subcode.ParamOutOfRange, subcode.ParamNullPtr:
		return solace.NewError(&solace.InvalidConfigurationError{}, constants.InvalidConfiguration+nativeError.Error(), nativeError)
	case subcode.ReplayNotSupported,
		subcode.ReplayDisabled,
		subcode.ClientInitiatedReplayNonExclusiveNotAllowed,
		subcode.ClientInitiatedReplayInactiveFlowNotAllowed,
		subcode.ClientInitiatedReplayBrowserFlowNotAllowed,
		subcode.ReplayTemporaryNotSupported,
		subcode.UnknownStartLocationType,
		subcode.ReplayMessageUnavailable,
		subcode.ReplayStarted,
		subcode.ReplayCancelled,
		subcode.ReplayStartTimeNotAvailable,
		subcode.ReplayMessageRejected,
		subcode.ReplayLogModified,
		subcode.MismatchedEndpointErrorID,
		subcode.OutOfReplayResources,
		subcode.ReplayStartMessageUnavailable:
		return solace.NewError(&solace.MessageReplayError{}, err.GetMessageAsString(), nativeError)
	default:
		return nativeError
	}
}
