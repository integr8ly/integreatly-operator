package resources

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	errorUtil "github.com/pkg/errors"
)

const (
	EnvForceReconcileTimeout = "ENV_FORCE_RECONCILE_TIMEOUT"
)

// returns envar for reconcile time else returns default time
func GetForcedReconcileTimeOrDefault(defaultTo time.Duration) time.Duration {
	recTime, exist := os.LookupEnv(EnvForceReconcileTimeout)
	if exist {
		rt, err := strconv.ParseInt(recTime, 10, 64)
		if err != nil {
			return defaultTo
		}
		return time.Duration(rt)
	}
	return defaultTo
}

func GeneratePassword() (string, error) {
	generatedPassword, err := uuid.NewRandom()
	if err != nil {
		return "", errorUtil.Wrap(err, "error generating password")
	}
	return strings.Replace(generatedPassword.String(), "-", "", 10), nil
}
