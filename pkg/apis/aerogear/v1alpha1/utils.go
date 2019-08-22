package v1alpha1

func ContainsClient(kcc []*KeycloakClient, id string) bool {
	for _, a := range kcc {
		if a.ID == id {
			return true
		}
	}
	return false
}
