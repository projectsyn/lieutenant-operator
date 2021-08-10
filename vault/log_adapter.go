package vault

import (
	"errors"

	"github.com/go-logr/logr"
)

type logAdapter struct {
	logr.Logger
}

var errLogAdapterLogError = errors.New("vault error log")

func (l logAdapter) Debug(msg string, keysAndValues ...interface{}) {
	l.V(1).Info(msg, keysAndValues...)
}

func (l logAdapter) Warn(msg string, keysAndValues ...interface{}) {
	l.Info("WARN: "+msg, keysAndValues...)
}

func (l logAdapter) Error(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(errLogAdapterLogError, msg, keysAndValues...)
}
