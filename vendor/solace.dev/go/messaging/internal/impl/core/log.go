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
	"strconv"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/logging"
)

var logger = logging.Default.For(nil)

func logCallback(logInfo *ccsmp.LogInfo) {
	switch logInfo.Level {

	// Critical
	case ccsmp.SolClientLogLevelEmergency:
		fallthrough
	case ccsmp.SolClientLogLevelAlert:
		fallthrough
	case ccsmp.SolClientLogLevelCritical:
		logger.Critical(logInfo.Message)

	// Error
	case ccsmp.SolClientLogLevelError:
		logger.Error(logInfo.Message)

	// Warning
	case ccsmp.SolClientLogLevelWarning:
		logger.Warning(logInfo.Message)

	// Info
	case ccsmp.SolClientLogLevelInfo:
		fallthrough
	case ccsmp.SolClientLogLevelNotice:
		logger.Info(logInfo.Message)

	// Debug
	case ccsmp.SolClientLogLevelDebug:
		logger.Debug(logInfo.Message)

	// Default
	default:
		logging.Default.Error("UNKNOWN LOG LEVEL " + strconv.Itoa(int(logInfo.Level)))
		logging.Default.Error(logInfo.Message)
	}
}

// SetLogLevel function
func SetLogLevel(level logging.LogLevel) {
	// Set log level
	logging.Default.SetLevel(level)
	SetNativeLogLevel(level)
}

// SetNativeLogLevel function
func SetNativeLogLevel(level logging.LogLevel) {
	// Set native logging level
	var ccsmpLevel ccsmp.SolClientLogLevel
	switch level {
	case logging.Critical:
		ccsmpLevel = ccsmp.SolClientLogLevelCritical
	case logging.Error:
		ccsmpLevel = ccsmp.SolClientLogLevelError
	case logging.Warning:
		ccsmpLevel = ccsmp.SolClientLogLevelWarning
	case logging.Info:
		ccsmpLevel = ccsmp.SolClientLogLevelInfo
	case logging.Debug:
		ccsmpLevel = ccsmp.SolClientLogLevelDebug
	default:
		logging.Default.Debug("Cannot set log level to unknown level " + strconv.Itoa(int(level)))
		return
	}
	err := ccsmp.SetLogLevel(ccsmpLevel)
	if err != nil {
		logging.Default.Error("Encountered error while setting native log level: " + err.GetMessageAsString())
	}
}
