package logger

import (
	"errors"
	"testing"
)

func TestLogger(t *testing.T) {

	log := NewLoggerWithContext(Fields{ComponentLogContext: "logger_unit_test"})
	log.Debug("This is a Debug log")
	log.Debugf("This is a Debugf log", Fields{"agr1": "agr1"})

	log.Info("This is a Info log")
	log.Infof("This is a Infof log", Fields{"agr1": "agr1"})

	log.Warning("This is a Warning log")
	log.Warningf("This is a Warningf log", Fields{"agr1": "agr1"})

	err := errors.New("This is an error")
	log.Error("This is a Error log", err)
	log.Errorf("This is a Errorf log", Fields{"agr1": "agr1"}, err)

	log.Error("This is a Error log with nil err object", nil)
	log.Errorf("This is a Errorf log with nil err object", nil, nil)
}
