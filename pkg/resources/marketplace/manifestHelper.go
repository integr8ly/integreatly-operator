package marketplace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"

	"github.com/Masterminds/semver"
	"gopkg.in/yaml.v2"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	manifestEnvVarKey  = "MANIFEST_DIR"
	defaultManifestDir = "manifests"
)

// Generates ConfigMap equivalent from a manifest package
func GenerateRegistryConfigMapFromManifest(manifestPackageName string) (map[string]string, error) {
	logger := l.NewLogger()
	manifestDir := fmt.Sprintf("%s/%s", GetManifestDirEnvVar(), manifestPackageName)

	configMapData := make(map[string]string)

	csvStringList, err := GetFilesFromManifestAsStringList(manifestDir, "^*.clusterserviceversion.yaml$", "")
	if err != nil {
		logger.Fatal("Error processing cluster service versions", l.Fields{"manifestPackageName": manifestPackageName}, err)
		return nil, err
	}

	packageStringList, err := GetFilesFromManifestAsStringList(manifestDir, "^*.package.yaml$", "")
	if err != nil {
		logger.Fatal("Error processing csv packages", l.Fields{"manifestPackageName": manifestPackageName}, err)
		return nil, err
	}

	crdStringList, err := GetCRDsFromManifestAsStringList(manifestDir, "^*.crd.yaml$", packageStringList, manifestPackageName)
	if err != nil {
		logger.Fatal("Error processing crds", l.Fields{"manifestPackageName": manifestPackageName}, err)
		return nil, err
	}

	configMapData["clusterServiceVersions"] = csvStringList
	configMapData["customResourceDefinitions"] = crdStringList
	configMapData["packages"] = packageStringList

	return configMapData, nil
}

func GetCRDsFromManifestAsStringList(dir string, regex string, packageYaml string, manifestPackageName string) (string, error) {
	logger := l.NewLogger()
	var stringList strings.Builder
	libRegEx, e := regexp.Compile(regex)
	if packageYaml != "" {
		if e != nil {
			logger.Fatal("Error compiling regex for registry file:", nil, e)
		}

		var folders []string

		// Get a list of fodlers in the manifest/<product> folder
		e = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() && info.Name() != manifestPackageName {
				//folder := strings.Split(path, "/")
				folders = append(folders, info.Name())
			}
			return nil
		})

		// Get Version number from list of folders
		// We can't sort/reverse folders correctly since they are lexicographically sorted
		// meaning 2 would be greater than 10
		vs := make([]*semver.Version, len(folders))
		for i, r := range folders {
			v, err := semver.NewVersion(r)
			if err != nil {
				logger.Error("Error parsing version:", nil, err)
			}

			vs[i] = v
		}

		sort.Sort(semver.Collection(vs))

		err := ReverseSlice(vs)
		if err != nil {
			logger.Error("ReverseSlice error", nil, err)
		}

		var currentVersion string

		// Get current version from package Yaml
		if packageYaml != "" {
			currentVersion, err = GetCurrentCSVFromManifest(packageYaml)
		}
		if err != nil {
			logger.Fatal("Error getting current csv from manifest", nil, err)
		}

		// iterate through all folders
		for _, folder := range vs {

			currentFolder := dir + string(os.PathSeparator) + folder.Original()
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
				files, err := os.ReadDir(currentFolder)
				if err == nil {
					for _, f := range files {
						// iterate through all fils in the folder
						if libRegEx.MatchString(f.Name()) {
							crdConfig := &apiextensionv1.CustomResourceDefinition{}

							GetCrdDetails(crdConfig, currentFolder, f)
							found := CheckFoldersForMatch(dir, vs, currentFolder, crdConfig, libRegEx)
							if !found {
								// if match isn't found, add file contents to stringlist
								err = ProcessYamlFile(currentFolder+string(os.PathSeparator)+f.Name(), &stringList)
								if err != nil {
									logger.Error("contents not added to stringlist", nil, err)
								}
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
func CheckFoldersForMatch(dir string, folders []*semver.Version, currentFolder string, crdConfig *apiextensionv1.CustomResourceDefinition, libRegEx *regexp.Regexp) bool {

	found := false
	for _, cfolder := range folders {
		cfolder := dir + string(os.PathSeparator) + cfolder.Original()

		if cfolder == currentFolder {
			break
		} else {
			files, err := os.ReadDir(cfolder)
			if err == nil {
				for _, f := range files {
					if libRegEx.MatchString(f.Name()) {
						needleCrd := &apiextensionv1.CustomResourceDefinition{}
						GetCrdDetails(needleCrd, cfolder, f)

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
func GetCrdDetails(crdConfig *apiextensionv1.CustomResourceDefinition, currentFolder string, f os.DirEntry) {
	yamlFile, err := os.ReadFile(filepath.Clean(currentFolder + string(os.PathSeparator) + f.Name()))
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
	}

	err = yaml.Unmarshal(yamlFile, &crdConfig)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
	}
}

// ReverseSlice reverses the order of the folders list
func ReverseSlice(data interface{}) error {
	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Slice {
		return (errors.New("data must be a slice type"))
	}
	valueLen := value.Len()
	for i := 0; i <= int((valueLen-1)/2); i++ {
		reverseIndex := valueLen - 1 - i
		tmp := value.Index(reverseIndex).Interface()
		value.Index(reverseIndex).Set(value.Index(i))
		value.Index(i).Set(reflect.ValueOf(tmp))
	}
	return nil
}

// Get manifest files from a directory recursively matching a regex and return as a yaml string list
func GetFilesFromManifestAsStringList(dir string, regex string, packageYaml string) (string, error) {
	logger := l.NewLogger()
	var stringList strings.Builder
	libRegEx, e := regexp.Compile(regex)
	if e != nil {
		logger.Fatal("Error compiling regex for registry file", nil, e)
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
	content, err := os.ReadFile(filepath.Clean(path))
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
	r, err := regexp.Compile(`[a-zA-Z]\.[Vv]?([0-9]+)\.([0-9]+)(\.|\-)([0-9]+)($|\n)`)
	if err != nil {
		return "", err
	}
	matches := r.FindStringSubmatch(packageYaml)
	if len(matches) < 5 {
		return "", errors.New("invalid csv version from manifest package")
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", err
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return "", err
	}
	patch, err := strconv.Atoi(matches[4])
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

// Get the manifest directory for when running locally vs when in container image
func GetManifestDirEnvVar() string {
	logger := l.NewLogger()
	if envVar := os.Getenv(manifestEnvVarKey); envVar != "" {
		logger.Infof("Using env var manifest dir", l.Fields{"envVar": envVar})
		return envVar
	}

	logger.Info("Using default manifest package dir")
	return defaultManifestDir
}
