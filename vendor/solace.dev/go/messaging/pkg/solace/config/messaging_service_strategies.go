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
	"time"
)

// AuthenticationStrategy represents an authentication strategy.
type AuthenticationStrategy struct {
	config ServicePropertyMap
}

func (strategy AuthenticationStrategy) String() string {
	return strategy.config.String()
}

// ToProperties gets the configuration of the authentication strategy in the form of a ServicePropertyMap.
func (strategy AuthenticationStrategy) ToProperties() ServicePropertyMap {
	// return empty map if config is nil
	if strategy.config == nil {
		return make(ServicePropertyMap)
	}
	// duplicate config
	return strategy.config.GetConfiguration()
}

// BasicUserNamePasswordAuthentication gets a basic authentication strategy with the provided credentials.
func BasicUserNamePasswordAuthentication(username, password string) AuthenticationStrategy {
	authenticationStrategy := AuthenticationStrategy{make(ServicePropertyMap)}
	authenticationStrategy.config[AuthenticationPropertySchemeBasicUserName] = username
	authenticationStrategy.config[AuthenticationPropertySchemeBasicPassword] = password
	authenticationStrategy.config[AuthenticationPropertyScheme] = AuthenticationSchemeBasic
	return authenticationStrategy
}

// ClientCertificateAuthentication gets a client certificate-based authentication strategy using
// the provided certificate file, key file, and key password if required. If no keyFile or keyPassword is
// required, use an empty string for the  keyFile and keyPassword arguments.
func ClientCertificateAuthentication(certificateFile, keyFile, keyPassword string) AuthenticationStrategy {
	authenticationStrategy := AuthenticationStrategy{make(ServicePropertyMap)}
	authenticationStrategy.config[AuthenticationPropertyScheme] = AuthenticationSchemeClientCertificate
	authenticationStrategy.config[AuthenticationPropertySchemeSSLClientCertFile] = certificateFile
	if keyFile != "" {
		authenticationStrategy.config[AuthenticationPropertySchemeSSLClientPrivateKeyFile] = keyFile
	}
	if keyPassword != "" {
		authenticationStrategy.config[AuthenticationPropertySchemeClientCertPrivateKeyFilePassword] = keyPassword
	}
	return authenticationStrategy
}

// OAuth2Authentication creates an OAuth2-based authentication strategy using the specified tokens.
// At least one of accessToken or OIDC idToken must be provided. Optionally, issuerIdentifier
// can be provided. If any of the parameters is not required, an empty string can be passed.
func OAuth2Authentication(accessToken, oidcIDToken, issuerIdentifier string) AuthenticationStrategy {
	authenticationStrategy := AuthenticationStrategy{make(ServicePropertyMap)}
	authenticationStrategy.config[AuthenticationPropertyScheme] = AuthenticationSchemeOAuth2
	if len(accessToken) > 0 {
		authenticationStrategy.config[AuthenticationPropertySchemeOAuth2AccessToken] = accessToken
	}
	// Only set the id token or issuer identifier if they are nonempty.
	if len(oidcIDToken) > 0 {
		authenticationStrategy.config[AuthenticationPropertySchemeOAuth2OIDCIDToken] = oidcIDToken
	}
	if len(issuerIdentifier) > 0 {
		authenticationStrategy.config[AuthenticationPropertySchemeOAuth2IssuerIdentifier] = issuerIdentifier
	}
	return authenticationStrategy
}

// KerberosAuthentication creates a Kerberos-based authentication strategy. Optionally,
// a Kerberos service name can be provided. If the service name is not required, an empty
// string can be passed to the serviceName argument.
//
// To implement Kerberos authentication for clients connecting to a broker, the following
// configuration is required the broker:
//  - A Kerberos Keytab must be loaded on the  broker. See "Event Broker File Management" for more information
//    on the Solace documentation website.
//  - Kerberos authentication must be configured and enabled for any Message VPNs that
//    Kerberos-authenticated clients will connect to.
//  - Optional: On an appliance, a Kerberos Service Principal Name (SPN) can be assigned to the IP address
//    for the message backbone VRF Kerberos‑authenticated clients will use.
// Further reference can be found at
// 	https://docs.solace.com/Configuring-and-Managing/Configuring-Client-Authentication.htm#Config-Kerberos
func KerberosAuthentication(serviceName string) AuthenticationStrategy {
	authenticationStrategy := AuthenticationStrategy{make(ServicePropertyMap)}
	authenticationStrategy.config[AuthenticationPropertyScheme] = AuthenticationSchemeKerberos
	if len(serviceName) > 0 {
		authenticationStrategy.config[AuthenticationPropertySchemeKerberosInstanceName] = serviceName
	}
	return authenticationStrategy
}

