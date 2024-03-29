//go:generate mockgen -source=logger.go -destination=mock_logger_test.go -package=pkg_test Logger

package pkg

import "log"

type Logger interface {
	Debug(fmt string, args ...interface{})
	Log(fmt string, args ...interface{})
}

type SilentLogger struct{}

func (s SilentLogger) Debug(fmt string, args ...interface{}) {}

func (s SilentLogger) Log(fmt string, args ...interface{}) {}

type DebugLogger struct{ Logger Logger }

func (d DebugLogger) Debug(fmt string, args ...interface{}) {
	d.Log(fmt, args...)
}

func (d DebugLogger) Log(fmt string, args ...interface{}) {
	d.Logger.Log(fmt, args...)
}

type InfoLogger struct{ Logger *log.Logger }

func (i InfoLogger) Debug(_ string, _ ...interface{}) {}

func (i InfoLogger) Log(fmt string, args ...interface{}) {
	i.Logger.Printf(fmt, args...)
}
