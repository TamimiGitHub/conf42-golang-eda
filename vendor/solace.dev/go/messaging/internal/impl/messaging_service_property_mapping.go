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

package impl

import (
	"fmt"
	"time"

	"solace.dev/go/messaging/internal/ccsmp"
	"solace.dev/go/messaging/pkg/solace/config"
)

// authenticationStrategyToUsernamePropertyMapping contains the mapping of authentication scheme to its relevant username property
var authenticationStrategyToUsernamePropertyMapping = map[string]config.ServiceProperty{
	config.AuthenticationSchemeBasic:             config.AuthenticationPropertySchemeBasicUserName,
	config.AuthenticationSchemeClientCertificate: config.AuthenticationPropertySchemeClientCertUserName,
	config.AuthenticationSchemeKerberos:          config.AuthenticationPropertySchemeKerberosUserName,
}

// authenticationSchemeRequiredPropertyMap contains the mapping of authentication scheme to required properties
var authenticationSchemeRequiredPropertyMap = map[string][][]config.ServiceProperty{
	config.AuthenticationSchemeBasic:             {{config.AuthenticationPropertySchemeBasicUserName}, {config.AuthenticationPropertySchemeBasicPassword}},
	config.AuthenticationSchemeClientCertificate: {{config.AuthenticationPropertySchemeSSLClientCertFile}, {config.AuthenticationPropertySchemeSSLClientPrivateKeyFile}},
	config.AuthenticationSchemeOAuth2:            {{config.AuthenticationPropertySchemeOAuth2AccessToken, config.AuthenticationPropertySchemeOAuth2OIDCIDToken}},
}

var requiredServiceProperties = []config.ServiceProperty{
	config.TransportLayerPropertyHost,
	config.ServicePropertyVPNName,
}