/* Retry Strategy */

// RetryStrategy represents the strategy to use when reconnecting.
type RetryStrategy struct {
	retries       int
	retryInterval time.Duration
}

// GetRetries returns the number of retries configured with the retry strategy.
func (strategy RetryStrategy) GetRetries() int {
	return strategy.retries
}

// GetRetryInterval returns the interval between (re)connection retry attempts.
func (strategy RetryStrategy) GetRetryInterval() time.Duration {
	return strategy.retryInterval
}

// RetryStrategyForeverRetry creates a retry strategy that runs forever
// with automatic retries and a retry interval of 3000 milliseconds.
func RetryStrategyForeverRetry() RetryStrategy {
	return RetryStrategy{-1, 3000 * time.Millisecond}
}

// RetryStrategyForeverRetryWithInterval creates a retry strategy that runs forever
// with automatic retries and a retry interval of the specified duration.
func RetryStrategyForeverRetryWithInterval(retryInterval time.Duration) RetryStrategy {
	return RetryStrategy{-1, retryInterval}
}

// RetryStrategyNeverRetry creates an instance of configuration for disabled automatic retries.
func RetryStrategyNeverRetry() RetryStrategy {
	return RetryStrategy{0, 3000 * time.Millisecond}
}

// RetryStrategyParameterizedRetry creates an instance of configuration for free configurability.
// retries must be greater than or equal to zero.
func RetryStrategyParameterizedRetry(retries uint, retryInterval time.Duration) RetryStrategy {
	return RetryStrategy{int(retries), retryInterval}
}

/* Transport Security Strategy */

// TransportSecurityStrategy represents the strategy to use when connecting to a broker
// via a secure port.
type TransportSecurityStrategy struct {
	config ServicePropertyMap
}

// TransportSecurityProtocol represents the various security protocols available to the API.
type TransportSecurityProtocol string

const (
	// TransportSecurityProtocolSSLv3 represents SSLv3.
	TransportSecurityProtocolSSLv3 TransportSecurityProtocol = "SSLv3"
	// TransportSecurityProtocolTLSv1 represents TLSv1.
	TransportSecurityProtocolTLSv1 TransportSecurityProtocol = "TLSv1"
	// TransportSecurityProtocolTLSv1_1 represents TLSv1.1.
	TransportSecurityProtocolTLSv1_1 TransportSecurityProtocol = "TLSv1.1"
	// TransportSecurityProtocolTLSv1_2 represents TLSv1.2.
	TransportSecurityProtocolTLSv1_2 TransportSecurityProtocol = "TLSv1.2"
)

// NewTransportSecurityStrategy creates a transport security strategy with default configuration.
// Properties can be overwritten by calling the various configuration functions on TransportSecurityStrategy.
func NewTransportSecurityStrategy() TransportSecurityStrategy {
	tss := TransportSecurityStrategy{
		config: make(ServicePropertyMap),
	}
	return tss
}

// Downgradable configures TLS so that the session connection is downgraded to plain-text after client authentication.
// WARNING: Downgrading SSL to plain-text after client authentication exposes a client and the data being sent
// to higher security risks.
func (tss TransportSecurityStrategy) Downgradable() TransportSecurityStrategy {
	tss.config[TransportLayerSecurityPropertyProtocolDowngradeTo] = DowngradeToPlaintext
	return tss
}

// WithExcludedProtocols specifies the list of SSL or TLS protocols to not use.
func (tss TransportSecurityStrategy) WithExcludedProtocols(protocols ...TransportSecurityProtocol) TransportSecurityStrategy {
	protocolsCS := ""
	for _, protocol := range protocols {
		if protocolsCS != "" {
			protocolsCS += ","
		}
		protocolsCS += string(protocol)
	}
	tss.config[TransportLayerSecurityPropertyExcludedProtocols] = protocolsCS
	return tss
}

// WithoutCertificateValidation configures TLS to not validate the server certificate
// configured on the remote broker.
// WARNING: Disabling certificate validation exposes clients and data being sent to higher security risks.
func (tss TransportSecurityStrategy) WithoutCertificateValidation() TransportSecurityStrategy {
	tss.config[TransportLayerSecurityPropertyCertValidated] = false
	return tss
}

