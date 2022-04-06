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

package ccsmp

/*
// specific flags for darwin static builds in C
#cgo CFLAGS: -I${SRCDIR}/../../lib/darwin/include
#cgo LDFLAGS: -L/usr/local/opt/openssl@1.1/lib ${SRCDIR}/../../lib/darwin/lib/libsolclient.a -lssl -lcrypto -framework Kerberos
*/
import "C"
