package resources

import (
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
)

func TestVersion(t *testing.T) {
	scenarios := []struct {
		Name            string
		TestVersion     string
		ExpectedVersion *Version
		Verifier        func(version *Version, err error, t *testing.T)
	}{
		{
			Name:        "test valid version",
			TestVersion: "1.2.3",
			Verifier: func(version *Version, err error, t *testing.T) {
				if version.Major != 1 {
					t.Fatalf("major version incorrect, expected %s, got %d", "1", version.Major)
				}
				if version.Minor != 2 {
					t.Fatalf("minor version incorrect, expected %s, got %d", "2", version.Minor)
				}
				if version.Patch != 3 {
					t.Fatalf("patch version incorrect, expected %s, got %d", "3", version.Patch)
				}
			},
		},
		{
			Name:        "test valid version starting with v",
			TestVersion: "v1.2.3",
			Verifier: func(version *Version, err error, t *testing.T) {
				if version.Major != 1 {
					t.Fatalf("major version incorrect, expected %s, got %d", "1", version.Major)
				}
				if version.Minor != 2 {
					t.Fatalf("minor version incorrect, expected %s, got %d", "2", version.Minor)
				}
				if version.Patch != 3 {
					t.Fatalf("patch version incorrect, expected %s, got %d", "3", version.Patch)
				}
			},
		},
		{
			Name:        "test valid version starting with capital V",
			TestVersion: "V1.2.3",
			Verifier: func(version *Version, err error, t *testing.T) {
				if version.Major != 1 {
					t.Fatalf("major version incorrect, expected %s, got %d", "1", version.Major)
				}
				if version.Minor != 2 {
					t.Fatalf("minor version incorrect, expected %s, got %d", "2", version.Minor)
				}
				if version.Patch != 3 {
					t.Fatalf("patch version incorrect, expected %s, got %d", "3", version.Patch)
				}
			},
		},
		{
			Name:        "test invalid version starting with non-v character",
			TestVersion: "ga1.2.3",
			Verifier: func(version *Version, err error, t *testing.T) {
				if err == nil {
					t.Fatalf("did not get expected error")
				}
			},
		},
		{
			Name:        "test version with many dots is invalid",
			TestVersion: "1.2.3.4.5.6",
			Verifier: func(version *Version, err error, t *testing.T) {
				if err == nil {
					t.Fatalf("did not get expected error")
				}
			},
		},
		{
			Name:        "test invalid version",
			TestVersion: "hello world",
			Verifier: func(version *Version, err error, t *testing.T) {
				if err == nil {
					t.Fatalf("did not get expected error")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			version, err := NewVersion(integreatlyv1alpha1.OperatorVersion(scenario.TestVersion))
			scenario.Verifier(version, err, t)
		})
	}
}

func TestComparisons(t *testing.T) {
	scenarios := []struct {
		Name     string
		V1       string
		V2       string
		Verifier func(v1, v2 *Version, t *testing.T)
	}{
		{
			Name: "patch only matches",
			V1:   "0.0.1",
			V2:   "0.0.1",
			Verifier: func(v1, v2 *Version, t *testing.T) {
				if !v1.Equals(v2) {
					t.Fatalf("expected %s to equal %s", v1.AsString(), v2.AsString())
				}
				if !v2.Equals(v1) {
					t.Fatalf("expected %s to equal %s", v2.AsString(), v1.AsString())
				}
				if v1.IsNewerThan(v2) {
					t.Fatalf("did not expect %s to be newer than %s", v1.AsString(), v2.AsString())
				}
				if v2.IsNewerThan(v1) {
					t.Fatalf("did not expect %s to be newer than %s", v2.AsString(), v1.AsString())
				}
			},
		},
		{
			Name: "major only matches",
			V1:   "0.1.0",
			V2:   "0.1.0",
			Verifier: func(v1, v2 *Version, t *testing.T) {
				if !v1.Equals(v2) {
					t.Fatalf("expected %s to equal %s", v1.AsString(), v2.AsString())
				}
				if !v2.Equals(v1) {
					t.Fatalf("expected %s to equal %s", v2.AsString(), v1.AsString())
				}
				if v1.IsNewerThan(v2) {
					t.Fatalf("did not expect %s to be newer than %s", v1.AsString(), v2.AsString())
				}
				if v2.IsNewerThan(v1) {
					t.Fatalf("did not expect %s to be newer than %s", v2.AsString(), v1.AsString())
				}
			},
		},
		{
			Name: "major only matches",
			V1:   "1.0.0",
			V2:   "1.0.0",
			Verifier: func(v1, v2 *Version, t *testing.T) {
				if !v1.Equals(v2) {
					t.Fatalf("expected %s to equal %s", v1.AsString(), v2.AsString())
				}
				if !v2.Equals(v1) {
					t.Fatalf("expected %s to equal %s", v2.AsString(), v1.AsString())
				}
				if v1.IsNewerThan(v2) {
					t.Fatalf("did not expect %s to be newer than %s", v1.AsString(), v2.AsString())
				}
				if v2.IsNewerThan(v1) {
					t.Fatalf("did not expect %s to be newer than %s", v2.AsString(), v1.AsString())
				}
			},
		},
		{
			Name: "patch differences found",
			V1:   "0.0.1",
			V2:   "0.0.2",
			Verifier: func(v1, v2 *Version, t *testing.T) {
				if v1.Equals(v2) {
					t.Fatalf("expected %s to not equal %s", v1.AsString(), v2.AsString())
				}
				if v2.Equals(v1) {
					t.Fatalf("expected %s to not equal %s", v2.AsString(), v1.AsString())
				}
				if v1.IsNewerThan(v2) {
					t.Fatalf("did not expect %s to be newer than %s", v1.AsString(), v2.AsString())
				}
				if !v2.IsNewerThan(v1) {
					t.Fatalf("expected %s to be newer than %s", v2.AsString(), v1.AsString())
				}
			},
		},
		{
			Name: "minor differences found",
			V1:   "0.1.0",
			V2:   "0.2.0",
			Verifier: func(v1, v2 *Version, t *testing.T) {
				if v1.Equals(v2) {
					t.Fatalf("expected %s to not equal %s", v1.AsString(), v2.AsString())
				}
				if v2.Equals(v1) {
					t.Fatalf("expected %s to not equal %s", v2.AsString(), v1.AsString())
				}
				if v1.IsNewerThan(v2) {
					t.Fatalf("did not expect %s to be newer than %s", v1.AsString(), v2.AsString())
				}
				if !v2.IsNewerThan(v1) {
					t.Fatalf("expected %s to be newer than %s", v2.AsString(), v1.AsString())
				}
			},
		},
		{
			Name: "major differences found",
			V1:   "1.0.0",
			V2:   "2.0.0",
			Verifier: func(v1, v2 *Version, t *testing.T) {
				if v1.Equals(v2) {
					t.Fatalf("expected %s to not equal %s", v1.AsString(), v2.AsString())
				}
				if v2.Equals(v1) {
					t.Fatalf("expected %s to not equal %s", v2.AsString(), v1.AsString())
				}
				if v1.IsNewerThan(v2) {
					t.Fatalf("did not expect %s to be newer than %s", v1.AsString(), v2.AsString())
				}
				if !v2.IsNewerThan(v1) {
					t.Fatalf("expected %s to be newer than %s", v2.AsString(), v1.AsString())
				}
			},
		},
		{
			Name: "major and minor differences found",
			V1:   "1.1.0",
			V2:   "2.0.0",
			Verifier: func(v1, v2 *Version, t *testing.T) {
				if v1.Equals(v2) {
					t.Fatalf("expected %s to not equal %s", v1.AsString(), v2.AsString())
				}
				if v2.Equals(v1) {
					t.Fatalf("expected %s to not equal %s", v2.AsString(), v1.AsString())
				}
				if v1.IsNewerThan(v2) {
					t.Fatalf("did not expect %s to be newer than %s", v1.AsString(), v2.AsString())
				}
				if !v2.IsNewerThan(v1) {
					t.Fatalf("expected %s to be newer than %s", v2.AsString(), v1.AsString())
				}
			},
		},
		{
			Name: "comparing versions with a dash against a dot",
			V1:   "1.0-0",
			V2:   "1.0.0",
			Verifier: func(v1, v2 *Version, t *testing.T) {
				if !v1.Equals(v2) {
					t.Fatalf("expected %s to equal %s", v1.AsString(), v2.AsString())
				}
				if !v2.Equals(v1) {
					t.Fatalf("expected %s to equal %s", v2.AsString(), v1.AsString())
				}
				if v1.IsNewerThan(v2) {
					t.Fatalf("did not expect %s to be newer than %s", v1.AsString(), v2.AsString())
				}
				if v2.IsNewerThan(v1) {
					t.Fatalf("did not expect %s to be newer than %s", v2.AsString(), v1.AsString())
				}
			},
		},
		{
			Name: "comparing long version numbers",
			V1:   "11231.87554564.8879879879",
			V2:   "11231.87554564.8879879879",
			Verifier: func(v1, v2 *Version, t *testing.T) {
				if !v1.Equals(v2) {
					t.Fatalf("expected %s to equal %s", v1.AsString(), v2.AsString())
				}
				if !v2.Equals(v1) {
					t.Fatalf("expected %s to equal %s", v2.AsString(), v1.AsString())
				}
				if v1.IsNewerThan(v2) {
					t.Fatalf("did not expect %s to be newer than %s", v1.AsString(), v2.AsString())
				}
				if v2.IsNewerThan(v1) {
					t.Fatalf("did not expect %s to be newer than %s", v2.AsString(), v1.AsString())
				}
			},
		},
		{
			Name: "comparing long unequal version numbers",
			V1:   "11231.87554565.8879879879",
			V2:   "11231.87554564.8879879879",
			Verifier: func(v1, v2 *Version, t *testing.T) {
				if v1.Equals(v2) {
					t.Fatalf("expected %s to not equal %s", v1.AsString(), v2.AsString())
				}
				if v2.Equals(v1) {
					t.Fatalf("expected %s to not equal %s", v2.AsString(), v1.AsString())
				}
				if !v1.IsNewerThan(v2) {
					t.Fatalf("expected %s to be newer than %s", v1.AsString(), v2.AsString())
				}
				if v2.IsNewerThan(v1) {
					t.Fatalf("did not expect %s to be newer than %s", v2.AsString(), v1.AsString())
				}
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			v1, _ := NewVersion(integreatlyv1alpha1.OperatorVersion(scenario.V1))
			v2, _ := NewVersion(integreatlyv1alpha1.OperatorVersion(scenario.V2))
			scenario.Verifier(v1, v2, t)
		})
	}
}
