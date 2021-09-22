// Package logger offers simple logging
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime/debug"
	"sync"
)

type severity int

type logger interface {
	Output(calldepth int, s string) error
	SetOutput(w io.Writer)
	SetFlags(flag int)
}

// Severity levels.
const (
	sDebug severity = iota
	sInfo
	sWarning
	sError
	sFatal
	sCount
)

// Severity tags.
const (
	tagDebug   = "\033[0m[DEBUG] "
	tagInfo    = "\033[97m[INFO]  "
	tagWarning = "\033[33m[WARN]  "
	tagError   = "\033[31m[ERROR] "
	tagFatal   = "\033[1;31m[FATAL] "
)

const (
	flags    = log.Lmsgprefix | log.Ltime
	resetSeq = "\033[0m"
)

var (
	logLock       sync.Mutex
	defaultLogger *Logger
)

func newLoggers() [sCount]logger {
	return [sCount]logger{
		log.New(os.Stderr, tagDebug, flags),
		log.New(os.Stderr, tagInfo, flags),
		log.New(os.Stderr, tagWarning, flags),
		log.New(os.Stderr, tagError, flags),
		log.New(os.Stderr, tagFatal, flags),
	}
}

// initialize resets defaultLogger.  Which allows tests to reset environment.
func initialize() {
	defaultLogger = &Logger{
		loggers:     newLoggers(),
		minSeverity: 0,
	}
}

func init() {
	initialize()
}

// Init sets up logging and should be called before log functions, usually in
// the caller's main(). Default log functions can be called before Init(), but
// every severity will be logged.
// The first call to Init populates the default logger and returns the
// generated logger, subsequent calls to Init will only return the generated
// logger.
func Init(level int) *Logger {
	l := Logger{
		loggers:     newLoggers(),
		initialized: true,
	}
	l.SetLevel(level)

	logLock.Lock()
	defer logLock.Unlock()
	if !defaultLogger.initialized {
		defaultLogger = &l
	}

	return &l
}

// A Logger represents an active logging object. Multiple loggers can be used
// simultaneously even if they are using the same writers.
type Logger struct {
	loggers     [sCount]logger
	minSeverity severity
	initialized bool
}

func (l *Logger) output(s severity, v ...interface{}) {
	if s < l.minSeverity {
		return
	}
	str := fmt.Sprint(v...) + resetSeq
	logLock.Lock()
	defer logLock.Unlock()
	l.loggers[s].Output(3, str)
}

func (l *Logger) outputf(s severity, format string, v ...interface{}) {
	if s < l.minSeverity {
		return
	}
	str := fmt.Sprintf(format, v...) + resetSeq
	logLock.Lock()
	defer logLock.Unlock()
	l.loggers[s].Output(3, str)
}

// SetOutput changes the output of the logger.
func (l *Logger) SetOutput(w io.Writer) {
	for _, logger := range l.loggers {
		logger.SetOutput(w)
	}
}

// SetLevel sets the verbosity level of the logger.
func (l *Logger) SetLevel(lvl int) {
	l.minSeverity = sFatal - severity(lvl)
}

// SetFlags sets the output flags of the logger.
func (l *Logger) SetFlags(flag int) {
	for _, logger := range l.loggers {
		logger.SetFlags(flag)
	}
}

// Debug logs with the Debug severity.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Debug(v ...interface{}) {
	l.output(sDebug, v...)
}

// Debugf logs with the Debug severity.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.outputf(sDebug, format, v...)
}

// Info logs with the Info severity.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Info(v ...interface{}) {
	l.output(sInfo, v...)
}

// Infof logs with the Info severity.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Infof(format string, v ...interface{}) {
	l.outputf(sInfo, format, v...)
}

// Warning logs with the Warning severity.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Warning(v ...interface{}) {
	l.output(sWarning, v...)
}

// Warningf logs with the Warning severity.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warningf(format string, v ...interface{}) {
	l.outputf(sWarning, format, v...)
}

