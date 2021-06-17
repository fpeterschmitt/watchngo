package watcher

import "log"

type Logger interface {
	Debug(fmt string, args ...interface{})
	Log(fmt string, args ...interface{})
}

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
