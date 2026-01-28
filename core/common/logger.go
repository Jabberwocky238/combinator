package combinator

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger is the global logger instance
var Logger *logrus.Logger
var GlobalErrorBuilder = NewErrorBuilder()

type ErrorBuilder struct {
	Namespaces []string
}

func NewErrorBuilder() *ErrorBuilder {
	return &ErrorBuilder{
		Namespaces: []string{},
	}
}

func (eb *ErrorBuilder) With(ns string) *ErrorBuilder {
	ebnew := &ErrorBuilder{
		Namespaces: make([]string, len(eb.Namespaces)),
	}
	copy(ebnew.Namespaces, eb.Namespaces)
	ebnew.Namespaces = append(ebnew.Namespaces, ns)
	return ebnew
}

func (eb *ErrorBuilder) String(msg string, args ...any) string {
	if len(eb.Namespaces) == 0 {
		return msg
	}
	ns := strings.Join(eb.Namespaces, ".")
	return fmt.Sprintf("[%s] %s", ns, fmt.Sprintf(msg, args...))
}

func (eb *ErrorBuilder) Error(msg string, args ...any) error {
	return fmt.Errorf("%s", eb.String(msg, args...))
}

func init() {
	Logger = logrus.New()

	// Set output to stdout
	Logger.SetOutput(os.Stdout)

	// Set log level
	Logger.SetLevel(logrus.DebugLevel)

	// Set formatter
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

// SetLogLevel sets the global log level
func SetLogLevel(level string) {
	switch level {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "info":
		Logger.SetLevel(logrus.InfoLevel)
	case "warn":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}
}
