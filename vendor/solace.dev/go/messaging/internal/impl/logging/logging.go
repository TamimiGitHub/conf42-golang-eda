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

// Package logging is defined below
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
)

// LogLevel type
type LogLevel byte

const (
	// Critical defined
	Critical LogLevel = iota

	// Error defined
	Error LogLevel = iota

	// Warning defined
	Warning LogLevel = iota

	// Info defined
	Info LogLevel = iota

	// Debug defined
	Debug LogLevel = iota

	logLevelCount LogLevel = iota

	defaultLogLevel = Warning

	defaultFlags = log.Ldate | log.Ltime | log.Lshortfile

	defaultCalldepth = 2
)

// Default defined
var Default LogLevelLogger

func init() {
	nativeLogger := log.New(os.Stderr, "", defaultFlags)
	logger := &logLevelLoggerCore{}
	logger.Logger = nativeLogger
	logger.logLevel = defaultLogLevel
	logger.calldepth = defaultCalldepth
	Default = logger
}

// For function
func For(item interface{}) LogLevelLogger {
	return Default.For(item)
}

// LogLevelNames defined
var LogLevelNames = []string{
	"CRITICAL",
	"ERROR",
	"WARNING",
	"INFO",
	"DEBUG",
}

// LogLevelLogger defined
type LogLevelLogger interface {
	SetLevel(logLevel LogLevel)
	GetLevel() LogLevel

	IsCriticalEnabled() bool
	IsErrorEnabled() bool
	IsWarningEnabled() bool
	IsInfoEnabled() bool
	IsDebugEnabled() bool

	Critical(message string)
	Error(message string)
	Warning(message string)
	Info(message string)
	Debug(message string)

	For(item interface{}) LogLevelLogger

	SetFlags(flag int)
	SetOutput(writer io.Writer)
}

type logLevelLoggerCore struct {
	*log.Logger
	logLevel  LogLevel
	calldepth int
}

func (logger *logLevelLoggerCore) IsCriticalEnabled() bool {
	return logger.logLevel >= Critical
}

func (logger *logLevelLoggerCore) IsErrorEnabled() bool {
	return logger.logLevel >= Error
}

func (logger *logLevelLoggerCore) IsWarningEnabled() bool {
	return logger.logLevel >= Warning
}

func (logger *logLevelLoggerCore) IsInfoEnabled() bool {
	return logger.logLevel >= Info
}

func (logger *logLevelLoggerCore) IsDebugEnabled() bool {
	return logger.logLevel >= Debug
}

func (logger *logLevelLoggerCore) SetLevel(logLevel LogLevel) {
	if logLevel < logLevelCount {
		logger.logLevel = logLevel
	}
}

func (logger *logLevelLoggerCore) GetLevel() LogLevel {
	return logger.logLevel
}

func (logger *logLevelLoggerCore) For(item interface{}) LogLevelLogger {
	var prefix string
	if item != nil {
		if t := reflect.TypeOf(item); t.Kind() == reflect.Ptr {
			prefix = fmt.Sprintf("%s$%p ", t.Elem().Name(), item)
		}
		// Do not generate prefix in case where item is not a pointer type
	}
	return &logAdapter{logger, prefix}
}

func (logger *logLevelLoggerCore) Critical(message string) {
	if logger.IsCriticalEnabled() {
		logger.Output(logger.calldepth, LogLevelNames[Critical]+" "+message)
	}
}

func (logger *logLevelLoggerCore) Error(message string) {
	if logger.IsErrorEnabled() {
		logger.Output(logger.calldepth, LogLevelNames[Error]+" "+message)
	}
}

func (logger *logLevelLoggerCore) Warning(message string) {
	if logger.IsWarningEnabled() {
		logger.Output(logger.calldepth, LogLevelNames[Warning]+" "+message)
	}
}

func (logger *logLevelLoggerCore) Info(message string) {
	if logger.IsInfoEnabled() {
		logger.Output(logger.calldepth, LogLevelNames[Info]+" "+message)
	}
}

func (logger *logLevelLoggerCore) Debug(message string) {
	if logger.IsDebugEnabled() {
		logger.Output(logger.calldepth, LogLevelNames[Debug]+" "+message)
	}
}

type logAdapter struct {
	*logLevelLoggerCore
	prefix string
}

func (logger *logAdapter) Critical(message string) {
	if logger.IsCriticalEnabled() {
		logger.Output(logger.calldepth, LogLevelNames[Critical]+" "+logger.prefix+message)
	}
}

func (logger *logAdapter) Error(message string) {
	if logger.IsErrorEnabled() {
		logger.Output(logger.calldepth, LogLevelNames[Error]+" "+logger.prefix+message)
	}
}

func (logger *logAdapter) Warning(message string) {
	if logger.IsWarningEnabled() {
		logger.Output(logger.calldepth, LogLevelNames[Warning]+" "+logger.prefix+message)
	}
}

func (logger *logAdapter) Info(message string) {
	if logger.IsInfoEnabled() {
		logger.Output(logger.calldepth, LogLevelNames[Info]+" "+logger.prefix+message)
	}
}

func (logger *logAdapter) Debug(message string) {
	if logger.IsDebugEnabled() {
		logger.Output(logger.calldepth, LogLevelNames[Debug]+" "+logger.prefix+message)
	}
}
