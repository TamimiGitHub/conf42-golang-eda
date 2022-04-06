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

// Package sdt contains the types needed to work with Structured Data on a message.
// In particular, two main data types are provided, sdt.Map and sdt.Stream. They can both be
// used as normal Golang maps and slices as well as can be passed as a payload on a message.
// When retrieved, a map will contain the converted data converted as per the table on sdt.Data.
// Furthermore, data types can be converted using the various getters such as GetBool which will
// attempt to convert the specified data into a builtin.bool.
package sdt

import (
	"fmt"
	"strconv"
	"unsafe"

	"solace.dev/go/messaging/pkg/solace/resource"
)

// Map represents a map between strings and values that can be converted into structured
// data types in a message payload. Accepts any valid sdt.Data types.
type Map map[string]Data

// Stream represents a stream of data that can be converted into structured data types
// in a message payload. Accepts any valid sdt.Data types.
type Stream []Data

// Data represents valid types that can be used in sdt.Map and sdt.Stream instances.
// Valid types are: int, uint, nil, bool, uint8, uint16, uint32, uint64, int8, int16, int32,
// int64, string, rune, float32, float64, byte, []byte, sdt.WChar, resource.Destination,
// sdt.Map and sdt.Stream.
//
//	Message type mapping is as follows:
//	solClient_fieldType     Golang Type
//	SOLCLIENT_BOOL          bool
//	SOLCLIENT_UINT8         uint8
//	SOLCLIENT_INT8          int8
//	SOLCLIENT_UINT16        uint16
//	SOLCLIENT_INT16         int16
//	SOLCLIENT_UINT32        uint32
//	SOLCLIENT_INT32         int32
//	SOLCLIENT_UINT64        uint64
//	SOLCLIENT_INT64         int64
//	SOLCLIENT_WCHAR         sdt.WChar
//	SOLCLIENT_STRING        string
//	SOLCLIENT_BYTEARRAY     []byte
//	SOLCLIENT_FLOAT         float32
//	SOLCLIENT_DOUBLE        float64
//	SOLCLIENT_MAP           sdt.Map
//	SOLCLIENT_STREAM        sdt.Stream
//	SOLCLIENT_NULL          nil
//	SOLCLIENT_DESTINATION   solace.Destination (solace.Queue or solace.Topic)
//	SOLCLIENT_UNKNOWN       []byte
//
// On outbound messages, the following additional types are supported with the following mappings:
//	byte      SOLCLIENT_INT8
//	int      SOLCLIENT_INT64
//	uint      SOLCLIENT_UINT64
//	rune      SOLCLIENT_INT32
//
// Notes on mappings:
//
// - int is always be converted to int64, even when on 32-bit architectures
//
// - uint is always be converted to uint64, even when on 32-bit architectures
//
// - byte is the same as int8 and is mapped accordingly
//
// In addition to these mappings, client-side conversions may be made to retrieve data
// in the type you want. The following table is used to determine these conversions:
//
//	Type    |  bool  |  byte  |  int  |  uint  |  uint8  |  int8  |  uint16  | int16  |  uint32  |  int32  |  uint64  |  int64  |  []byte  |  float32  |  float64  |  string  |  WChar  |
//	-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
//	bool    |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	byte    |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	int     |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	uint    |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	uint8   |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	int8    |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	uint16  |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	int16   |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	uint32  |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	int32   |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	uint64  |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	int64   |   x        x        x       x         x        x         x         x         x          x         x          x          x
//	[]byte  |                                                                                                                         x                                    x
//	float32 |                                                                                                                                     x                        x
//	float64 |                                                                                                                                                    x         x
//	string  |                                                                                                                                                              x        x
//	WChar   |                                       x        x                                                                                                             x        x
//
// For each integer-type conversion, the value will be converted if it is
// within the range of the requested type, otherwise a conversion error
// will be thrown indicating that the value is outside the domain.
// String types will be converted to integers using strconv.Stoi and
// then returned if it is within the range of the requested type.
// For boolean conversions from integer types, all non-zero values map to true.
// For boolean conversion from strings, "true" and "1" map to true, "false" and
// "0" map to false, and all other strings are not converted. WChar conversion
// converts strings of length 1 into a WChar. All other strings are rejected.
//
// In addition, the following one-to-one conversions are used:
//	sdt.Map, sdt.Stream
// Lastly, Destination is mapped with these getters:
//	GetQueue, GetTopic, and GetDestination
type Data interface{}