// Any new properties must be added to this map
var servicePropertyToCCSMPMap = map[config.ServiceProperty]property{
	/* Authentication Properties */
	// Note: username properties should not be added to this map, and instead added to the authenticationStrategyToUsernamePropertyMapping map above
	config.AuthenticationPropertyScheme:                                 {ccsmp.SolClientSessionPropAuthenticationScheme, defaultConverter},
	config.AuthenticationPropertySchemeBasicPassword:                    {ccsmp.SolClientSessionPropPassword, defaultConverter},
	config.AuthenticationPropertySchemeSSLClientCertFile:                {ccsmp.SolClientSessionPropSslClientCertificateFile, defaultConverter},
	config.AuthenticationPropertySchemeSSLClientPrivateKeyFile:          {ccsmp.SolClientSessionPropSslClientPrivateKeyFile, defaultConverter},
	config.AuthenticationPropertySchemeClientCertPrivateKeyFilePassword: {ccsmp.SolClientSessionPropSslClientPrivateKeyFilePassword, defaultConverter},
	config.AuthenticationPropertySchemeKerberosInstanceName:             {ccsmp.SolClientSessionPropKrbServiceName, defaultConverter},
	config.AuthenticationPropertySchemeOAuth2AccessToken:                {ccsmp.SolClientSessionPropOauth2AccessToken, defaultConverter},
	config.AuthenticationPropertySchemeOAuth2IssuerIdentifier:           {ccsmp.SolClientSessionPropOauth2IssuerIdentifier, defaultConverter},
	config.AuthenticationPropertySchemeOAuth2OIDCIDToken:                {ccsmp.SolClientSessionPropOidcIDToken, defaultConverter},

	/* Client Properties */
	config.ClientPropertyName:                   {ccsmp.SolClientSessionPropClientName, defaultConverter},
	config.ClientPropertyApplicationDescription: {ccsmp.SolClientSessionPropApplicationDescription, defaultConverter},

	/* Service Properties */
	config.ServicePropertyVPNName:                           {ccsmp.SolClientSessionPropVpnName, defaultConverter},
	config.ServicePropertyGenerateSenderID:                  {ccsmp.SolClientSessionPropGenerateSenderID, booleanConverter},
	config.ServicePropertyGenerateSendTimestamps:            {ccsmp.SolClientSessionPropGenerateSendTimestamps, booleanConverter},
	config.ServicePropertyGenerateReceiveTimestamps:         {ccsmp.SolClientSessionPropGenerateRcvTimestamps, booleanConverter},
	config.ServicePropertyReceiverDirectSubscriptionReapply: {ccsmp.SolClientSessionPropReapplySubscriptions, booleanConverter},

	/* Transport Layer Properties */
	config.TransportLayerPropertyHost:                             {ccsmp.SolClientSessionPropHost, defaultConverter},
	config.TransportLayerPropertyConnectionAttemptsTimeout:        {ccsmp.SolClientSessionPropConnectTimeoutMs, durationConverter},
	config.TransportLayerPropertyConnectionRetries:                {ccsmp.SolClientSessionPropConnectRetries, defaultConverter},
	config.TransportLayerPropertyConnectionRetriesPerHost:         {ccsmp.SolClientSessionPropConnectRetriesPerHost, defaultConverter},
	config.TransportLayerPropertyReconnectionAttempts:             {ccsmp.SolClientSessionPropReconnectRetries, defaultConverter},
	config.TransportLayerPropertyReconnectionAttemptsWaitInterval: {ccsmp.SolClientSessionPropReconnectRetryWaitMs, durationConverter},
	config.TransportLayerPropertyKeepAliveInterval:                {ccsmp.SolClientSessionPropKeepAliveIntMs, durationConverter},
	config.TransportLayerPropertyKeepAliveWithoutResponseLimit:    {ccsmp.SolClientSessionPropKeepAliveLimit, defaultConverter},
	config.TransportLayerPropertySocketOutputBufferSize:           {ccsmp.SolClientSessionPropSocketRcvBufSize, defaultConverter},
	config.TransportLayerPropertySocketInputBufferSize:            {ccsmp.SolClientSessionPropSocketSendBufSize, defaultConverter},
	config.TransportLayerPropertySocketTCPOptionNoDelay:           {ccsmp.SolClientSessionPropTCPNodelay, booleanConverter},
	config.TransportLayerPropertyCompressionLevel:                 {ccsmp.SolClientSessionPropCompressionLevel, defaultConverter},

	/* Transport Layer Security Property */
	config.TransportLayerSecurityPropertyCertValidated:          {ccsmp.SolClientSessionPropSslValidateCertificate, booleanConverter},
	config.TransportLayerSecurityPropertyCertRejectExpired:      {ccsmp.SolClientSessionPropSslValidateCertificateDate, booleanConverter},
	config.TransportLayerSecurityPropertyCertValidateServername: {ccsmp.SolClientSessionPropSslValidateCertificateHost, booleanConverter},
	config.TransportLayerSecurityPropertyExcludedProtocols:      {ccsmp.SolClientSessionPropSslExcludedProtocols, defaultConverter},
	config.TransportLayerSecurityPropertyProtocolDowngradeTo:    {ccsmp.SolClientSessionPropSslConnectionDowngradeTo, defaultConverter},
	config.TransportLayerSecurityPropertyCipherSuites:           {ccsmp.SolClientSessionPropSslCipherSuites, defaultConverter},
	config.TransportLayerSecurityPropertyTrustStorePath:         {ccsmp.SolClientSessionPropSslTrustStoreDir, defaultConverter},
	config.TransportLayerSecurityPropertyTrustedCommonNameList:  {ccsmp.SolClientSessionPropSslTrustedCommonNameList, defaultConverter},
}

// propertyConverter is used to convert the interface{} stored in a map to a string used by CCSMP
type propertyConverter func(value interface{}) string

type property struct {
	solClientPropertyName string
	converter             propertyConverter
}

func defaultConverter(value interface{}) string {
	return fmt.Sprintf("%v", value)
}

func booleanConverter(value interface{}) string {
	switch converted := value.(type) {
	case bool:
		if converted {
			return ccsmp.SolClientPropEnableVal
		}
	case string:
		if converted == "true" {
			return ccsmp.SolClientPropEnableVal
		}
	// handle case of the user inputting 0 for disabled
	case int:
		if converted == 0 {
			return ccsmp.SolClientPropDisableVal
		}
		return ccsmp.SolClientPropEnableVal
	// handle values loaded from a config file
	case float64:
		if converted == 0 {
			return ccsmp.SolClientPropDisableVal
		}
		return ccsmp.SolClientPropEnableVal
	}
	return ccsmp.SolClientPropDisableVal
}

// Durations should be converted in certain cases to milliseconds
func durationConverter(value interface{}) string {
	switch converted := value.(type) {
	case time.Duration:
		return fmt.Sprintf("%d", converted/time.Millisecond)
	}
	return fmt.Sprintf("%v", value)
}
