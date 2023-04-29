package logger

import (
	"github.com/go-logr/logr"
	liberr "github.com/konveyor/tackle2-hub/error"
	"os"
	"strconv"
)

const (
	Stack = "stacktrace"
	Error = "error"
	None  = ""
)
const (
	EnvDevelopment = "LOG_DEVELOPMENT"
	EnvLevel       = "LOG_LEVEL"
)

//
// Settings is sink settings.
var Settings _Settings

func init() {
	Settings.Load()
}

//
// Factory is a factory.
var Factory Builder

func init() {
	Factory = &ZapBuilder{}
}

//
// Sink -.
type Sink struct {
	Delegate logr.LogSink
	name     string
}

//
// WithName returns a named logger.
func WithName(name string, kvpair ...interface{}) logr.Logger {
	d := Factory.New()
	s := &Sink{
		Delegate: d.GetSink(),
		name:     name,
	}
	s.Delegate = s.Delegate.WithValues(kvpair...)
	s.Delegate = s.Delegate.WithName(name)
	return logr.New(s)
}

func (s *Sink) Init(info logr.RuntimeInfo) {
	return
}

//
// Info logs at info.
func (s *Sink) Info(level int, message string, kvpair ...interface{}) {
	s.Delegate.Info(level, message, kvpair...)
}

//
// Error logs an error.
func (s *Sink) Error(err error, message string, kvpair ...interface{}) {
	if err == nil {
		return
	}
	le, wrapped := err.(*liberr.Error)
	if wrapped {
		err = le.Unwrap()
		if context := le.Context(); context != nil {
			context = append(
				context,
				kvpair...)
			kvpair = context
		}
		kvpair = append(
			kvpair,
			Error,
			le.Error(),
			Stack,
			le.Stack())

		s.Delegate.Info(0, message, kvpair...)
		return
	}
	if wErr, wrapped := err.(interface {
		Unwrap() error
	}); wrapped {
		err = wErr.Unwrap()
	}
	if err == nil {
		return
	}

	s.Delegate.Error(err, message, kvpair...)
}

//
// Trace logs an error without a description.
func (s *Sink) Trace(err error, kvpair ...interface{}) {
	s.Error(err, None, kvpair...)
}

//
// Enabled returns whether logger is enabled.
func (s *Sink) Enabled(level int) bool {
	return s.Delegate.Enabled(level)
}

//
// WithName returns a logger with name.
func (s *Sink) WithName(name string) logr.LogSink {
	return &Sink{
		Delegate: s.Delegate.WithName(name),
		name:     name,
	}
}

//
// WithValues returns a logger with values.
func (s *Sink) WithValues(kvpair ...interface{}) logr.LogSink {
	return &Sink{
		Delegate: s.Delegate.WithValues(kvpair...),
		name:     s.name,
	}
}

//
// Package settings.
type _Settings struct {
	// Debug threshold.
	// Level determines when the real
	// debug logger is used.
	DebugThreshold int
	// Development configuration.
	Development bool
	// Info level threshold.
	// Higher level increases verbosity.
	Level int
}

//
// Load determine development logger.
func (r *_Settings) Load() {
	r.DebugThreshold = 4
	if s, found := os.LookupEnv(EnvDevelopment); found {
		bv, err := strconv.ParseBool(s)
		if err == nil {
			r.Development = bv
		}
	}
	if s, found := os.LookupEnv(EnvLevel); found {
		n, err := strconv.ParseInt(s, 10, 8)
		if err == nil {
			r.Level = int(n)
		}
	}
}

//
// The level is at or above the debug threshold.
func (r *_Settings) atDebug(level int) bool {
	return level >= r.DebugThreshold
}
