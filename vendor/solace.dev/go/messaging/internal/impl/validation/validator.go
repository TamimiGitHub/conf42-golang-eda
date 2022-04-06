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

// Package validation is defined below
package validation

import (
	"fmt"
	"strings"

	"solace.dev/go/messaging/pkg/solace"
)

// TODO we need to come back to this section and review each type of validator

func requiredPropertyError(key string) error {
	return solace.NewError(&solace.InvalidConfigurationError{}, fmt.Sprintf("expected required property %s to be set", key), nil)
}

func wrongTypeError(key, t string, value interface{}) error {
	return solace.NewError(&solace.IllegalArgumentError{}, fmt.Sprintf("expected configured value for %s to be %s, got %T", key, t, value), nil)
}

// StringPropertyValidation function
func StringPropertyValidation(key string, property interface{}, allowed ...string) (string, bool, error) {
	if property == nil {
		return "", false, requiredPropertyError(key)
	}
	propertyAsString, ok := property.(string)
	if !ok {
		return "", true, wrongTypeError(key, "string", property)
	}
	if len(allowed) > 0 {
		for _, str := range allowed {
			if str == propertyAsString {
				return str, true, nil
			}
		}
		return "", true, solace.NewError(&solace.IllegalArgumentError{}, fmt.Sprintf("got invalid value '%s' for property %s, valid values are [%s]", propertyAsString, key, strings.Join(allowed, ",")), nil)
	}

	return propertyAsString, true, nil
}

// IntegerPropertyValidation function
func IntegerPropertyValidation(key string, property interface{}) (int, bool, error) {
	if property == nil {
		return 0, false, requiredPropertyError(key)
	}
	var integerValue int
	switch convertedProperty := property.(type) {
	case int:
		integerValue = convertedProperty
	case uint:
		integerValue = int(convertedProperty)
	case float64: // comes from json.Unmarshal
		integerValue = int(convertedProperty)
	default:
		return 0, true, wrongTypeError(key, "integer", property)
	}
	return integerValue, true, nil
}

// Int64PropertyValidation function
func Int64PropertyValidation(key string, property interface{}) (int64, bool, error) {
	if property == nil {
		return 0, false, requiredPropertyError(key)
	}
	var ret int64
	switch converted := property.(type) {
	case int64:
		ret = converted
	case int32:
		ret = int64(converted)
	case int:
		ret = int64(converted)
	default:
		return 0, true, wrongTypeError(key, "int64", property)
	}
	return ret, true, nil
}

// Uint64PropertyValidation function
func Uint64PropertyValidation(key string, property interface{}) (uint64, bool, error) {
	if property == nil {
		return 0, false, requiredPropertyError(key)
	}
	var ret uint64
	switch converted := property.(type) {
	case uint64:
		ret = converted
	case uint32:
		ret = uint64(converted)
	case uint:
		ret = uint64(converted)
	case int:
		ret = uint64(converted)
	default:
		return 0, true, wrongTypeError(key, "uint64", property)
	}
	return ret, true, nil
}

// IntegerPropertyValidationWithRange function
func IntegerPropertyValidationWithRange(key string, property interface{}, min int, max int) (int, bool, error) {
	integer, present, err := IntegerPropertyValidation(key, property)
	if !present || err != nil {
		return integer, present, err
	}
	if integer >= min && integer <= max {
		return integer, true, nil
	}
	return 0, true, solace.NewError(&solace.IllegalArgumentError{}, fmt.Sprintf("expected configured value for %s to be within range [%d,%d], got %d", key, min, max, integer), nil)
}

// BooleanPropertyValidation function
func BooleanPropertyValidation(key string, property interface{}) (bool, bool, error) {
	if property == nil {
		return false, false, requiredPropertyError(key)
	}
	var ret bool
	switch casted := property.(type) {
	case int:
		ret = casted == 1
	case bool:
		ret = casted
	case string:
		ret = casted == "true" || casted == "True"
	default:
		return false, true, wrongTypeError(key, "bool", property)
	}
	return ret, true, nil
}
