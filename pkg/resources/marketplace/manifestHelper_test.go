package marketplace

import (
	"os"
	"testing"
)

func TestGetManifestDirEnvVar(t *testing.T) {

	scenarios := []struct {
		Name        string
		SetEnvVar   bool
		ExpectedDir string
	}{
		{
			Name:        "Test default manifest dir is used if env var is not defined",
			SetEnvVar:   false,
			ExpectedDir: defaultManifestDir,
		},
		{
			Name:        "Test env var manifest dir is used if env var is defined",
			SetEnvVar:   true,
			ExpectedDir: "test/manifest",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if scenario.SetEnvVar {
				os.Setenv(manifestEnvVarKey, scenario.ExpectedDir)
			}

			manifestDir := GetManifestDirEnvVar()
			if manifestDir != scenario.ExpectedDir {
				t.Fatalf("Expected %v but got %v", manifestDir, scenario.ExpectedDir)
			}
		})
	}
}

func TestGetCurrentCSVFromManifest(t *testing.T) {

	scenarios := []struct {
		Name              string
		PackageYamlString string
		ExpectedVer       string
		ExpectedErr       bool
	}{
		{
			Name: "Test 1 - get current csv from packyaml string",
			PackageYamlString: `packageName: integreatly-3scale
                                channels:
                                - name: integreatly
                                currentCSV: 3scale.1.2.2`,
			ExpectedVer: "1.2.2",
			ExpectedErr: false,
		},
		{
			Name: "Test 2 - test get version with a v and large version numbers in string",
			PackageYamlString: `packageName: integreatly
                                channels:
                                - name: integreatly
                                currentCSV: integrealty-operator.v12.2321.1237`,
			ExpectedVer: "12.2321.1237",
			ExpectedErr: false,
		},
		{
			Name: "Test 3 - test get version with a - in string ",
			PackageYamlString: `packageName: integreatly-3scale
                                channels:
                                - name: integreatly
                                currentCSV: 3scale-operator.v1.9-7`,
			ExpectedVer: "1.9.7",
			ExpectedErr: false,
		},
		{
			Name: "Test 4 - Error due to invalid version with too many dots",
			PackageYamlString: `channels:
                                - currentCSV: cloud-resources.v10.9.8.7.6
                                name: integreatly
                                defaultChannel: integreatly
                                packageName: integreatly-cloud-resources`,
			ExpectedErr: true,
		},
		{
			Name:              "Test 5 - Error with invalid string entirely",
			PackageYamlString: "Bad String",
			ExpectedErr:       true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			currentCsvVer, err := GetCurrentCSVFromManifest(scenario.PackageYamlString)
			if scenario.ExpectedErr && err == nil {
				t.Fatalf("Expected error but got none")
			}

			if currentCsvVer != scenario.ExpectedVer {
				t.Fatalf("Expected %v but got %v", scenario.ExpectedVer, currentCsvVer)
			}
		})
	}
}
