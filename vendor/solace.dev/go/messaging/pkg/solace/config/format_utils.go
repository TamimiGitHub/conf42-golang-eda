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

import (
	"encoding/json"
	"fmt"
	"strings"
)

// For internal use only. Do not use.
var errIllegalKey = fmt.Errorf("path cannot be both a node and a leaf")

// For internal use only. Do not use.
func flattenMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, val := range m {
		castedMap, ok := val.(map[string]interface{})
		if ok {
			nestedResult := flattenMap(castedMap)
			for nestedKey, nestedVal := range nestedResult {
				result[key+"."+nestedKey] = nestedVal
			}
		} else {
			result[key] = val
		}
	}
	return result
}

// For internal use only. Do not use.
func nestMap(m map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range m {
		keys := strings.Split(k, ".")
		nested := result
		for i, key := range keys {
			if i+1 == len(keys) {
				_, ok := nested[key]
				if ok {
					return nil, errIllegalKey
				}
				nested[key] = v
			} else {
				val, ok := nested[key]
				if !ok {
					val = make(map[string]interface{})
					nested[key] = val
				}
				castedVal, ok := val.(map[string]interface{})
				if !ok {
					return nil, errIllegalKey
				}
				nested = castedVal
			}
		}
	}
	return result, nil
}

// For internal use only. Do not use.
func flattenJSON(b []byte) (map[string]interface{}, error) {
	parsed := make(map[string]interface{})
	err := json.Unmarshal(b, &parsed)
	if err != nil {
		return nil, err
	}
	return flattenMap(parsed), nil
}

// For internal use only. Do not use.
func nestJSON(m map[string]interface{}) ([]byte, error) {
	nestedMap, err := nestMap(m)
	if err != nil {
		return nil, err
	}
	return json.Marshal(nestedMap)
}