// WChar represents the wide character type. WChar is a uint16 value
// holding UTF-16 data. On publish, this value is converted into
// SOLCLIENT_WCHAR, and on receive SOLCLIENT_WCHAR is converted into
// WChar.
type WChar uint16

// KeyNotFoundError is returned if the requested key is not found.
type KeyNotFoundError struct {
	// Key is the key that was being accessed and does not exist.
	Key string
}

func (err *KeyNotFoundError) Error() string {
	return fmt.Sprintf("could not find mapping for key %s", err.Key)
}

// OutOfBoundsError is returned if the specified index is out of bounds.
type OutOfBoundsError struct {
	Index int
}

func (err *OutOfBoundsError) Error() string {
	return fmt.Sprintf("index %d is out of bounds", err.Index)
}

// FormatConversionError is returned if there is an error converting
// stored data to a specified return type.
type FormatConversionError struct {
	// Message is the message of the error
	Message string
	// Data is the data that could not be converted.
	Data interface{}
}

// Error implements the error interface and  returns the error message as string.
func (err *FormatConversionError) Error() string {
	return err.Message
}

func defaultFormatConversionError(data interface{}, desiredType string) *FormatConversionError {
	return &FormatConversionError{
		Message: fmt.Sprintf("cannot convert %T to %s", data, desiredType),
		Data:    data,
	}
}

func nestedFormatConversionError(data interface{}, desiredType string, nested error) *FormatConversionError {
	return &FormatConversionError{
		Message: fmt.Sprintf("cannot convert %T to %s: %s", data, desiredType, nested.Error()),
		Data:    data,
	}
}

// IllegalTypeError is returned if an error occurs while encoding a stored
// type to a message when setting a payload or user property.
type IllegalTypeError struct {
	Data interface{}
}

// Error implements the error interface and  returns the error message as string.
func (err *IllegalTypeError) Error() string {
	return fmt.Sprintf("type %T is not a valid instance of sdt.Data", err.Data)
}

// GetBool retrieves the data using the specified key and attempts to convert the
// data into a boolean form, if necessary. Returns the boolean value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into boolean form.
func (sdtMap Map) GetBool(key string) (val bool, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return false, &KeyNotFoundError{key}
	}
	return getBool(elem)
}

// GetByte retrieves the data at the specified key and attempt to convert the
// data into byte form if necessary. Returns the byte value of the field,
// and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into byte form.
func (sdtMap Map) GetByte(key string) (val byte, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getUint8(elem)
}

// GetInt retrieves the data at the specified key and attempt to convert the
// data into int form if necessary. Returns the int value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into int form.
func (sdtMap Map) GetInt(key string) (val int, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getInt(elem)
}

// GetUInt retrieves the data at the specified key and attempt to convert the
// data into uint form if necessary. Returns the uint value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into uint form.
func (sdtMap Map) GetUInt(key string) (val uint, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getUint(elem)
}

// GetInt8 retrieves the data at the specified key and attempt to convert the
// data into int8 form if necessary. Returns the int8 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into int8 form.
func (sdtMap Map) GetInt8(key string) (val int8, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getInt8(elem)
}

// GetInt16 retrieves the data at the specified key and attempt to convert the
// data into int16 form if necessary. Returns the int16 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into int16 form.
func (sdtMap Map) GetInt16(key string) (val int16, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getInt16(elem)
}

// GetInt32 retrieves the data at the specified key and attempt to convert the
// data into int32 form if necessary. Returns the int32 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into int32 form.
func (sdtMap Map) GetInt32(key string) (val int32, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getInt32(elem)
}

// GetInt64 retrieves the data at the specified key and attempt to convert the
// data into int64 form if necessary. Returns the int64 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into int64 form.
func (sdtMap Map) GetInt64(key string) (val int64, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getInt64(elem)
}

// GetUInt8 retrieves the data at the specified key and attempt to convert the
// data into uint8 form if necessary. Returns the uint8 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into uint8 form.
func (sdtMap Map) GetUInt8(key string) (val uint8, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getUint8(elem)
}

// GetUInt16 retrieves the data at the specified key and attempt to convert the
// data into uint16 form if necessary. Returns the uint16 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError if a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into uint16 form.
func (sdtMap Map) GetUInt16(key string) (val uint16, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getUint16(elem)
}

