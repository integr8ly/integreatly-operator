package resources

import (
	"github.com/hashicorp/go-version"
	errorUtil "github.com/pkg/errors"
)

func VerifyVersionUpgradeNeeded(currentVersion string, desiredVersion string) (bool, error) {
	current, err := version.NewVersion(currentVersion)

	if err != nil {
		return false, errorUtil.Wrap(err, "failed to parse current version")
	}
	desired, err := version.NewVersion(desiredVersion)

	if err != nil {
		return false, errorUtil.Wrap(err, "failed to parse desired version")
	}

	return current.LessThan(desired), nil
}
