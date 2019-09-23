package mobilesecurityservicedb

import (
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
)

const (
	size                  = 1
	databaseName          = "mobile_security_service"
	databasePassword      = "postgres"
	databaseUser          = "postgresql"
	databaseNameParam     = "POSTGRESQL_DATABASE"
	databasePasswordParam = "POSTGRESQL_PASSWORD"
	databaseUserParam     = "POSTGRESQL_USER"
	//The imaged used in this project is from Red Hat. See more in https://docs.okd.io/latest/using_images/db_images/postgresql.html
	image                  = "centos/postgresql-96-centos7"
	containerName          = "database"
	databasePort           = 5432
	databaseMemoryLimit    = "512Mi"
	databaseMemoryRequest  = "512Mi"
	databaseStorageRequest = "1Gi"
)

// addMandatorySpecsDefinitions will add the specs which are mandatory for Mobile Security Service CR in the case them
// not be applied
func addMandatorySpecsDefinitions(db *mobilesecurityservicev1alpha1.MobileSecurityServiceDB) {

	/*
	   CR DB Resource
	   ---------------------
	*/

	if db.Spec.Size == 0 {
		db.Spec.Size = size
	}

	/*
		Environment Variables
		---------------------
		The following values are used to create the ConfigMap and the Environment Variables which will use these values
		These values are used for both the Mobile Security Service and its Database
	*/

	if db.Spec.DatabaseName == "" {
		db.Spec.DatabaseName = databaseName
	}

	if db.Spec.DatabasePassword == "" {
		db.Spec.DatabasePassword = databasePassword
	}

	if db.Spec.DatabaseUser == "" {
		db.Spec.DatabaseUser = databaseUser
	}

	/*
	   Database Container
	   ---------------------------------
	*/

	//Following are the values which will be used as the key label for the environment variable of the database image.
	if db.Spec.DatabaseNameParam == "" {
		db.Spec.DatabaseNameParam = databaseNameParam
	}

	if db.Spec.DatabasePasswordParam == "" {
		db.Spec.DatabasePasswordParam = databasePasswordParam
	}

	if db.Spec.DatabaseUserParam == "" {
		db.Spec.DatabaseUserParam = databaseUserParam
	}

	if db.Spec.Image == "" {
		db.Spec.Image = image
	}

	if db.Spec.ContainerName == "" {
		db.Spec.ContainerName = containerName
	}

	if db.Spec.DatabaseMemoryLimit == "" {
		db.Spec.DatabaseMemoryLimit = databaseMemoryLimit
	}

	if db.Spec.DatabaseMemoryRequest == "" {
		db.Spec.DatabaseMemoryRequest = databaseMemoryRequest
	}

	if db.Spec.DatabaseStorageRequest == "" {
		db.Spec.DatabaseStorageRequest = databaseStorageRequest
	}

	if db.Spec.DatabasePort == 0 {
		db.Spec.DatabasePort = databasePort
	}
}
