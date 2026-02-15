package common

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

// NamespacedLogger provides namespaced logging and error building.
// Use NewRootLogger() to create the root, and .With("sub") to derive child loggers.
type NamespacedLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
	ns     string
}

// NewRootLogger creates the root logger instance. Call once in main/init.
func NewRootLogger() *NamespacedLogger {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.SetLevel(logrus.DebugLevel)
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	return &NamespacedLogger{
		logger: l,
		entry:  l.WithField("ns", "root"),
		ns:     "root",
	}
}

// With derives a child logger with an appended namespace. e.g. logger.With("sqlite") → "rdb.sqlite"
func (l *NamespacedLogger) With(sub string) *NamespacedLogger {
	ns := l.ns + "." + sub
	return &NamespacedLogger{
		logger: l.logger,
		entry:  l.logger.WithField("ns", ns),
		ns:     ns,
	}
}

// --- Logging methods ---

func (l *NamespacedLogger) Debugf(format string, args ...any) {
	l.entry.Debugf(format, args...)
}

func (l *NamespacedLogger) Infof(format string, args ...any) {
	l.entry.Infof(format, args...)
}

func (l *NamespacedLogger) Warnf(format string, args ...any) {
	l.entry.Warnf(format, args...)
}

func (l *NamespacedLogger) Errorf(format string, args ...any) {
	l.entry.Errorf(format, args...)
}

func (l *NamespacedLogger) Fatalf(format string, args ...any) {
	l.entry.Fatalf(format, args...)
}

func (l *NamespacedLogger) Debug(args ...any) {
	l.entry.Debug(args...)
}

func (l *NamespacedLogger) Info(args ...any) {
	l.entry.Info(args...)
}

func (l *NamespacedLogger) Warn(args ...any) {
	l.entry.Warn(args...)
}

func (l *NamespacedLogger) Error(args ...any) {
	l.entry.Error(args...)
}

func (l *NamespacedLogger) Fatal(args ...any) {
	l.entry.Fatal(args...)
}

// --- Error building methods ---

// NewError creates a namespaced error: [rdb.sqlite] msg
func (l *NamespacedLogger) NewError(msg string, args ...any) error {
	return fmt.Errorf("[%s] %s", l.ns, fmt.Sprintf(msg, args...))
}

// Str formats a namespaced string: [rdb.sqlite] msg
func (l *NamespacedLogger) Str(msg string, args ...any) string {
	return fmt.Sprintf("[%s] %s", l.ns, fmt.Sprintf(msg, args...))
}

// SetLogLevel sets the log level on this logger and all children
func (l *NamespacedLogger) SetLogLevel(level string) {
	switch level {
	case "debug":
		l.logger.SetLevel(logrus.DebugLevel)
	case "info":
		l.logger.SetLevel(logrus.InfoLevel)
	case "warn":
		l.logger.SetLevel(logrus.WarnLevel)
	case "error":
		l.logger.SetLevel(logrus.ErrorLevel)
	default:
		l.logger.SetLevel(logrus.InfoLevel)
	}
}
