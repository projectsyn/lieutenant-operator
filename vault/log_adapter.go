package vault

import (
	"errors"

	"github.com/go-logr/logr"
)

type logAdapter struct {
	logr.Logger
}

var errLogAdapterLogError = errors.New("vault error log")

func (l logAdapter) Trace(msg string, fields ...map[string]interface{}) {
	l.V(2).Info(msg, flatten(fields)...)
}

func (l logAdapter) Debug(msg string, fields ...map[string]interface{}) {
	l.V(1).Info(msg, flatten(fields)...)
}

func (l logAdapter) Info(msg string, fields ...map[string]interface{}) {
	l.V(0).Info(msg, flatten(fields)...)
}

func (l logAdapter) Warn(msg string, fields ...map[string]interface{}) {
	l.V(0).Info("Warn: "+msg, flatten(fields)...)
}

func (l logAdapter) Error(msg string, fields ...map[string]interface{}) {
	l.V(0).Error(errLogAdapterLogError, msg, flatten(fields)...)
}

func flatten(fields []map[string]interface{}) []interface{} {
	flattenedLen := 0
	for _, field := range fields {
		flattenedLen += len(field) * 2
	}

	flattened := make([]interface{}, 0, flattenedLen)
	for _, field := range fields {
		for name, value := range field {
			flattened = append(flattened, name, value)
		}
	}
	return flattened
}
