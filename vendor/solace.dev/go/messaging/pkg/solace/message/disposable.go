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

package message

// Disposable implies that the implementing data structure needs to free
// the underlying resources. This is done with the Dispose function. It is optional to use
// this function to free the underlying resources because any implementing data structure must attempt
// to free the underlying resources using a finalizer. For
// performance purposes, we recommend to explicitly free the underlying resources
// before garbage collection runs.
type Disposable interface {
	// Dispose frees all the underlying resources of the Disposable instance.
	// Dispose is idempotent, and removes any redundant finalizers on the
	// instance and substantially improves garbage-collection performance.
	// This function is thread-safe, and subsequent calls to Dispose
	// block and wait for the first call to complete. Additional calls
	// return immediately. The instance is considered unusable after Dispose
	// has been called.
	Dispose()

	// IsDisposed checks if the Disposable instance has been disposed by
	// a call to Dispose. IsDisposeed returns true if Dispose has been called,
	// otherwise false if it is still usable. Dispose may or may not have returned.
	// The instance is considered unusable if IsDisposed returns true.
	IsDisposed() bool
}
