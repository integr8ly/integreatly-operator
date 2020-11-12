package resources

import uuid "github.com/nu7hatch/gouuid"

func GetUUID() string {
	u, _ := uuid.NewV4()
	return u.String()[0:6]
}
