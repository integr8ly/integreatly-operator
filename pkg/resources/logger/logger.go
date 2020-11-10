package logger

import (
	logrus "github.com/sirupsen/logrus"
)

const (
	ControllerLogContext = "controller"
	StageLogContext      = "stage"
	ProductLogContext    = "product"
	ComponentLogContext  = "component"
)

type Logger struct {
	Logger *logrus.Entry
}
type Fields map[string]interface{}

func NewLogger() Logger {
	return Logger{
		Logger: logrus.NewEntry(logrus.StandardLogger()),
	}
}

func (l Logger) WithContext(fields Fields) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields(fields))
}

func (l Logger) Infof(message string, fields map[string]interface{}) {
	l.Logger.WithFields(fields).Info(message)
}

func (l Logger) Info(message string) {
	l.Logger.Info(message)
}

func (l Logger) Debugf(message string, fields map[string]interface{}) {
	l.Logger.WithFields(fields).Debug(message)
}

func (l Logger) Debug(message string) {
	l.Logger.Debug(message)
}

func (l Logger) Errorf(message string, fields map[string]interface{}) {
	l.Logger.WithFields(fields).Debug(message)
}

func (l Logger) Error(message string) {
	l.Logger.Debug(message)
}

func (l Logger) Fatalf(message string, fields map[string]interface{}) {
	l.Logger.WithFields(fields).Debug(message)
}

func (l Logger) Fatal(message string) {
	l.Logger.Debug(message)
}

func (l Logger) Warningf(message string, fields map[string]interface{}) {
	l.Logger.WithFields(fields).Debug(message)
}

func (l Logger) Warning(message string) {
	l.Logger.Debug(message)
}
