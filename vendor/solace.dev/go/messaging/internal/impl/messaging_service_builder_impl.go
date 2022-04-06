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

// Package impl is defined below
package impl

import (
	"fmt"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/internal/impl/constants"
	"solace.dev/go/messaging/internal/impl/core"
	"solace.dev/go/messaging/internal/impl/logging"
	"solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/config"
)

// NewMessagingServiceBuilder creates a messaging service builder
func NewMessagingServiceBuilder() solace.MessagingServiceBuilder {
	builder := &messagingServiceBuilderImpl{}
	builder.logger = logging.For(builder)
	builder.configuration = constants.DefaultConfiguration.GetConfiguration()
	return builder
}

type messagingServiceBuilderImpl struct {
	configuration config.ServicePropertyMap
	logger        logging.LogLevelLogger
}

func (builder *messagingServiceBuilderImpl) mergeProperties(properties config.ServicePropertyMap) {
	if properties == nil {
		return
	}
	for key, value := range properties {
		builder.configuration[key] = value
	}
}

// Build will build a new MessagingService based on the provided configuration.
// Returns the built MessagingService instance or nil if an error occurred.
// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
func (builder *messagingServiceBuilderImpl) Build() (solace.MessagingService, error) {
	// Start by cloning the configuration
	configuration := builder.configuration.GetConfiguration()
	// validate configuration
	if err := validateRequiredProperties(configuration); err != nil {
		return nil, err
	}

	logger := builder.logger

	// the host string is used in various callbacks for service events
	// TODO should we retrieve the host from the session after its created?
	host := configuration[config.TransportLayerPropertyHost]
	hostString := defaultConverter(host)

	// Build the basic property list
	propertyList := toPropertyList(configuration, logger)

	// Build the messaging service
	messagingService := newMessagingServiceImpl(logger)
	transport, err := core.NewTransport(hostString, propertyList)
	if err != nil {
		return nil, err
	}
	messagingService.transport = transport
	return messagingService, nil
}

// Converts the properties configured in this builder to a string list parsable by ccsmp
func toPropertyList(servicePropertyMap config.ServicePropertyMap, logger logging.LogLevelLogger) []string {
	propertyList := []string{}

	// First we must include all default properties
	for key, value := range constants.DefaultCcsmpProperties {
		propertyList = append(propertyList, key)
		propertyList = append(propertyList, value)
	}

	// Determine the correct username to use. It will not have been set as part of the loop above.
	// The username is not part of the servicePropertyToCCSMPMap, thus the usernames will not make it in
	if username, ok := getUsername(servicePropertyMap, logger); ok {
		propertyList = append(propertyList, ccsmp.SolClientSessionPropUsername)
		propertyList = append(propertyList, username)
	}

	// Then we must iterate over the builder's configuration and set all the appropriate keys
	for key, value := range servicePropertyMap {
		ccsmpProperty, ok := servicePropertyToCCSMPMap[key]
		if !ok {
			logger.Debug("Passed an unused property: " + string(key))
			continue
		}
		if value == nil {
			continue
		}
		propertyList = append(propertyList, ccsmpProperty.solClientPropertyName)
		propertyList = append(propertyList, ccsmpProperty.converter(value))
	}

	return propertyList
}

func validateRequiredProperties(servicePropertyMap config.ServicePropertyMap) error {
	for _, requiredProp := range requiredServiceProperties {
		if _, ok := servicePropertyMap[requiredProp]; !ok {
			return solace.NewError(&solace.InvalidConfigurationError{},
				fmt.Sprintf(constants.MissingServiceProperty, requiredProp), nil)
		}
	}
	scheme, ok := servicePropertyMap[config.AuthenticationPropertyScheme]
	if !ok {
		scheme = constants.DefaultAuthenticationScheme
	}
	requiredProps, ok := authenticationSchemeRequiredPropertyMap[defaultConverter(scheme)]
	if !ok {
		return nil
	}
	for _, requiredPropsForScheme := range requiredProps {
		hasRequiredProp := false
		for _, requiredProp := range requiredPropsForScheme {
			if _, ok := servicePropertyMap[requiredProp]; ok {
				hasRequiredProp = true
			}
		}
		if !hasRequiredProp {
			var errorMessage string
			if len(requiredPropsForScheme) > 1 {
				errorMessage = fmt.Sprintf(constants.MissingServicePropertiesGivenScheme, requiredPropsForScheme, scheme)
			} else {
				errorMessage = fmt.Sprintf(constants.MissingServicePropertyGivenScheme, requiredPropsForScheme, scheme)
			}
			return solace.NewError(&solace.InvalidConfigurationError{}, errorMessage, nil)
		}
	}
	return nil
}

