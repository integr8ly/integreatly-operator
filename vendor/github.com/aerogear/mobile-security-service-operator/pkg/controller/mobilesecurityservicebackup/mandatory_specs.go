package mobilesecurityservicebackup

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
)

const (
	schedule        = "0 0 * * *"
	image           = "quay.io/integreatly/backup-container:1.0.8"
	databaseVersion = "9.6"
)

// addMandatorySpecsDefinitions will add the specs which are mandatory for MobileSecurityServiceBackup CR in the case them
// not be applied
func addMandatorySpecsDefinitions(bkp *mobilesecurityservicev1alpha1.MobileSecurityServiceBackup) {

	/*
		 Backup Container
		---------------------
		See https://github.com/integr8ly/backup-container-image
	*/

	if bkp.Spec.Schedule == "" {
		bkp.Spec.Schedule = schedule
	}

	if bkp.Spec.Image == "" {
		bkp.Spec.Image = image
	}

	if bkp.Spec.DatabaseVersion == "" {
		bkp.Spec.DatabaseVersion = databaseVersion
	}
}