// GetUInt32 retrieves the data at the specified key and attempt to convert the
// data into uint32 form if necessary. Returns the uint32 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError if the data cannot be converted into uint32 form.
func (sdtMap Map) GetUInt32(key string) (val uint32, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getUint32(elem)
}

// GetUInt64 retrieves the data at the specified key and attempt to convert the
// data into uint64 form if necessary. Returns the uint64 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into uint64 form
func (sdtMap Map) GetUInt64(key string) (val uint64, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getUint64(elem)
}

// GetFloat32 retrieves the data at the specified key and attempt to convert the
// data into float32 form if necessary. Returns the float32 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into float32 form.
func (sdtMap Map) GetFloat32(key string) (val float32, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getFloat32(elem)
}

// GetFloat64 retrieves the data at the specified key and attempt to convert the
// data into float64 form if necessary. Returns the float64 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into float64 form.
func (sdtMap Map) GetFloat64(key string) (val float64, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return 0, &KeyNotFoundError{key}
	}
	return getFloat64(elem)
}

// GetString retrieves the data at the specified key and attempt to convert the
// data into string form if necessary. Returns the string value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into string form.
func (sdtMap Map) GetString(key string) (val string, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return "", &KeyNotFoundError{key}
	}
	return getString(elem)
}

// GetWChar retrieves the data at the specified key and attempt to convert the
// data into WChar form if necessary. Returns the WChar value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into WChar form.
func (sdtMap Map) GetWChar(key string) (val WChar, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return WChar(0), &KeyNotFoundError{key}
	}
	return getWChar(elem)
}

// GetByteArray retrieves the data at the specified key and attempt to convert the
// data into []byte form if necessary. Returns the []byte value of the
// field, and one of the following errors if it occurred:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data cannot be converted into []byte form.
func (sdtMap Map) GetByteArray(key string) (val []byte, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return nil, &KeyNotFoundError{key}
	}
	return getByteArray(elem)
}

// GetMap retrieves the data at the specified key as an sdt.Map.
// One of the following errors may occur:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data is not of type sdt.Map.
func (sdtMap Map) GetMap(key string) (val Map, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return nil, &KeyNotFoundError{key}
	}
	return getMap(elem)
}

// GetStream retrieves the data at the specified key as an sdt.Stream. One of the following errors may occur:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data is not of type sdt.Stream.
func (sdtMap Map) GetStream(key string) (val Stream, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return nil, &KeyNotFoundError{key}
	}
	return getStream(elem)
}

// GetDestination retrieves the data at the specified key as a Destination.
// One of the following errors may occur:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data is not of type Destination.
func (sdtMap Map) GetDestination(key string) (val resource.Destination, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return nil, &KeyNotFoundError{key}
	}
	return getDestination(elem)
}

// GetQueue retrieves the data at the specified key as a Queue.
// One of the following errors may occur:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data is not of type Queue.
func (sdtMap Map) GetQueue(key string) (val *resource.Queue, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return nil, &KeyNotFoundError{key}
	}
	return getQueue(elem)
}

// GetTopic retrieves the data at the specified key as a Topic.
// One of the following errors may occur:
//
// - sdt.KeyNotFoundError - If a mapping for the specified key was not found.
//
// - sdt.FormatConversionError - If the data is not of type Topic.
func (sdtMap Map) GetTopic(key string) (val *resource.Topic, err error) {
	elem, ok := sdtMap[key]
	if !ok {
		return nil, &KeyNotFoundError{key}
	}
	return getTopic(elem)
}

// GetBool retrieves the data at the specified index and attempt to convert the
// data into boolean form if necessary. Returns the boolean value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into boolean form.
func (sdtStream Stream) GetBool(index int) (val bool, err error) {
	if index >= len(sdtStream) || index < 0 {
		return false, &OutOfBoundsError{Index: index}
	}
	return getBool(sdtStream[index])
}

// GetByte retrieves the data at the specified index and attempt to convert the
// data into byte form if necessary. Returns the byte value of the field,
// and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into byte form.
func (sdtStream Stream) GetByte(index int) (val byte, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getUint8(sdtStream[index])
}

// GetInt retrieves the data at the specified index and attempt to convert the
// data into int form if necessary. Returns the int value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into int form.
func (sdtStream Stream) GetInt(index int) (val int, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getInt(sdtStream[index])
}

