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

// Package solace contains the main type definitions for the various messaging services.
// You can use MessagingServiceBuilder to create a client-based messaging service.
// If you want to  use secure socket layer (SSL) endpoints, OpenSSL 1.1.1 must installed on the systems
// that run your client applications. Client applications secure connections to an event broker (or broker) using
// SSL endpoints. For example on PubSub+ software event brokers, you can use
// SMF TLS/SSL (default port of 55443) and Web Transport TLS/SSL connectivity (default port 1443)
// for messaging. The ports that are utilized depends on the configuration broker.
//
// For an overview of TLS/SSL Encryption, see  TLS/SSL Encryption Overview in the Solace documentation
// at https://docs.solace.com/Overviews/TLS-SSL-Message-Encryption-Overview.htm.
//
// MessageServiceBuilder is retrieved through
// the messaging package as follows.
//  package main
//
//  import solace.dev/go/messaging
//  import solace.dev/go/messaging/pkg/solace
//
//  func main() {
//  	var messagingServiceBuilder solace.MessagingServiceBuilder
//  	messagingServiceBuilder = messaging.NewMessagingServiceBuilder()
//  	messagingService, err := messagingServiceBuilder.Build()
//  	...
//  }
//
// Before the MessagingService is created, global properties can be set by environment variable. The
// following environment variables are recognized and handled during API initialization:
//  - SOLCLIENT_GLOBAL_PROP_GSS_KRB_LIB: GSS (Kerberos) library name. If not set the default value is OS specific
//   - Linux/MacOS: libgssapi_krb5.so.2
//   - Windows: secur32.dll
//
//  - SOLCLIENT_GLOBAL_PROP_SSL_LIB: TLS Protocol library name. If not set the default value is OS specific:
//   - Linux: libssl.so
//   - MacOS: libssl.dylib
//   - Windows: libssl-1_1.dll
//
//  - SOLCLIENT_GLOBAL_PROP_CRYPTO_LIB: TLS Cryptography library name.  If not set the default value is OS specific:
//   - Linux: libcrypto.so
//   - MacOS: libcrypto.dylib
//   - Windows: libcrypto-1_1.dll-
//
//  - GLOBAL_GSS_KRB_LIB: Alternate name for SOLCLIENT_GLOBAL_PROP_GSS_KRB_LIB
//  - GLOBAL_SSL_LIB: Alternate name for SOLCLIENT_GLOBAL_PROP_SSL_LIB
//  - GLOBAL_CRYPTO_LIB: Alternate name for SOLCLIENT_GLOBAL_PROP_CRYPTO_LIB
package solace
