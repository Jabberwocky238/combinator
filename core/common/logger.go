package combinator

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Logger is the global logger instance
var Logger *logrus.Logger

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
