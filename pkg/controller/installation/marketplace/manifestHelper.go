package marketplace

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	manifestEnvVarKey  = "MANIFEST_DIR"
	defaultManifestDir = "manifests"
)

// Generates ConfigMap equivalent from a manifest package
func GenerateRegistryConfigMapFromManifest(manifestPackageName string) (map[string]string, error) {
	manifestDir := fmt.Sprintf("%s/%s", GetManifestDirEnvVar(), manifestPackageName)

	configMapData := make(map[string]string)

	csvStringList, err := GetFilesFromManifestAsStringList(manifestDir, "^*.clusterserviceversion.yaml$", "")
	if err != nil {
		logrus.Fatalf("Error proccessing cluster service versions from %s with error: %s", manifestPackageName, err)
		return nil, err
	}

	packageStringList, err := GetFilesFromManifestAsStringList(manifestDir, "^*.package.yaml$", "")
	if err != nil {
		logrus.Fatalf("Error proccessing csv packages from %s with error: %s", manifestPackageName, err)
		return nil, err
	}

	crdStringList, err := GetFilesFromManifestAsStringList(manifestDir, "^*.crd.yaml$", packageStringList)
	if err != nil {
		logrus.Fatalf("Error proccessing crds from %s with error: %s", manifestPackageName, err)
		return nil, err
	}

	configMapData["clusterServiceVersions"] = csvStringList
	configMapData["customResourceDefinitions"] = crdStringList
	configMapData["packages"] = packageStringList

	return configMapData, nil
}

// Get manifest files from a directory recursively matching a regex and return as a yaml string list
func GetFilesFromManifestAsStringList(dir string, regex string, packageYaml string) (string, error) {
	var stringList strings.Builder
	libRegEx, e := regexp.Compile(regex)
	if e != nil {
		logrus.Fatalf("Error compiling regex for registry file: %s", e)
	}

	e = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if packageYaml != "" { // PackageYaml would not be empty string if getting CRD - only want to get the CRDs from the package of the currentCSV version
			version, err := GetCurrentCSVFromManifest(packageYaml)

			if err == nil && libRegEx.MatchString(info.Name()) && strings.Contains(path, version) {
				return ProcessYamlFile(path, &stringList)
			}

			return err
		} else { // Otherwise find all matching files and process to a string list
			if err == nil && libRegEx.MatchString(info.Name()) {
				return ProcessYamlFile(path, &stringList)
			}
		}

		return nil
	})

	return stringList.String(), e
}

func ProcessYamlFile(path string, stringList *strings.Builder) error {
	yamlString, err := ReadAndFormatManifestYamlFile(path)

	stringList.WriteString(yamlString)

	return err
}

// Process each line of files from manifest for correct yaml format to use in config map
func ReadAndFormatManifestYamlFile(path string) (string, error) {
	var formattedString strings.Builder

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	linesSplit := strings.Split(string(content), "\n")

	for i := 0; i < len(linesSplit); i++ {
		// For the first line check and append - at start of line
		if i == 0 {
			if !strings.HasPrefix(linesSplit[i], "- ") {
				formattedString.WriteString("- ")
			}
		} else {
			// Want to then correctly indent remaining lines for correct yaml formatting
			formattedString.WriteString("  ")
		}

		formattedString.WriteString(linesSplit[i])
		formattedString.WriteString("\n")

	}

	return formattedString.String(), nil
}

// Gets the version number from the package yaml string
func GetCurrentCSVFromManifest(packageYaml string) (string, error) {
	r, _ := regexp.Compile(`[a-zA-Z]\.[Vv]?([0-9]+)\.([0-9]+)(\.|\-)([0-9]+)($|\n)`)
	matches := r.FindStringSubmatch(packageYaml)
	if len(matches) < 5 {
		return "", errors.New("Invalid csv version from manifest package")
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[4])

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

// Get the manifest directory for when running locally vs when in container image
func GetManifestDirEnvVar() string {
	if envVar := os.Getenv(manifestEnvVarKey); envVar != "" {
		logrus.Infof("Using env var manifest dir: %s", envVar)
		return envVar
	}

	logrus.Info("Using default manifest package dir")
	return defaultManifestDir
}
