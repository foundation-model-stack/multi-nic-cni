/*
Copyright 2018 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package logr contains utilities for fetching a new logger
// when one is not already available.
//
// # The Log Handle
//
// This package contains a root logr.Logger Log.  It may be used to
// get a handle to whatever the root logging implementation is.  By
// default, no implementation exists, and the handle returns "promises"
// to loggers.  When the implementation is set using SetLogger, these
// "promises" will be converted over to real loggers.
//
// # Logr
//
// All logging in Gate is structured, using a set of interfaces
// defined by a package called logr
// (https://pkg.go.dev/github.com/go-logr/logr).  The sub-package zap provides
// helpers for setting up logr backed by Zap (go.uber.org/zap).
package logr

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// Logger is Gate's logger interface.
type Logger = logr.Logger

// SetLogger sets a concrete logging implementation for all deferred Loggers.
func SetLogger(l Logger) {
	loggerWasSetLock.Lock()
	defer loggerWasSetLock.Unlock()

	loggerWasSet = true
	dlog.Fulfill(l.GetSink())
}

// It is safe to assume that if this wasn't set within the first 30 seconds of a binaries
// lifetime, it will never get set. The DelegatingLogSink causes a high number of memory
// allocations when not given an actual Logger, so we set a NullLogSink to avoid that.
//
// We need to keep the DelegatingLogSink because we have various inits() that get a logger from
// here. They will always get executed before any code that imports logr
// has a chance to run and hence to set an actual logger.
func init() {
	// Init is blocking, so start a new goroutine
	go func() {
		time.Sleep(30 * time.Second)
		loggerWasSetLock.Lock()
		defer loggerWasSetLock.Unlock()
		if !loggerWasSet {
			dlog.Fulfill(NullLogSink{})
		}
	}()
}

var (
	loggerWasSetLock sync.Mutex
	loggerWasSet     bool
)

// Log is the base logger used by Gate.  It delegates
// to another logr.Logger. You *must* call SetLogger to
// get any actual logging. If SetLogger is not called within
// the first 30 seconds of a binaries lifetime, it will get
// set to a NullLogSink.
var (
	dlog        = NewDelegatingLogSink(NullLogSink{})
	Log  Logger = logr.New(dlog)
)

// FromContext returns a logger with predefined values from a context.Context.
func FromContext(ctx context.Context, keysAndValues ...interface{}) Logger {
	log := Log
	if ctx != nil {
		if logger, err := logr.FromContext(ctx); err == nil {
			log = logger
		}
	}
	return log.WithValues(keysAndValues...)
}

// IntoContext takes a context and sets the logger as one of its values.
// Use FromContext function to retrieve the logger.
func IntoContext(ctx context.Context, log Logger) context.Context {
	return logr.NewContext(ctx, log)
}

// NopLog is a Logger that does nothing.
var NopLog Logger = logr.New(NullLogSink{})

// OrNop returns NopLog if the passed Logger has no LogSink.
func OrNop(l Logger) Logger {
	if l.GetSink() == nil {
		return NopLog
	}
	return l
}
