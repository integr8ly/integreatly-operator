package config

const (
	OboNamespaceSuffix = "-observability"
)

func GetOboNamespace(installationNamespace string) string {
	return installationNamespace + OboNamespaceSuffix
}
