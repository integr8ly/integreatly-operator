package resources

// Checks if value given is contained in a string array
func Contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}

	return false
}