// GetUInt retrieves the data at the specified index and attempt to convert the
// data into uint form if necessary. Returns the uint value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into uint form.
func (sdtStream Stream) GetUInt(index int) (val uint, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getUint(sdtStream[index])
}

// GetInt8 retrieves the data at the specified index and attempt to convert the
// data into int8 form if necessary. Returns the int8 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into int8 form.
func (sdtStream Stream) GetInt8(index int) (val int8, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getInt8(sdtStream[index])
}

// GetInt16 retrieves the data at the specified index and attempt to convert the
// data into int16 form if necessary. Returns the int16 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into int16 form.
func (sdtStream Stream) GetInt16(index int) (val int16, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getInt16(sdtStream[index])
}

// GetInt32 retrieves the data at the specified index and attempt to convert the
// data into int32 form if necessary. Returns the int32 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into int32 form.
func (sdtStream Stream) GetInt32(index int) (val int32, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getInt32(sdtStream[index])
}

// GetInt64 retrieves the data at the specified index and attempt to convert the
// data into int64 form if necessary. Returns the int64 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into int64 form.
func (sdtStream Stream) GetInt64(index int) (val int64, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getInt64(sdtStream[index])
}

// GetUInt8 retrieves the data at the specified index and attempt to convert the
// data into uint8 form if necessary. Returns the uint8 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into uint8 form.
func (sdtStream Stream) GetUInt8(index int) (val uint8, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getUint8(sdtStream[index])
}

// GetUInt16 retrieves the data at the specified index and attempt to convert the
// data into uint16 form if necessary. Returns the uint16 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into uint16 form.
func (sdtStream Stream) GetUInt16(index int) (val uint16, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getUint16(sdtStream[index])
}

// GetUInt32 retrieves the data at the specified index and attempt to convert the
// data into uint32 form if necessary. Returns the uint32 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into uint32 form.
func (sdtStream Stream) GetUInt32(index int) (val uint32, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getUint32(sdtStream[index])
}

// GetUInt64 retrieves the data at the specified index and attempt to convert the
// data into uint64 form if necessary. Returns the uint64 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into uint64 form.
func (sdtStream Stream) GetUInt64(index int) (val uint64, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getUint64(sdtStream[index])
}

// GetFloat32 retrieves the data at the specified index and attempt to convert the
// data into float32 form if necessary. Returns the float32 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into float32 form.
func (sdtStream Stream) GetFloat32(index int) (val float32, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getFloat32(sdtStream[index])
}

// GetFloat64 retrieves the data at the specified index and attempt to convert the
// data into float64 form if necessary. Returns the float64 value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into float64 form.
func (sdtStream Stream) GetFloat64(index int) (val float64, err error) {
	if index >= len(sdtStream) || index < 0 {
		return 0, &OutOfBoundsError{Index: index}
	}
	return getFloat64(sdtStream[index])
}

// GetString retrieves the data at the specified index and attempt to convert the
// data into string form if necessary. Returns the string value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into string form.
func (sdtStream Stream) GetString(index int) (val string, err error) {
	if index >= len(sdtStream) || index < 0 {
		return "", &OutOfBoundsError{Index: index}
	}
	return getString(sdtStream[index])
}

// GetWChar retrieves the data at the specified index and attempt to convert the
// data into WChar form if necessary. Returns the WChar value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into WChar form.
func (sdtStream Stream) GetWChar(index int) (val WChar, err error) {
	if index >= len(sdtStream) || index < 0 {
		return WChar(0), &OutOfBoundsError{Index: index}
	}
	return getWChar(sdtStream[index])
}

// GetByteArray retrieves the data at the specified index and attempt to convert the
// data into []byte form if necessary. Returns the []byte value of the
// field, and one of the following errors if it occurred:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data cannot be converted into []byte form.
func (sdtStream Stream) GetByteArray(index int) (val []byte, err error) {
	if index >= len(sdtStream) || index < 0 {
		return nil, &OutOfBoundsError{Index: index}
	}
	return getByteArray(sdtStream[index])
}

// GetMap retrieves the data at the specified index as an sdt.Map.
// One of the following errors may occur:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data is not of type sdt.Map.
func (sdtStream Stream) GetMap(index int) (val Map, err error) {
	if index >= len(sdtStream) || index < 0 {
		return nil, &OutOfBoundsError{Index: index}
	}
	return getMap(sdtStream[index])
}

