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
	"fmt"
	"time"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/internal/impl/logging"
)

const dateFormat = "Jan _2 15:04:05"

// SetVersion function
func SetVersion(defaultVersion string) {
	err, ccsmpVersion, ccsmpBuildTime, ccsmpVariant := ccsmp.SolClientVersionGet()

	if err == nil {
		var newVersion, newBuildTime, newVariant string

		// defaults
		golangVersionString := defaultVersion

		newVersion = fmt.Sprintf("%s %s / %s %s", constants.APIName, golangVersionString, constants.CoreAPIName, ccsmpVersion)
		newBuildTime = fmt.Sprintf("%s %s / %s %s", constants.APIName, time.Now().Format(dateFormat), constants.CoreAPIName, ccsmpBuildTime)
		newVariant = fmt.Sprintf("%s / %s", constants.APIName, ccsmpVariant)

		// Take information and set version
		ccsmp.SolClientVersionSet(newVersion, newBuildTime, newVariant)
	} else {
		logging.Default.Error("Failed to get core library version: " + err.GetMessageAsString())
	}
}

// GetVersion function
func GetVersion() (version, buildTime, variant string) {
	err, ccsmpVersion, ccsmpBuildTime, ccsmpVariant := ccsmp.SolClientVersionGet()
	if err != nil {
		logging.Default.Error("Failed to get core library version: " + err.GetMessageAsString())
	}
	return ccsmpVersion, ccsmpBuildTime, ccsmpVariant
}
