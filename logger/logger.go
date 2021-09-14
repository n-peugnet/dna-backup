// Package logger offers simple logging
package logger

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
)

type severity int

type logger interface {
	Output(calldepth int, s string) error
	SetFlags(flag int)
}

// Severity levels.
const (
	sInfo severity = iota
	sWarning
	sError
	sFatal
)

// Severity tags.
const (
	tagInfo    = "\033[0m[INFO]  "
	tagWarning = "\033[1;33m[WARN]  "
	tagError   = "\033[1;31m[ERROR] "
	tagFatal   = "\033[1;31m[FATAL] "
)

const (
	flags = log.Lmsgprefix | log.Ltime
)

var (
	logLock       sync.Mutex
	defaultLogger *Logger
)

// initialize resets defaultLogger.  Which allows tests to reset environment.
func initialize() {
	defaultLogger = &Logger{
		loggers: []logger{
			log.New(os.Stderr, tagInfo, flags),
			log.New(os.Stderr, tagWarning, flags),
			log.New(os.Stderr, tagError, flags),
			log.New(os.Stderr, tagFatal, flags),
		},
		level: 3,
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

	loggers := []logger{
		log.New(os.Stderr, tagInfo, flags),
		log.New(os.Stderr, tagWarning, flags),
		log.New(os.Stderr, tagError, flags),
		log.New(os.Stderr, tagFatal, flags),
	}
	l := Logger{loggers: loggers, level: level, initialized: true}

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
	loggers     []logger
	level       int
	initialized bool
}

func (l *Logger) output(s severity, depth int, txt string) {
	if s < sFatal-severity(l.level) {
		return
	}
	logLock.Lock()
	defer logLock.Unlock()
	if int(s) >= len(l.loggers) {
		panic(fmt.Sprintln("unrecognized severity:", s))
	}
	l.loggers[s].Output(3+depth, txt+"\033[0m")
}

// SetFlags sets the output flags for the logger.
func (l *Logger) SetFlags(flag int) {
	for _, logger := range l.loggers {
		logger.SetFlags(flag)
	}
}

// Info logs with the Info severity.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Info(v ...interface{}) {
	l.output(sInfo, 0, fmt.Sprint(v...))
}

// InfoDepth acts as Info but uses depth to determine which call frame to log.
// InfoDepth(0, "msg") is the same as Info("msg").
func (l *Logger) InfoDepth(depth int, v ...interface{}) {
	l.output(sInfo, depth, fmt.Sprint(v...))
}

// Infof logs with the Info severity.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Infof(format string, v ...interface{}) {
	l.output(sInfo, 0, fmt.Sprintf(format, v...))
}

// Warning logs with the Warning severity.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Warning(v ...interface{}) {
	l.output(sWarning, 0, fmt.Sprint(v...))
}

// WarningDepth acts as Warning but uses depth to determine which call frame to log.
// WarningDepth(0, "msg") is the same as Warning("msg").
func (l *Logger) WarningDepth(depth int, v ...interface{}) {
	l.output(sWarning, depth, fmt.Sprint(v...))
}

// Warningf logs with the Warning severity.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warningf(format string, v ...interface{}) {
	l.output(sWarning, 0, fmt.Sprintf(format, v...))
}

// Error logs with the ERROR severity.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Error(v ...interface{}) {
	l.output(sError, 0, fmt.Sprint(v...))
}

// ErrorDepth acts as Error but uses depth to determine which call frame to log.
// ErrorDepth(0, "msg") is the same as Error("msg").
func (l *Logger) ErrorDepth(depth int, v ...interface{}) {
	l.output(sError, depth, fmt.Sprint(v...))
}

// Errorf logs with the Error severity.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.output(sError, 0, fmt.Sprintf(format, v...))
}

// Panic uses the default logger and logs with the Error severity.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.output(sError, 0, s)
	panic(s)
}

// Panicf uses the default logger and logs with the Error severity.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.output(sError, 0, s)
	panic(s)
}

// Fatal logs with the Fatal severity, and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Fatal(v ...interface{}) {
	l.output(sFatal, 0, fmt.Sprint(v...))
	os.Exit(1)
}

// Fatalf logs with the Fatal severity, and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.output(sFatal, 0, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// SetFlags sets the output flags for the logger.
func SetFlags(flag int) {
	defaultLogger.SetFlags(flag)
}

// Info uses the default logger and logs with the Info severity.
// Arguments are handled in the manner of fmt.Print.
func Info(v ...interface{}) {
	defaultLogger.output(sInfo, 0, fmt.Sprint(v...))
}

// Infof uses the default logger and logs with the Info severity.
// Arguments are handled in the manner of fmt.Printf.
func Infof(format string, v ...interface{}) {
	defaultLogger.output(sInfo, 0, fmt.Sprintf(format, v...))
}

// Warning uses the default logger and logs with the Warning severity.
// Arguments are handled in the manner of fmt.Print.
func Warning(v ...interface{}) {
	defaultLogger.output(sWarning, 0, fmt.Sprint(v...))
}

// Warningf uses the default logger and logs with the Warning severity.
// Arguments are handled in the manner of fmt.Printf.
func Warningf(format string, v ...interface{}) {
	defaultLogger.output(sWarning, 0, fmt.Sprintf(format, v...))
}

// Error uses the default logger and logs with the Error severity.
// Arguments are handled in the manner of fmt.Print.
func Error(v ...interface{}) {
	defaultLogger.output(sError, 0, fmt.Sprint(v...))
}

// Errorf uses the default logger and logs with the Error severity.
// Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, v ...interface{}) {
	defaultLogger.output(sError, 0, fmt.Sprintf(format, v...))
}

// Panic uses the default logger and logs with the Error severity.
// Arguments are handled in the manner of fmt.Print.
func Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	defaultLogger.output(sError, 0, s)
	panic(s)
}

// Panicf uses the default logger and logs with the Error severity.
// Arguments are handled in the manner of fmt.Printf.
func Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	defaultLogger.output(sError, 0, s)
	panic(s)
}

// Fatal uses the default logger, logs with the Fatal severity,
// and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Print.
func Fatal(v ...interface{}) {
	defaultLogger.output(sFatal, 0, fmt.Sprint(v...))
	debug.PrintStack()
	os.Exit(1)
}

// Fatalf uses the default logger, logs with the Fatal severity,
// and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Printf.
func Fatalf(format string, v ...interface{}) {
	defaultLogger.output(sFatal, 0, fmt.Sprintf(format, v...))
	debug.PrintStack()
	os.Exit(1)
}