// GetStream retrieves the data at the specified index as an sdt.Stream.
// One of the following errors may occur:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data is not of type sdt.Stream.
func (sdtStream Stream) GetStream(index int) (val Stream, err error) {
	if index >= len(sdtStream) || index < 0 {
		return nil, &OutOfBoundsError{Index: index}
	}
	return getStream(sdtStream[index])
}

// GetDestination retrieves the data at the specified index as a Destination.
// One of the following errors may occur:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data is not of type Destination.
func (sdtStream Stream) GetDestination(index int) (val resource.Destination, err error) {
	if index >= len(sdtStream) || index < 0 {
		return nil, &OutOfBoundsError{Index: index}
	}
	return getDestination(sdtStream[index])
}

// GetQueue retrieves the data at the specified index as a Queue.
// One of the following errors may occur:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data is not of type Queue.
func (sdtStream Stream) GetQueue(index int) (val *resource.Queue, err error) {
	if index >= len(sdtStream) || index < 0 {
		return nil, &OutOfBoundsError{Index: index}
	}
	return getQueue(sdtStream[index])
}

// GetTopic retrieves the data at the specified index as a Topic.
// One of the following errors may occur:
//
// - sdt.OutOfBoundsError - If the specified index is out of the stream's bounds.
//
// - sdt.FormatConversionError - If the data is not of type Topic.
func (sdtStream Stream) GetTopic(index int) (val *resource.Topic, err error) {
	if index >= len(sdtStream) || index < 0 {
		return nil, &OutOfBoundsError{Index: index}
	}
	return getTopic(sdtStream[index])
}

func getBool(elem interface{}) (bool, error) {
	switch casted := elem.(type) {
	case bool:
		return casted, nil
	case int:
		return casted != 0, nil
	case uint:
		return casted != 0, nil
	case uint8:
		return casted != 0, nil
	case int8:
		return casted != 0, nil
	case uint16:
		return casted != 0, nil
	case int16:
		return casted != 0, nil
	case uint32:
		return casted != 0, nil
	case int32:
		return casted != 0, nil
	case uint64:
		return casted != 0, nil
	case int64:
		return casted != 0, nil
	case string:
		if casted == "true" || casted == "1" {
			return true, nil
		} else if casted == "false" || casted == "0" {
			return false, nil
		}
		return false, &FormatConversionError{
			fmt.Sprintf("cannot convert string '%s' to bool", casted),
			casted,
		}
	// case []byte:
	// case float32:
	// case float64:
	// case WChar:
	default:
		return false, defaultFormatConversionError(elem, "bool")
	}
}

func getUint8(elem interface{}) (uint8, error) {
	converted, err := parseUint(elem, 8)
	if err != nil {
		return 0, err
	}
	return uint8(converted), nil
}

func getInt8(elem interface{}) (int8, error) {
	converted, err := parseInt(elem, 8)
	if err != nil {
		return 0, err
	}
	return int8(converted), nil
}

func getUint16(elem interface{}) (uint16, error) {
	converted, err := parseUint(elem, 16)
	if err != nil {
		return 0, err
	}
	return uint16(converted), nil
}

func getInt16(elem interface{}) (int16, error) {
	converted, err := parseInt(elem, 16)
	if err != nil {
		return 0, err
	}
	return int16(converted), nil
}

func getUint32(elem interface{}) (uint32, error) {
	converted, err := parseUint(elem, 32)
	if err != nil {
		return 0, err
	}
	return uint32(converted), nil
}

func getInt32(elem interface{}) (int32, error) {
	converted, err := parseInt(elem, 32)
	if err != nil {
		return 0, err
	}
	return int32(converted), nil
}

func getUint64(elem interface{}) (uint64, error) {
	converted, err := parseUint(elem, 64)
	if err != nil {
		return 0, err
	}
	return uint64(converted), nil
}

func getInt64(elem interface{}) (int64, error) {
	converted, err := parseInt(elem, 64)
	if err != nil {
		return 0, err
	}
	return int64(converted), nil
}

var uintSize = 8 * int(unsafe.Sizeof(uint(0)))

func getUint(elem interface{}) (uint, error) {
	converted, err := parseUint(elem, uintSize)
	if err != nil {
		return 0, err
	}
	return uint(converted), nil
}

var intSize = 8 * int(unsafe.Sizeof(int(0)))

func getInt(elem interface{}) (int, error) {
	converted, err := parseInt(elem, intSize)
	if err != nil {
		return 0, err
	}
	return int(converted), nil
}

