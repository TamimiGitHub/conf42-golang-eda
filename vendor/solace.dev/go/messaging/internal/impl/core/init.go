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
	"os"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/logging"
)

var supportedKeys = []string{
	ccsmp.SolClientGlobalPropGssKrbLib,
	ccsmp.SolClientGlobalPropCryptoLib,
	ccsmp.SolClientGlobalPropSslLib,
	"SOLCLIENT_GLOBAL_PROP_GSS_KRB_LIB",
	"SOLCLIENT_GLOBAL_PROP_CRYPTO_LIB",
	"SOLCLIENT_GLOBAL_PROP_SSL_LIB",
}

var libraryEnvironmentMapping = map[string]string{
	"SOLCLIENT_GLOBAL_PROP_GSS_KRB_LIB": ccsmp.SolClientGlobalPropGssKrbLib,
	"SOLCLIENT_GLOBAL_PROP_CRYPTO_LIB":  ccsmp.SolClientGlobalPropCryptoLib,
	"SOLCLIENT_GLOBAL_PROP_SSL_LIB":     ccsmp.SolClientGlobalPropSslLib,
}

// ccsmp initialization, calls solClient_initialize
func init() {
	propertyMap := make(map[string]string)
	for _, env := range supportedKeys {
		if val, ok := os.LookupEnv(env); ok {
			var key string
			if mapping, ok := libraryEnvironmentMapping[env]; ok {
				key = mapping
			} else {
				key = env
			}
			propertyMap[key] = val
		}
	}
	propertyList := []string{}
	for key, value := range propertyMap {
		propertyList = append(propertyList, key, value)
	}
	err := ccsmp.SolClientInitialize(propertyList)
	if err != nil {
		logging.Default.Critical("Encountered an error while initializing Solace native API" + err.GetMessageAsString())
		// We should be careful of panics. This is a good place for a panic as this error is unrecoverable
		panic(fmt.Sprintf("error initializing Solace native API: %s", err.GetMessageAsString()))
	}
	err = ccsmp.SetLogCallback(logCallback)
	if err != nil {
		logging.Default.Error("Encountered error while setting native log callback: " + err.GetMessageAsString())
	}
	SetNativeLogLevel(logging.Default.GetLevel())
}
