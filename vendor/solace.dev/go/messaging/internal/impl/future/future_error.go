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

// Package future is defined below
package future

import "sync"

// FutureError interface
type FutureError interface {
	// a call to Get will block until the future is resolved
	// may return immediately if already resolved
	Get() error
	Complete(err error)
}

// NewFutureError function
func NewFutureError() FutureError {
	mutex := &sync.Mutex{}
	cond := sync.NewCond(mutex)
	return &futureError{
		resolved: false,
		result:   nil,
		mutex:    mutex,
		cond:     cond,
	}
}

type futureError struct {
	resolved bool
	result   error
	mutex    *sync.Mutex
	cond     *sync.Cond
}

func (future *futureError) Complete(err error) {
	future.mutex.Lock()
	defer future.mutex.Unlock()
	future.result = err
	future.resolved = true
	future.cond.Broadcast()
}

func (future *futureError) Get() error {
	future.mutex.Lock()
	defer future.mutex.Unlock()
	for !future.resolved {
		future.cond.Wait()
	}
	return future.result
}
