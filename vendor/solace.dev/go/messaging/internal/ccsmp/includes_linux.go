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
// specific flags for linux static builds in C
#cgo CFLAGS: -I${SRCDIR}/../../lib/linux/include
#cgo LDFLAGS: ${SRCDIR}/../../lib/linux/lib/libsolclient.a -lm -ldl -lpthread -lrt

#include <dlfcn.h>
*/
import "C"

// Because of some strange behaviour of CCSMP, libdl is not actually required as far as the compiler is concerned.
// As a result, we must write a function that allows us to link against the dynamic libdl system library. This
// function should NOT be used as it does nothing and may cause issues.
func funcToLinkAgainstLibdl() {
	C.dlerror()
}
