package cluster

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"github.com/integr8ly/integreatly-operator/utils"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterVersionTestScenario struct {
	Name           string
	FakeSigsClient k8sclient.Client
	ExpectedError  string
	ExpectedValue  bool
}

var version1 = &configv1.ClusterVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    metav1.Time{},
				CompletionTime: nil,
				Version:        "4.9.0-rc123",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

var version2 = &configv1.ClusterVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    metav1.Time{},
				CompletionTime: nil,
				Version:        "4.8.0-rc123",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

var version3 = &configv1.ClusterVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    metav1.Time{},
				CompletionTime: nil,
				Version:        "10.8.0-rc123",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

var version4 = &configv1.ClusterVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name: "wrongname",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    metav1.Time{},
				CompletionTime: nil,
				Version:        "10.8.0-rc123",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

var version5 = &configv1.ClusterVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    metav1.Time{},
				CompletionTime: nil,
				Version:        "fakeversion",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

var version6 = &configv1.ClusterVersion{
	ObjectMeta: metav1.ObjectMeta{
		Name: "version",
	},
	Status: configv1.ClusterVersionStatus{
		History: []configv1.UpdateHistory{
			{
				State:          "",
				StartedTime:    metav1.Time{},
				CompletionTime: nil,
				Version:        "string1.string2.string3",
				Image:          "",
				Verified:       false,
			},
		},
	},
}

func TestVersions(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []ClusterVersionTestScenario{
		{
			Name:           "Test cluster versions 4.9",
			FakeSigsClient: utils.NewTestClient(scheme, version1),
			ExpectedError:  "",
			ExpectedValue:  false,
		},
		{
			Name:           "Test cluster versions 4.8",
			FakeSigsClient: utils.NewTestClient(scheme, version2),
			ExpectedError:  "",
			ExpectedValue:  true,
		},
		{
			Name:           "Test cluster versions 10.8",
			FakeSigsClient: utils.NewTestClient(scheme, version3),
			ExpectedError:  "",
			ExpectedValue:  false,
		},
		{
			Name:           "Test when cluster CR does not exist",
			FakeSigsClient: utils.NewTestClient(scheme, version4),
			ExpectedError:  "failed to fetch version: clusterversions.config.openshift.io \"version\" not found",
			ExpectedValue:  false,
		},
		{
			Name:           "Test invalid version syntax 1",
			FakeSigsClient: utils.NewTestClient(scheme, version5),
			ExpectedError:  "Error splitting cluster version history fakeversion",
			ExpectedValue:  false,
		},
		{
			Name:           "Test invalid version syntax 2",
			FakeSigsClient: utils.NewTestClient(scheme, version6),
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

func TestGetClusterType(t *testing.T) {
	type testScenario struct {
		Name     string
		Input    *configv1.Infrastructure
		Expected string
		Error    bool
	}

	scenarios := []testScenario{
		{
			Name: "Get AWS cluster type",
			Input: &configv1.Infrastructure{
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type: configv1.AWSPlatformType,
						AWS: &configv1.AWSPlatformStatus{
							ResourceTags: []configv1.AWSResourceTag{
								{
									Key:   "red-hat-clustertype",
									Value: "OSD",
								},
							},
						},
					},
				},
			},
			Expected: "OSD",
			Error:    false,
		},
		{
			Name: "Get AWS error on cluster type",
			Input: &configv1.Infrastructure{
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type: configv1.AWSPlatformType,
						AWS: &configv1.AWSPlatformStatus{
							ResourceTags: []configv1.AWSResourceTag{
								{
									Key:   "Missing Key",
									Value: "OSD",
								},
							},
						},
					},
				},
			},
			Expected: "",
			Error:    true,
		},
		{
			Name: "Get Unknown on cluster type and Error",
			Input: &configv1.Infrastructure{
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type: "Unknown Type",
						AWS: &configv1.AWSPlatformStatus{
							ResourceTags: []configv1.AWSResourceTag{
								{
									Key:   "Missing Key",
									Value: "OSD",
								},
							},
						},
					},
				},
			},
			Expected: "",
			Error:    true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			actual, err := GetClusterType(scenario.Input)

			if actual != scenario.Expected {
				t.Fatalf("Test: %s; Infrastructure does not contain the expected result; Actual: %s, Expected: %s", scenario.Name, actual, scenario.Expected)
			}

			if scenario.Error && err == nil {
				t.Fatalf("Test: %s; Failed to raise error when error was expected", scenario.Name)
			}
		})
	}
}

func TestGetClusterVersionCR(t *testing.T) {
	type args struct {
		ctx          context.Context
		serverClient k8sclient.Client
	}

	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		args    args
		want    *configv1.ClusterVersion
		wantErr bool
	}{
		{
			name: "Cluster version exists",
			args: args{
				ctx: context.TODO(),
				serverClient: utils.NewTestClient(scheme,
					&configv1.ClusterVersion{
						TypeMeta: metav1.TypeMeta{
							Kind:       "ClusterVersion",
							APIVersion: "config.openshift.io/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "version",
						},
						Spec:   configv1.ClusterVersionSpec{},
						Status: configv1.ClusterVersionStatus{},
					},
				),
			},
			want: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "version",
					ResourceVersion: "999",
				},
				Spec:   configv1.ClusterVersionSpec{},
				Status: configv1.ClusterVersionStatus{},
			},
			wantErr: false,
		},
		{
			name: "Cluster version does not exists",
			args: args{
				ctx: context.TODO(),
				serverClient: utils.NewTestClient(scheme,
					&configv1.ClusterVersion{
						TypeMeta: metav1.TypeMeta{
							Kind:       "ClusterVersion",
							APIVersion: "config.openshift.io/v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "does not exist",
						},
						Spec:   configv1.ClusterVersionSpec{},
						Status: configv1.ClusterVersionStatus{},
					},
				),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetClusterVersionCR(tt.args.ctx, tt.args.serverClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClusterVersionCR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClusterVersionCR() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetExternalClusterId(t *testing.T) {
	type args struct {
		cr *configv1.ClusterVersion
	}
	tests := []struct {
		name    string
		args    args
		want    configv1.ClusterID
		wantErr bool
	}{
		{
			name: "Found external cluster ID",
			args: args{
				cr: &configv1.ClusterVersion{
					Spec: configv1.ClusterVersionSpec{
						ClusterID: "clusterID",
					},
				},
			},
			want:    "clusterID",
			wantErr: false,
		},
		{
			name: "External cluster ID not found",
			args: args{
				cr: &configv1.ClusterVersion{
					Spec: configv1.ClusterVersionSpec{},
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetExternalClusterId(tt.args.cr)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetExternalClusterId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetExternalClusterId() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetClusterVersion(t *testing.T) {
	type args struct {
		cr *configv1.ClusterVersion
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Cluster Version found",
			args: args{
				cr: &configv1.ClusterVersion{
					Status: configv1.ClusterVersionStatus{
						Desired: configv1.Release{
							Version: "4.10.3",
						},
					},
				},
			},
			want:    "4.10.3",
			wantErr: false,
		},
		{
			name: "Cluster version not found",
			args: args{
				cr: &configv1.ClusterVersion{
					Status: configv1.ClusterVersionStatus{
						Desired: configv1.Release{
							Version: "",
						},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetClusterVersion(tt.args.cr)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClusterVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetClusterVersion() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func getLogger() l.Logger {
	return l.NewLoggerWithContext(l.Fields{})
}
