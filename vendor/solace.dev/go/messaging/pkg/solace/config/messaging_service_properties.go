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

import "fmt"

// ServicePropertiesConfigurationProvider represents a generic configuration provider that can be used
// to provide configurations to MessagingServiceBuilders.
type ServicePropertiesConfigurationProvider interface {
	// GetConfiguration retrieves the configuration provided by the provider
	// in the form of a ServicePropertyMap.
	GetConfiguration() ServicePropertyMap
}

// ServiceProperty is a key for a property.
type ServiceProperty string

// ServicePropertyMap is a map of SolaceServicePropery keys to values.
type ServicePropertyMap map[ServiceProperty]interface{}

// GetConfiguration returns a copy of the servicePropertyMap.
func (servicePropertyMap ServicePropertyMap) GetConfiguration() ServicePropertyMap {
	ret := make(ServicePropertyMap)
	for key, value := range servicePropertyMap {
		ret[key] = value
	}
	return ret
}

// MarshalJSON implements the json.Marshaler interface.
func (servicePropertyMap ServicePropertyMap) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	for k, v := range servicePropertyMap {
		m[string(k)] = v
	}
	return nestJSON(m)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (servicePropertyMap ServicePropertyMap) UnmarshalJSON(b []byte) error {
	m, err := flattenJSON(b)
	if err != nil {
		return err
	}
	for key, val := range m {
		servicePropertyMap[ServiceProperty(key)] = val
	}
	return nil
}

const obfuscation = "******"

var obfuscatedProperties = []ServiceProperty{
	AuthenticationPropertySchemeBasicPassword,
	AuthenticationPropertySchemeClientCertPrivateKeyFilePassword,
	AuthenticationPropertySchemeOAuth2AccessToken,
	AuthenticationPropertySchemeOAuth2OIDCIDToken,
}

// String implements the fmt.Stringer interface.
func (servicePropertyMap ServicePropertyMap) String() string {
	toPrint := make(map[string]interface{})
	for key, value := range servicePropertyMap {
		for _, property := range obfuscatedProperties {
			if key == property {
				value = obfuscation
				break
			}
		}
		toPrint[string(key)] = value
	}
	return fmt.Sprint(toPrint)
}