// WithCertificateValidation configures TLS validation on certificates. By default, validation is performed.
// WARNING: Disabling certificate validation exposes clients and data being sent to higher security risks.
//
// ignoreExpiration: when set to true, expired certificates are accepted.
//
// validateServerName: When set to true, certificates without the matching host are not accepted.
//
// trustStoreFilePath: The location of the trust store files. If an empty string is passed, no file path will be set.
//
// trustedCommonNameList: A comma-separated list of acceptable common names for matching with server certificates. An empty string will match no names.
func (tss TransportSecurityStrategy) WithCertificateValidation(
	ignoreExpiration bool,
	validateServerName bool,
	trustStoreFilePath string,
	trustedCommonNameList string,
) TransportSecurityStrategy {
	tss.config[TransportLayerSecurityPropertyCertValidated] = true
	tss.config[TransportLayerSecurityPropertyCertRejectExpired] = !ignoreExpiration
	tss.config[TransportLayerSecurityPropertyCertValidateServername] = validateServerName
	if trustStoreFilePath != "" {
		tss.config[TransportLayerSecurityPropertyTrustStorePath] = trustStoreFilePath
	}
	if trustedCommonNameList != "" {
		tss.config[TransportLayerSecurityPropertyTrustedCommonNameList] = trustedCommonNameList
	}
	return tss
}

// WithCipherSuites configures cipher suites to use. The cipher suites value is a comma-separated
// list of cipher suites and must be from the following table:
//	+-----------------+-------------------------------+--------------------+
//	| 'AES256-SHA'    | 'ECDHE-RSA-AES256-SHA'        | 'AES256-GCM-SHA384'|
//	+-----------------+-------------------------------+--------------------+
//	| 'AES256-SHA256' | 'ECDHE-RSA-AES256-GCM-SHA384' | 'AES128-SHA256'    |
//	+-----------------+-------------------------------+--------------------+
//	| 'DES-CBC3-SHA'  | 'ECDHE-RSA-DES-CBC3-SHA'      |                    |
//	+-----------------+-------------------------------+--------------------+
//	| 'RC4-SHA'       | 'ECDHE-RSA-AES256-SHA384'     | 'AES128            |
//	+-----------------+-------------------------------+--------------------+
//	| 'ECDHE-RSA-AES128-SHA256'                       | 'AES128-GCM-SHA256'|
//	+-----------------+-------------------------------+--------------------+
//	| 'RC4-MD5'       | 'ECDHE-RSA-AES128-GCM-SHA256' |                    |
//	+----------------------------------------+-----------------------------+
//	| 'TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384'| 'ECDHE-RSA-AES128-SHA'      |
//	+----------------------------------------+-----------------------------+
//	| 'TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384'|                             |
//	+----------------------------------------+-----------------------------+
//	| 'TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA'   |                             |
//	+----------------------------------------+-----------------------------+
//	| 'TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA'  |                             |
//	+----------------------------------------+-----------------------------+
//	| 'TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256'|                             |
//	+----------------------------------------+-----------------------------+
//	| 'TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA'   |                             |
//	+----------------------------------------+-----------------------------+
//	| 'TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256'|                             |
//	+-----------------------------------+----------------------------------+
//	| 'TLS_RSA_WITH_AES_128_GCM_SHA256' |                                  |
//	+-----------------------------------+----------------------------------+
//	| 'TLS_RSA_WITH_AES_128_CBC_SHA256' |'TLS_RSA_WITH_AES_256_GCM_SHA384' |
//	+-----------------------------------+----------------------------------+
//	| 'TLS_RSA_WITH_AES_256_CBC_SHA256' | 'TLS_RSA_WITH_AES_256_CBC_SHA'   |
//	+-----------------------------------+----------------------------------+
//	| 'SSL_RSA_WITH_3DES_EDE_CBC_SHA    | 'TLS_RSA_WITH_AES_128_CBC_SHA'   |
//	+-----------------------------------+----------------------------------+
//	| 'SSL_RSA_WITH_RC4_128_SHA'        | 'SSL_RSA_WITH_RC4_128_MD5'       |
//	+-----------------------------------+----------------------------------+
func (tss TransportSecurityStrategy) WithCipherSuites(cipherSuiteList string) TransportSecurityStrategy {
	tss.config[TransportLayerSecurityPropertyCipherSuites] = cipherSuiteList
	return tss
}

// ToProperties returns the configuration in the form of a ServicePropertyMap.
func (tss TransportSecurityStrategy) ToProperties() ServicePropertyMap {
	if tss.config == nil {
		return make(ServicePropertyMap)
	}
	return tss.config.GetConfiguration()
}
