package util

// MergeMaps merges a list maps into the first one. B overrides A if keys collide.
func MergeMaps(base map[string]string, merges ...map[string]string) map[string]string {
	for _, m := range merges {
		for key, value := range m {
			base[key] = value
		}
	}
	return base
}
