package resources

import (
	"time"
)

func SafeTimeDereference(t *time.Time) time.Time {
	if t != nil {
		return *t
	}
	return time.Time{}
}
