package resources

import (
	"context"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	configv1 "github.com/openshift/api/config/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strconv"
	"testing"
)

type ClusterVersionTestScenario struct {
	Name           string
	FakeSigsClient k8sclient.Client
	logger         l.Logger
	ExpectedError  string
	ExpectedValue  bool
}

var version1 = &configv1.ClusterVersion{
	ObjectMeta: v1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    v1.Time{},
				CompletionTime: nil,
				Version:        "4.9.0-rc123",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

var version2 = &configv1.ClusterVersion{
	ObjectMeta: v1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    v1.Time{},
				CompletionTime: nil,
				Version:        "4.8.0-rc123",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

var version3 = &configv1.ClusterVersion{
	ObjectMeta: v1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    v1.Time{},
				CompletionTime: nil,
				Version:        "10.8.0-rc123",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

var version4 = &configv1.ClusterVersion{
	ObjectMeta: v1.ObjectMeta{
		Name: "wrongname",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    v1.Time{},
				CompletionTime: nil,
				Version:        "10.8.0-rc123",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

var version5 = &configv1.ClusterVersion{
	ObjectMeta: v1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    v1.Time{},
				CompletionTime: nil,
				Version:        "fakeversion",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

var version6 = &configv1.ClusterVersion{
	ObjectMeta: v1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    v1.Time{},
				CompletionTime: nil,
				Version:        "string1.string2.string3",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

func getClusterBuildScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	err := configv1.AddToScheme(scheme)
	return scheme, err
}

func TestVersions(t *testing.T) {
	scheme, err := getClusterBuildScheme()
	if err != nil {
		t.Fatalf("Error creating build scheme")
	}

	scenarios := []ClusterVersionTestScenario{
		{
			Name:           "Test cluster versions 4.9",
			FakeSigsClient: fake.NewFakeClientWithScheme(scheme, version1),
			ExpectedError:  "",
			ExpectedValue:  false,
		},
		{
			Name:           "Test cluster versions 4.8",
			FakeSigsClient: fake.NewFakeClientWithScheme(scheme, version2),
			ExpectedError:  "",
			ExpectedValue:  true,
		},
		{
			Name:           "Test cluster versions 10.8",
			FakeSigsClient: fake.NewFakeClientWithScheme(scheme, version3),
			ExpectedError:  "",
			ExpectedValue:  false,
		},
		{
			Name:           "Test when cluster CR does not exist",
			FakeSigsClient: fake.NewFakeClientWithScheme(scheme, version4),
			ExpectedError:  "failed to fetch version: clusterversions.config.openshift.io \"version\" not found",
			ExpectedValue:  false,
		},
		{
			Name:           "Test invalid version syntax 1",
			FakeSigsClient: fake.NewFakeClientWithScheme(scheme, version5),
			ExpectedError:  "Error splitting cluster version history fakeversion",
			ExpectedValue:  false,
		},
		{
			Name:           "Test invalid version syntax 2",
			FakeSigsClient: fake.NewFakeClientWithScheme(scheme, version6),
			ExpectedError:  "Error parsing cluster version string1.string2",
			ExpectedValue:  false,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			before, err := ClusterVersionBefore49(context.TODO(), s.FakeSigsClient, getLogger())

			if s.ExpectedError != "" && err == nil {
				t.Fatal("Expected an error " + s.ExpectedError)
			}

			if s.ExpectedError != "" && s.ExpectedError != err.Error() {
				t.Fatal("Wrong error returned, expected: " + s.ExpectedError + " got: " + err.Error())
			}

			if s.ExpectedError == "" && s.ExpectedValue != before {
				t.Fatal("Wrong result returned, expected: " + strconv.FormatBool(s.ExpectedValue) + " got: " + strconv.FormatBool(before))
			}
		})
	}
}
