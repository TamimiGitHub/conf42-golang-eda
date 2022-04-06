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

package message

import (
	"fmt"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/internal/impl/logging"
	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/message/rgmid"
)

type replicationGroupMessageID struct {
	rmidPt ccsmp.SolClientRGMIDPt
}

// ReplicationGroupMessageIDFromString function
func ReplicationGroupMessageIDFromString(str string) (rgmid.ReplicationGroupMessageID, error) {
	rmidPt, errorInfo := ccsmp.SolClientRGMIDFromString(str)
	if errorInfo != nil {
		return nil, solace.NewError(&solace.IllegalArgumentError{},
			fmt.Sprintf(constants.CouldNotCreateRGMID, str),
			core.ToNativeError(errorInfo))
	}
	return &replicationGroupMessageID{rmidPt: rmidPt}, nil
}

// Compare compares the ReplicationGroupMessageID to another.
// Not all valid ReplicationGroupMessageID
// instances can be compared. If the messages identified were not published
// to the same broker or HA pair, then they are not comparable and a
// solace.IllegalArgumentError is returned.
func (rmid *replicationGroupMessageID) Compare(other rgmid.ReplicationGroupMessageID) (int, error) {
	casted, ok := other.(*replicationGroupMessageID)
	if !ok {
		return 0, solace.NewError(&solace.IllegalArgumentError{},
			fmt.Sprintf(constants.CouldNotCompareRGMIDBadType, other), nil)
	}
	result, errorInfo := ccsmp.SolClientRGMIDCompare(rmid.rmidPt, casted.rmidPt)
	if errorInfo != nil {
		return 0, solace.NewError(&solace.IllegalArgumentError{},
			fmt.Sprintf(constants.CouldNotCompareRGMID, errorInfo.GetMessageAsString()),
			core.ToNativeError(errorInfo))
	}
	return result, nil
}

// String returns the ReplicationGroupMessageID as a string
func (rmid *replicationGroupMessageID) String() string {
	str, errorInfo := ccsmp.SolClientRGMIDToString(rmid.rmidPt)
	if errorInfo != nil {
		logging.Default.Warning("Failed to convert ReplicationGroupMessageID to string: " + errorInfo.GetMessageAsString())
	}
	return str
}