const (
	/* AuthenticaionProperties */

	// AuthenticationPropertyScheme defines the keys for the authentication scheme type.
	// Possible values for configuration of authentication scheme are defined in
	// the AuthenticationScheme type, which has the following possible strings,
	// accessible by constants
	//  config.AuthenticationSchemeBasic | "AUTHENTICATION_SCHEME_BASIC"
	//  config.AuthenticationSchemeClientCertificate | "AUTHENTICATION_SCHEME_CLIENT_CERTIFICATE"
	//  config.AuthenticationSchemeKerberos | "AUTHENTICATION_SCHEME_GSS_KRB"
	//  config.AuthenticationSchemeOAuth2 | "AUTHENTICATION_SCHEME_OAUTH2"
	AuthenticationPropertyScheme ServiceProperty = "solace.messaging.authentication.scheme"

	// AuthenticationPropertySchemeBasicUserName specifies the username for basic authentication.
	AuthenticationPropertySchemeBasicUserName ServiceProperty = "solace.messaging.authentication.basic.username"

	// AuthenticationPropertySchemeBasicPassword specifies password for basic authentication.
	AuthenticationPropertySchemeBasicPassword ServiceProperty = "solace.messaging.authentication.basic.password"

	// AuthenticationPropertySchemeSSLClientCertFile specifies the client certificate file used for Secure Socket Layer (SSL).
	AuthenticationPropertySchemeSSLClientCertFile ServiceProperty = "solace.messaging.authentication.client-cert.file"

	// AuthenticationPropertySchemeSSLClientPrivateKeyFile specifies the client private key file.
	AuthenticationPropertySchemeSSLClientPrivateKeyFile ServiceProperty = "solace.messaging.authentication.client-cert.private-key-file"

	// AuthenticationPropertySchemeClientCertPrivateKeyFilePassword specifies the private key password for client certificate authentication.
	AuthenticationPropertySchemeClientCertPrivateKeyFilePassword ServiceProperty = "solace.messaging.authentication.client-cert.private-key-password"

	// AuthenticationPropertySchemeClientCertUserName specifies the username to use when connecting with client certificate authentication.
	AuthenticationPropertySchemeClientCertUserName ServiceProperty = "solace.messaging.authentication.client-cert.username"

	// AuthenticationPropertySchemeKerberosInstanceName specifies the first part of Kerberos Service Principal Name (SPN) of the
	// form ServiceName/Hostname\@REALM (for Windows) or Host Based Service of the form
	// ServiceName\@Hostname (for Linux and SunOS).
	AuthenticationPropertySchemeKerberosInstanceName ServiceProperty = "solace.messaging.authentication.kerberos.instance-name"

	// AuthenticationPropertySchemeKerberosUserName allows configuration of the client username to use when connecting to the broker.
	// It is only used if Kerberos is the chosen authentication strategy.
	// This property is ignored by default and only needed if the 'allow-api-provided-username' is enabled in the
	// configuration of the message-vpn in the broker. This property is not recommended for use.
	AuthenticationPropertySchemeKerberosUserName ServiceProperty = "solace.messaging.authentication.kerberos.username"

	// AuthenticationPropertySchemeOAuth2AccessToken specifies an access token for OAuth 2.0 token-based authentication.
	AuthenticationPropertySchemeOAuth2AccessToken ServiceProperty = "solace.messaging.authentication.oauth2.access-token"

	// AuthenticationPropertySchemeOAuth2IssuerIdentifier defines an optional issuer identifier for OAuth 2.0 token-based authentication.
	AuthenticationPropertySchemeOAuth2IssuerIdentifier ServiceProperty = "solace.messaging.authentication.oauth2.issuer-identifier"

	// AuthenticationPropertySchemeOAuth2OIDCIDToken specifies the ID token for Open ID Connect token-based authentication.
	AuthenticationPropertySchemeOAuth2OIDCIDToken ServiceProperty = "solace.messaging.authentication.oauth2.oidc-id-token"

	/* ClientProperties */

	// ClientPropertyName sets the client name used when connecting to the broker. This field is set on:
	//   MessagingServiceBuilder::BuildWithApplicationID
	ClientPropertyName ServiceProperty = "solace.messaging.client.name"

	// ClientPropertyApplicationDescription can be set to set the application description viewable on a broker.
	ClientPropertyApplicationDescription ServiceProperty = "solace.messaging.client.application-description"

	/* Service Properties */

	// ServicePropertyVPNName name of the Message VPN to attempt to join when connecting to a broker. Default
	// value is "" when this property is not specified. When using default values, the client attempts to join
	// the default Message VPN for the client. This parameter is only valid for sessions operating
	// in client mode. If specified, it must be a maximum of 32 bytes in length when encoded as
	// UTF-8.
	ServicePropertyVPNName ServiceProperty = "solace.messaging.service.vpn-name"

	// ServicePropertyGenerateSenderID specifies whether the client name should be included in the SenderID
	// message-header parameter. Setting this property is optional.
	ServicePropertyGenerateSenderID ServiceProperty = "solace.messaging.service.generate-sender-id"

	// ServicePropertyGenerateSendTimestamps specifies whether timestamps should be generated for outbound messages.
	ServicePropertyGenerateSendTimestamps ServiceProperty = "solace.messaging.service.generate-send-timestamps"

	// ServicePropertyGenerateReceiveTimestamps specifies whether timestamps should be generated on inbound messages.
	ServicePropertyGenerateReceiveTimestamps ServiceProperty = "solace.messaging.service.generate-receive-timestamps"

	// ServicePropertyReceiverDirectSubscriptionReapply enables reapplying a subscription when a session reconnection occurs
	// for a direct message receiver the value type is boolean.
	ServicePropertyReceiverDirectSubscriptionReapply ServiceProperty = "solace.messaging.service.receivers.direct.subscription.reapply"

	/* TransportLayerProperties */

	// TransportLayerPropertyHost is IPv4 or IPv6 address or host name of the broker to which to connect.
	// Multiple entries are permitted when each address is separated by a comma.
	// The entry for the HOST property should provide a protocol, host and port.
	TransportLayerPropertyHost ServiceProperty = "solace.messaging.transport.host"

	// TransportLayerPropertyConnectionAttemptsTimeout is the timeout period for a connect operation to a given host.
	TransportLayerPropertyConnectionAttemptsTimeout ServiceProperty = "solace.messaging.transport.connection-attempts-timeout"

	// TransportLayerPropertyConnectionRetries is how many times to try connecting to a broker during connection setup.
	TransportLayerPropertyConnectionRetries ServiceProperty = "solace.messaging.transport.connection-retries"

	// TransportLayerPropertyConnectionRetriesPerHost defines how many connection or reconnection attempts are made to a
	// single host before moving to the next host in the list,  when using a host list.
	TransportLayerPropertyConnectionRetriesPerHost ServiceProperty = "solace.messaging.transport.connection.retries-per-host"

	// TransportLayerPropertyReconnectionAttempts is the number reconnection attempts to the broker (or list of brokers) after
	// a connected MessagingService goes down.
	// Zero means no automatic reconnection attempts, while a -1 means attempt to reconnect forever. The default valid range is greather than or equal to -1.
	// When using a host list, each time the API works through the host list without establishing a connection is considered a
	// reconnection retry. Each reconnection retry begins with the first host listed. After each unsuccessful attempt to reconnect
	// to a host, the API waits for the amount of time set for TransportLayerPropertyReconnectionAttemptsWaitInterval before attempting another
	// connection to a broker. The number of times attempted to connect to one broker before moving on to the
	// next listed host is determined by the value set for TransportLayerPropertyConnectionRetriesPerHost.
	TransportLayerPropertyReconnectionAttempts ServiceProperty = "solace.messaging.transport.reconnection-attempts"

	// TransportLayerPropertyReconnectionAttemptsWaitInterval sets how much time (in milliseconds) to wait between each connection or reconnection attempt to the configured host.
	// If a connection or reconnection attempt to the configured host (which may be a list) is not successful, the API waits for
	// the amount of time set for ReconnectionAttemptsWaitInterval, and then makes another connection or reconnection attempt.
	// The valid range is greater than or equal to zero.
	TransportLayerPropertyReconnectionAttemptsWaitInterval ServiceProperty = "solace.messaging.transport.reconnection-attempts-wait-interval"

	// TransportLayerPropertyKeepAliveInterval is the amount of time (in milliseconds) to wait between sending out Keep-Alive messages.
	TransportLayerPropertyKeepAliveInterval ServiceProperty = "solace.messaging.transport.keep-alive-interval"

	// TransportLayerPropertyKeepAliveWithoutResponseLimit is the maximum number of consecutive keep-alive messages that can be sent
	// without receiving a response before the connection is closed by the API.
	TransportLayerPropertyKeepAliveWithoutResponseLimit ServiceProperty = "solace.messaging.transport.keep-alive-without-response-limit"

	// TransportLayerPropertySocketOutputBufferSize is the value for the socket send buffer size (in bytes).
	// Zero indicates to not set the value and leave the buffer size set at operating system default. The valid range is zero or
	// a value greater than or equal to 1024.
	TransportLayerPropertySocketOutputBufferSize ServiceProperty = "solace.messaging.transport.socket.output-buffer-size"

	// TransportLayerPropertySocketInputBufferSize is the value for socket receive buffer size (in bytes).
	// Zero indicates to not set the value and leave the buffer size set at operating system default. The valid range is zero or
	// a value greater than or equal to 1024.
	TransportLayerPropertySocketInputBufferSize ServiceProperty = "solace.messaging.transport.socket.input-buffer-size"

	// TransportLayerPropertySocketTCPOptionNoDelay is a boolean value to enable TCP no delay.
	TransportLayerPropertySocketTCPOptionNoDelay ServiceProperty = "solace.messaging.transport.socket.tcp-option-no-delay"

	// TransportLayerPropertyCompressionLevel enables messages to be compressed with ZLIB before transmission and decompressed on receive.
	// This property should preferably be set by MessagingServiceClientBuilder.WithCompressionLevel function.
	// The valid range is zero (off) or a value from 1 to 9, where 1 is less compression (fastest) and 9 is most compression (slowest).
	// Note: If no port is specified in  TransportLayerPropertyHost, the API  automatically connects to either the default
	// non-compressed listen port (55555) or default compressed listen port (55003) based on the specified
	// CompressionLevel. If a port is specified in TransportLayerPropertyHost, you must specify the non-compressed listen port if you are not
	// using compression (compression level 0), or the compressed listen port if using compression (compression levels 1 to 9).
	TransportLayerPropertyCompressionLevel ServiceProperty = "solace.messaging.transport.compression-level"

	/* TransportLayerSecurityProperty */

	// TransportLayerSecurityPropertyCertValidated is a boolean property that will specify if server certificates should be validated.
	// Setting this property to false exposes a client and data being sent to a higher security risk.
	TransportLayerSecurityPropertyCertValidated ServiceProperty = "solace.messaging.tls.cert-validated"

	// TransportLayerSecurityPropertyCertRejectExpired is a boolean property that specifies if server certificate's expiration date should be validated.
	// Setting this property to false exposes a client and data being sent to a higher security risk.
	TransportLayerSecurityPropertyCertRejectExpired ServiceProperty = "solace.messaging.tls.cert-reject-expired"

	// TransportLayerSecurityPropertyCertValidateServername is a boolean property that enables or disables server certificate hostname or
	// IP address validation. When enabled, the certificate received from the broker must match
	// the host used to connect.
	TransportLayerSecurityPropertyCertValidateServername ServiceProperty = "solace.messaging.tls.cert-validate-servername"

	// TransportLayerSecurityPropertyExcludedProtocols is a comma-separated list specifying SSL protocols to exclude.
	// Valid protocols are 'SSLv3', 'TLSv1', 'TLSv1.1' and 'TLSv1.2'
	TransportLayerSecurityPropertyExcludedProtocols ServiceProperty = "solace.messaging.tls.excluded-protocols"

	// TransportLayerSecurityPropertyProtocolDowngradeTo specifies a transport protocol that the SSL connection is  downgraded to after
	// client authentication. Allowed transport protocol is 'PLAIN_TEXT'.
	// May be combined with non-zero compression level to achieve compression without encryption.
	TransportLayerSecurityPropertyProtocolDowngradeTo ServiceProperty = "solace.messaging.tls.protocol-downgrade-to"

	// TransportLayerSecurityPropertyCipherSuites specifies a comma-separated list of cipher suites to use.
	// Allowed cipher suites are as follows:
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
	TransportLayerSecurityPropertyCipherSuites ServiceProperty = "solace.messaging.tls.cipher-suites"

	// TransportLayerSecurityPropertyTrustStorePath specifies the path of the directory where trusted certificates are found
	// A maximum of 64 files are allowed in the trust store directory. The maximum depth for the
	// certificate chain verification is 3.
	TransportLayerSecurityPropertyTrustStorePath ServiceProperty = "solace.messaging.tls.trust-store-path"

	// TransportLayerSecurityPropertyTrustedCommonNameList is provided for legacy installations and is not recommended as part of our best practices.
	// The API performs Subject Alternative Name (SAN) verification when the SAN is found in the server certificate,
	// which is generally the best practice for a secure connection.
	// This property specifies a comma-separated list of acceptable common names in certificate validation.
	// The number of common names specified by an application is limited to 16. Leading and trailing whitespaces are considered
	// to be part of the common names and are not ignored.
	// If the application does not provide any common names, there is no common-name verification.
	// An empty string specifies that no common-name verification is required.
	TransportLayerSecurityPropertyTrustedCommonNameList ServiceProperty = "solace.messaging.tls.trusted-common-name-list"
)
