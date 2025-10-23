package resources

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func NewVersion(version integreatlyv1alpha1.OperatorVersion) (*Version, error) {
	r, err := regexp.Compile(`^[Vv]?([0-9]+)\.([0-9]+)(\.|\-)([0-9]+)$`)
	if err != nil {
		return nil, err
	}

	matches := r.FindStringSubmatch(string(version))
	if len(matches) < 5 {
		return nil, errors.New("invalid version")
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, err
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, err
	}
	patch, err := strconv.Atoi(matches[4])
	if err != nil {
		return nil, err
	}

	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

func (v *Version) Equals(other *Version) bool {
	return v.Major == other.Major && v.Minor == other.Minor && v.Patch == other.Patch
}

func (v *Version) IsNewerThan(other *Version) bool {
	return v.Major > other.Major || (v.Major == other.Major && v.Minor > other.Minor) || (v.Major == other.Major && v.Minor == other.Minor && v.Patch > other.Patch)
}

func (v *Version) AsString() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
