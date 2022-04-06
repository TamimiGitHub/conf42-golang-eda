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

// PublisherPropertiesConfigurationProvider describes the behavior of a configuration provider
// that provides properties for publishers.
type PublisherPropertiesConfigurationProvider interface {
	GetConfiguration() PublisherPropertyMap
}

// PublisherProperty is a property that can be set on a publisher.
type PublisherProperty string

// PublisherPropertyMap is a map of PublisherProperty keys to values.
type PublisherPropertyMap map[PublisherProperty]interface{}

// GetConfiguration returns a copy of the PublisherPropertyMap.
func (publisherPropertyMap PublisherPropertyMap) GetConfiguration() PublisherPropertyMap {
	ret := make(PublisherPropertyMap)
	for key, value := range publisherPropertyMap {
		ret[key] = value
	}
	return ret
}

// MarshalJSON implements the json.Marshaler interface.
func (publisherPropertyMap PublisherPropertyMap) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	for k, v := range publisherPropertyMap {
		m[string(k)] = v
	}
	return nestJSON(m)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (publisherPropertyMap PublisherPropertyMap) UnmarshalJSON(b []byte) error {
	m, err := flattenJSON(b)
	if err != nil {
		return err
	}
	for key, val := range m {
		publisherPropertyMap[PublisherProperty(key)] = val
	}
	return nil
}

const (
	// PublisherPropertyBackPressureStrategy sets the back pressure strategy on a publisher.
	// Valid values are BUFFER_WAIT_WHEN_FULL or BUFFER_REJECT_WHEN_FULL where BUFFER_WAIT_WHEN_FULL
	// will block until space is available and BUFFER_REJECT_WHEN_FULL will return an error if the buffer
	// is full.
	PublisherPropertyBackPressureStrategy PublisherProperty = "solace.messaging.publisher.back-pressure.strategy"
	// PublisherPropertyBackPressureBufferCapacity sets the buffer size of the publisher.
	// Valid values are greater than or equal to 1 for back pressure strategy Wait and greater than or
	// equal to 0 for back pressure strategy Reject.
	PublisherPropertyBackPressureBufferCapacity PublisherProperty = "solace.messaging.publisher.back-pressure.buffer-capacity"
)
