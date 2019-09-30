package resources

import (
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strconv"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func NewVersion(version string) (*Version, error) {
	r, _ := regexp.Compile(`^[Vv]?([0-9]+)\.([0-9]+)(\.|\-)([0-9]+)$`)
	matches := r.FindStringSubmatch(version)
	if len(matches) < 5 {
		return nil, errors.New("invalid version")
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[4])
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