func parseInt(elem interface{}, bits int) (int64, error) {
	switch casted := elem.(type) {
	case bool:
		if casted {
			return 1, nil
		}
		return 0, nil
	case []byte, float32, float64, WChar:
		goto conversionErr
	case int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
		goto fromString
	case string:
		goto fromString
	default:
		goto conversionErr
	}
conversionErr:
	return 0, defaultFormatConversionError(elem, fmt.Sprintf("int%d", bits))
fromString:
	str := fmt.Sprint(elem)
	i, err := strconv.ParseInt(str, 10, bits)
	if err != nil {
		return 0, nestedFormatConversionError(elem, fmt.Sprintf("int%d", bits), err)
	}
	return int64(i), nil
}

func parseUint(elem interface{}, bits int) (uint64, error) {
	switch casted := elem.(type) {
	case bool:
		if casted {
			return 1, nil
		}
		return 0, nil

	case []byte, float32, float64, WChar:
		goto conversionErr
	case int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
		goto fromString
	case string:
		goto fromString
	default:
		goto conversionErr
	}
conversionErr:
	return 0, defaultFormatConversionError(elem, fmt.Sprintf("uint%d", bits))
fromString:
	str := fmt.Sprint(elem)
	i, err := strconv.ParseUint(str, 10, bits)
	if err != nil {
		return 0, nestedFormatConversionError(elem, fmt.Sprintf("uint%d", bits), err)
	}
	return uint64(i), nil
}

func getByteArray(elem interface{}) ([]byte, error) {
	switch casted := elem.(type) {
	case []byte:
		return casted, nil
	case string:
		return []byte(casted), nil
	default:
		return nil, defaultFormatConversionError(elem, "[]uint8")
	}
}

func getFloat32(elem interface{}) (float32, error) {
	const typeString = "float32"
	switch casted := elem.(type) {
	case float32:
		return casted, nil
	case string:
		f, err := strconv.ParseFloat(casted, 32)
		if err != nil {
			return 0, nestedFormatConversionError(casted, typeString, err)
		}
		return float32(f), nil
	default:
		return 0, defaultFormatConversionError(elem, typeString)
	}
}

func getFloat64(elem interface{}) (float64, error) {
	const typeString = "float64"
	switch casted := elem.(type) {
	case float64:
		return casted, nil
	case string:
		f, err := strconv.ParseFloat(casted, 64)
		if err != nil {
			return 0, nestedFormatConversionError(casted, typeString, err)
		}
		return f, nil
	default:
		return 0, defaultFormatConversionError(elem, typeString)
	}
}

func getString(elem interface{}) (string, error) {
	switch casted := elem.(type) {
	case string:
		return casted, nil
	default:
		return "", defaultFormatConversionError(elem, "string")
	}
}

func getWChar(elem interface{}) (WChar, error) {
	switch casted := elem.(type) {
	case uint8:
		return WChar(casted), nil
	case int8:
		return WChar(casted), nil
	case WChar:
		return casted, nil
	case string:
		if len(casted) == 1 {
			return WChar(casted[0]), nil
		}
		return 0, &FormatConversionError{fmt.Sprintf("cannot convert string of length %d to sdt.WChar", len(casted)), casted}
	default:
		return 0, defaultFormatConversionError(elem, "sdt.WChar")
	}
}

func getMap(elem interface{}) (Map, error) {
	if casted, ok := elem.(Map); ok {
		return casted, nil
	}
	return nil, defaultFormatConversionError(elem, "sdt.Map")
}

func getStream(elem interface{}) (Stream, error) {
	if casted, ok := elem.(Stream); ok {
		return casted, nil
	}
	return nil, defaultFormatConversionError(elem, "sdt.Stream")
}

func getQueue(elem interface{}) (*resource.Queue, error) {
	if casted, ok := elem.(*resource.Queue); ok {
		return casted, nil
	}
	return nil, defaultFormatConversionError(elem, "*resource.Queue")
}

func getTopic(elem interface{}) (*resource.Topic, error) {
	if casted, ok := elem.(*resource.Topic); ok {
		return casted, nil
	}
	return nil, defaultFormatConversionError(elem, "*resource.Topic")
}

func getDestination(elem interface{}) (resource.Destination, error) {
	if casted, ok := elem.(resource.Destination); ok {
		return casted, nil
	}
	return nil, defaultFormatConversionError(elem, "resource.Destination")
}
