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

// The various authentication schemes available for use when configuring a MessagingService's authentication.
const (
	// AuthenticationSchemeBasic configures basic authentication.
	AuthenticationSchemeBasic = "AUTHENTICATION_SCHEME_BASIC"
	// AuthenticationSchemeClientCertificate configures client certificate authentication.
	AuthenticationSchemeClientCertificate = "AUTHENTICATION_SCHEME_CLIENT_CERTIFICATE"
	// AuthenticationSchemeKerberos configures Generic Security Service for Kerberos authentication.
	AuthenticationSchemeKerberos = "AUTHENTICATION_SCHEME_GSS_KRB"
	// AuthenticationSchemeOAuth2 configures OAuth2 authentication.
	AuthenticationSchemeOAuth2 = "AUTHENTICATION_SCHEME_OAUTH2"
)

// DowngradeToPlaintext can be used with TransportLayerSecurityPropertyProtocolDowngradeTo to configure
// downgrading the TLS connection to the broker to plain-text after authenticating over TLS.
const DowngradeToPlaintext = "PLAIN_TEXT"
