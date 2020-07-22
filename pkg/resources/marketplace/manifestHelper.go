package marketplace

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	manifestEnvVarKey  = "MANIFEST_DIR"
	defaultManifestDir = "manifests"
)

type CrdConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Group string `yaml:"group"`
		Names struct {
			Kind     string `yaml:"kind"`
			ListKind string `yaml:"listKind"`
			Plural   string `yaml:"plural"`
			Singular string `yaml:"singular"`
		} `yaml:"names"`
	} `yaml:"spec"`
}

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

	crdStringList, err := GetCRDsFromManifestAsStringList(manifestDir, "^*.crd.yaml$", packageStringList, manifestPackageName)
	if err != nil {
		logrus.Fatalf("Error proccessing crds from %s with error: %s", manifestPackageName, err)
		return nil, err
	}

	configMapData["clusterServiceVersions"] = csvStringList
	configMapData["customResourceDefinitions"] = crdStringList
	configMapData["packages"] = packageStringList

	return configMapData, nil
}

func GetCRDsFromManifestAsStringList(dir string, regex string, packageYaml string, manifestPackageName string) (string, error) {
	var stringList strings.Builder
	libRegEx, e := regexp.Compile(regex)
	if packageYaml != "" {
		if e != nil {
			logrus.Fatalf("Error compiling regex for registry file: %s", e)
		}

		var folders []string

		// Get a list of fodlers in the manifest/<product> folder
		e = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() && info.Name() != manifestPackageName {
				folders = append(folders, path)
			}
			return nil
		})

		ReverseSlice(folders)

		var currentVersion string
		var err error

		// Get current version from package Yaml
		if packageYaml != "" {
			currentVersion, err = GetCurrentCSVFromManifest(packageYaml)
		}
		if err != nil {
			logrus.Fatal("Error")
		}

		// iterate through all folders
		for _, currentFolder := range folders {

			// add current version crds
			if strings.Contains(currentFolder, currentVersion) {
				e = filepath.Walk(currentFolder, func(path string, info os.FileInfo, err error) error {
					if err == nil && libRegEx.MatchString(info.Name()) && strings.Contains(path, currentVersion) {
						return ProcessYamlFile(path, &stringList)
					}
					return nil
				})

			} else {
				// iterate all other versions look for crds that don't exist in the current version
				// and add them to the stringlist for building the config map
				files, err := ioutil.ReadDir(currentFolder)
				if err == nil {
					for _, f := range files {
						// iterate through all fils in the folder
						if libRegEx.MatchString(f.Name()) {
							var crdConfig CrdConfig
							GetCrdDetails(&crdConfig, currentFolder, f)
							found := CheckFoldersForMatch(folders, currentFolder, &crdConfig, libRegEx)
							if !found {
								// if match isn't found, add file contents to stringlist
								ProcessYamlFile(currentFolder+string(os.PathSeparator)+f.Name(), &stringList)
							}

						}
					}
				}
			}

		}
	}
	return stringList.String(), e
}

// CheckFoldersForMatch searchs other folders for a crd with the same APIVersion, group and kind
func CheckFoldersForMatch(folders []string, currentFolder string, crdConfig *CrdConfig, libRegEx *regexp.Regexp) bool {

	found := false
	for _, folder := range folders {
		if folder == currentFolder {
			break
		} else {
			files, err := ioutil.ReadDir(folder)
			if err == nil {
				for _, f := range files {
					if libRegEx.MatchString(f.Name()) {
						var needleCrd CrdConfig
						GetCrdDetails(&needleCrd, folder, f)

						if crdConfig.APIVersion == needleCrd.APIVersion &&
							crdConfig.Spec.Group == needleCrd.Spec.Group &&
							crdConfig.Spec.Names.Kind == needleCrd.Spec.Names.Kind {
							found = true
						}
					}
				}
			}
		}
	}
	return found
}

// GetCrdDetails reads the crd file
func GetCrdDetails(crdConfig *CrdConfig, currentFolder string, f os.FileInfo) {
	yamlFile, err := ioutil.ReadFile(currentFolder + string(os.PathSeparator) + f.Name())

	err = yaml.Unmarshal(yamlFile, &crdConfig)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}
}

// ReverseSlice reverses the order of the folders list
func ReverseSlice(data interface{}) {
	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Slice {
		panic(errors.New("data must be a slice type"))
	}
	valueLen := value.Len()
	for i := 0; i <= int((valueLen-1)/2); i++ {
		reverseIndex := valueLen - 1 - i
		tmp := value.Index(reverseIndex).Interface()
		value.Index(reverseIndex).Set(value.Index(i))
		value.Index(i).Set(reflect.ValueOf(tmp))
	}
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
		}
		// Otherwise find all matching files and process to a string list
		if err == nil && libRegEx.MatchString(info.Name()) {
			return ProcessYamlFile(path, &stringList)
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
