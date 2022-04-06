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

package solace

import "solace.dev/go/messaging/pkg/solace/subcode"

// A solaceError struct used as a basis for all wrapping errors.
type solaceError struct {
	message string
	wrapped error
}

// setErrorInfo will set the information on the error.
func (err *solaceError) setErrorInfo(message string, wrapped error) {
	err.message = message
	err.wrapped = wrapped
}

// Error is an error returned from the API.
type Error interface {
	error
	setErrorInfo(message string, wrapped error)
}

// Error returns the error message.
func (err *solaceError) Error() string {
	return err.message
}

// Unwrap returns the wrapped error.
func (err *solaceError) Unwrap() error {
	return err.wrapped
}

// NativeError is a struct that stores the error message and subcode for the error message.
type NativeError struct {
	message string
	subcode subcode.Code
}

// Error returns the error message from solace.NativeError.
func (err *NativeError) Error() string {
	return err.message
}

// SubCode returns the subcode associated with the specified error, otherwise SubCodeNone if no subcode is relevant.
func (err *NativeError) SubCode() subcode.Code {
	return err.subcode
}

// NewNativeError returns a new solace.NativeError with the given message and subcode when applicable.
func NewNativeError(message string, subcode subcode.Code) *NativeError {
	return &NativeError{
		message: message,
		subcode: subcode,
	}
}

// AuthenticationError indicates an authentication related error occurred when connecting to a remote event broker.
// The pointer type *AuthenticationError is returned.
type AuthenticationError struct {
	solaceError
}

// IllegalStateError indicates an invalid state occurred when performing an action.
// The pointer type *IllegalStateError is returned.
type IllegalStateError struct {
	solaceError
}

// IllegalArgumentError indicates an invalid argument was passed to a function.
// The pointer type *IllegalArgumentError is returned.
type IllegalArgumentError struct {
	solaceError
}

// InvalidConfigurationError indicates that a specified configuration is invalid.
// These errors are returned by the Build functions of a builder.
// The pointer type *InvalidConfigurationError is returned.
type InvalidConfigurationError struct {
	solaceError
}

// PublisherOverflowError indicates when publishing has stopped due to
// internal buffer limits.
// The pointer type *PublisherOverflowError is returned.
type PublisherOverflowError struct {
	solaceError
}

// IncompleteMessageDeliveryError indicates that some messages were not delivered.
// The pointer type *IncompleteMessageDeliveryError is returned.
type IncompleteMessageDeliveryError struct {
	solaceError
}

// ServiceUnreachableError indicates that a remote service connection could not be established.
// The pointer type *ServiceUnreachableError is returned.
type ServiceUnreachableError struct {
	solaceError
}

// TimeoutError indicates that a timeout error occurred.
// The pointer type *TimeoutError is returned.
type TimeoutError struct {
	solaceError
}

// MessageReplayError indicates
type MessageReplayError struct {
	solaceError
}

// NewError returns a new Solace error with the specified message and wrapped error.
func NewError(err Error, message string, wrapped error) Error {
	err.setErrorInfo(message, wrapped)
	return err
}