// Gets the username to use
func getUsername(servicePropertyMap config.ServicePropertyMap, logger logging.LogLevelLogger) (string, bool) {
	scheme, ok := servicePropertyMap[config.AuthenticationPropertyScheme]
	if !ok {
		scheme = constants.DefaultAuthenticationScheme
	}
	configuredUsernameKey, ok := authenticationStrategyToUsernamePropertyMapping[defaultConverter(scheme)]
	if ok {
		username, ok := servicePropertyMap[configuredUsernameKey]
		if ok {
			// we want to remove the property so it is not reported as "unused"
			delete(servicePropertyMap, configuredUsernameKey)
			return defaultConverter(username), true
		}
	}
	return "", false
}

// BuildWithApplicationID will build a new MessagingService based on the provided configuration
// using the given application ID as the application ID.
// Returns the built MessagingService instance or nil if an error occurred.
// Returns solace/errors.*InvalidConfigurationError if an invalid configuration is provided.
func (builder *messagingServiceBuilderImpl) BuildWithApplicationID(applicationID string) (solace.MessagingService, error) {
	builder.configuration[config.ClientPropertyName] = applicationID
	return builder.Build()
}

// FromConfigurationProvider sets the configuration based on the given confniguration provider.
// The following are built in configuration providers:
//   ServicePropertyMap: can be used to set a ServiceProperty to a value programatically
// The ServicePropertiesConfigurationProvider interface can also be implemented by a type
// to have it act as a configuration factory by implementing
//   func (type MyType) GetConfiguration() ServicePropertyMap {...}
// Any properties provided by the configuration provider will be layered overtop of any
// previously set properties, including those set by specifying various strategies.
func (builder *messagingServiceBuilderImpl) FromConfigurationProvider(provider config.ServicePropertiesConfigurationProvider) solace.MessagingServiceBuilder {
	if provider == nil {
		return builder
	}
	builder.mergeProperties(provider.GetConfiguration())
	return builder
}

// WithAuthenticationStrategy will configure the resulting messaging service
// with the given authentication configuration
func (builder *messagingServiceBuilderImpl) WithAuthenticationStrategy(authenticationStrategy config.AuthenticationStrategy) solace.MessagingServiceBuilder {
	builder.mergeProperties(authenticationStrategy.ToProperties())
	return builder
}

// WithRetryStrategy will configure the resulting messaging service
// with the given retry strategy
func (builder *messagingServiceBuilderImpl) WithConnectionRetryStrategy(retryStrategy config.RetryStrategy) solace.MessagingServiceBuilder {
	builder.mergeProperties(config.ServicePropertyMap{
		config.TransportLayerPropertyConnectionRetries:                retryStrategy.GetRetries(),
		config.TransportLayerPropertyReconnectionAttemptsWaitInterval: retryStrategy.GetRetryInterval(),
	})
	return builder
}

// WithMessageCompression will configure the resulting messaging service
// with the given compression factor. The builder will attempt to use
// the given compression level with the given host and port, and will
// fail to build if attempting to use compression on non-secured and
// non-compressed port.
func (builder *messagingServiceBuilderImpl) WithMessageCompression(compressionFactor int) solace.MessagingServiceBuilder {
	builder.configuration[config.TransportLayerPropertyCompressionLevel] = compressionFactor
	return builder
}

// WithReconnectionRetryStrategy will configure the resulting messaging service
// with the given reconnection strategy
func (builder *messagingServiceBuilderImpl) WithReconnectionRetryStrategy(retryStrategy config.RetryStrategy) solace.MessagingServiceBuilder {
	builder.mergeProperties(config.ServicePropertyMap{
		config.TransportLayerPropertyReconnectionAttempts:             retryStrategy.GetRetries(),
		config.TransportLayerPropertyReconnectionAttemptsWaitInterval: retryStrategy.GetRetryInterval(),
	})
	return builder
}

// WithTransportSecurityStrategy will configure the resulting messaging service
// with the given transport security strategy
func (builder *messagingServiceBuilderImpl) WithTransportSecurityStrategy(transportSecurityStrategy config.TransportSecurityStrategy) solace.MessagingServiceBuilder {
	builder.mergeProperties(transportSecurityStrategy.ToProperties())
	return builder
}

func (builder *messagingServiceBuilderImpl) String() string {
	return fmt.Sprintf("solace.MessagingServiceBuilder at %p", builder)
}