// Error logs with the Error severity.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Error(v ...interface{}) {
	v = append(v, "\n"+string(debug.Stack()))
	l.output(sError, v...)
}

// Errorf logs with the Error severity.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Errorf(format string, v ...interface{}) {
	v = append(v, "\n"+string(debug.Stack()))
	l.outputf(sError, format+"%s", v...)
}

// Panic uses the default logger and logs with the Error severity, and ends up with panic().
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.output(sError, s)
	panic(s)
}

// Panicf uses the default logger and logs with the Error severity, and ends up with panic().
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.output(sError, s)
	panic(s)
}

// Fatal logs with the Fatal severity, and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Fatal(v ...interface{}) {
	v = append(v, "\n"+string(debug.Stack()))
	l.output(sFatal, v...)
	os.Exit(1)
}

// Fatalf logs with the Fatal severity, and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Fatalf(format string, v ...interface{}) {
	v = append(v, "\n"+string(debug.Stack()))
	l.outputf(sFatal, format+"%s", v...)
	os.Exit(1)
}

// SetOutput changes the output of the default logger.
func SetOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}

// SetLevel sets the verbosity level of the default logger.
func SetLevel(lvl int) {
	defaultLogger.SetLevel(lvl)
}

// SetFlags sets the output flags of the default logger.
func SetFlags(flag int) {
	defaultLogger.SetFlags(flag)
}

// Debug uses the default logger and logs with the Debug severity.
// Arguments are handled in the manner of fmt.Print.
func Debug(v ...interface{}) {
	defaultLogger.output(sDebug, v...)
}

// Debugf uses the default logger and logs with the Debug severity.
// Arguments are handled in the manner of fmt.Printf.
func Debugf(format string, v ...interface{}) {
	defaultLogger.outputf(sDebug, format, v...)
}

// Info uses the default logger and logs with the Info severity.
// Arguments are handled in the manner of fmt.Print.
func Info(v ...interface{}) {
	defaultLogger.output(sInfo, v...)
}

// Infof uses the default logger and logs with the Info severity.
// Arguments are handled in the manner of fmt.Printf.
func Infof(format string, v ...interface{}) {
	defaultLogger.outputf(sInfo, format, v...)
}

// Warning uses the default logger and logs with the Warning severity.
// Arguments are handled in the manner of fmt.Print.
func Warning(v ...interface{}) {
	defaultLogger.output(sWarning, v...)
}

// Warningf uses the default logger and logs with the Warning severity.
// Arguments are handled in the manner of fmt.Printf.
func Warningf(format string, v ...interface{}) {
	defaultLogger.outputf(sWarning, format, v...)
}

// Error uses the default logger and logs with the Error severity.
// Arguments are handled in the manner of fmt.Print.
func Error(v ...interface{}) {
	v = append(v, "\n"+string(debug.Stack()))
	defaultLogger.output(sError, v...)
}

// Errorf uses the default logger and logs with the Error severity.
// Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, v ...interface{}) {
	v = append(v, "\n"+string(debug.Stack()))
	defaultLogger.outputf(sError, format+"%s", v...)
}

// Panic uses the default logger and logs with the Error severity, and ends up with panic().
// Arguments are handled in the manner of fmt.Print.
func Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	defaultLogger.output(sError, s)
	panic(s)
}

// Panicf uses the default logger and logs with the Error severity, and ends up with panic().
// Arguments are handled in the manner of fmt.Printf.
func Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	defaultLogger.output(sError, s)
	panic(s)
}

// Fatal uses the default logger, logs with the Fatal severity,
// and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Print.
func Fatal(v ...interface{}) {
	v = append(v, "\n"+string(debug.Stack()))
	defaultLogger.output(sFatal, v...)
	os.Exit(1)
}

// Fatalf uses the default logger, logs with the Fatal severity,
// and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Printf.
func Fatalf(format string, v ...interface{}) {
	v = append(v, "\n"+string(debug.Stack()))
	defaultLogger.outputf(sFatal, format+"%s", v...)
	debug.PrintStack()
	os.Exit(1)
}
