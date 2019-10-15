package v1alpha1

import (
	"k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MobileSecurityServiceBackupSpec defines the desired state of MobileSecurityServiceBackup
// +k8s:openapi-gen=true
type MobileSecurityServiceBackupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// Schedule period for the CronJob  "0 0 * * *" # daily at 00:00.
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	Schedule string `json:"schedule,omitempty"`

	// Image:tag used to do the backup.
	// More Info: https://github.com/integr8ly/backup-container-image
	Image string `json:"image,omitempty"`

	// Database version. (E.g 9.6).
	// IMPORTANT: Just the first 2 digits should be used.
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	DatabaseVersion string `json:"databaseVersion,omitempty"`

	// Used to create the directory where the files will be stored
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	ProductName string `json:"productName,omitempty"`

	// Name of AWS S3 storage.
	// Required to create the Secret with the data to allow send the backup files to AWS S3 storage.
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	AwsS3BucketName string `json:"awsS3BucketName,omitempty"`

	// Key ID of AWS S3 storage.
	// Required to create the Secret with the data to allow send the backup files to AWS S3 storage.
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	AwsAccessKeyId string `json:"awsAccessKeyId,omitempty"`

	// Secret/Token of AWS S3 storage.
	// Required to create the Secret with the data to allow send the backup files to AWS S3 storage.
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	AwsSecretAccessKey string `json:"awsSecretAccessKey,omitempty"`

	// Name of the secret with the AWS data credentials already created in the cluster
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	AwsCredentialsSecretName string `json:"awsCredentialsSecretName,omitempty"`
	// Name of the namespace where the scret with the AWS data credentials is in the cluster
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	AwsCredentialsSecretNamespace string `json:"awsCredentialsSecretNamespace,omitempty"`

	// Name of the secret with the EncryptionKey data already created in the cluster
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	EncryptionKeySecretName string `json:"encryptionKeySecretName,omitempty"`

	// Name of the namespace where the secret with the EncryptionKey data is in the cluster
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	EncryptionKeySecretNamespace string `json:"encryptionKeySecretNamespace,omitempty"`

	// GPG public key to create the EncryptionKeySecret with this data
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	// See here how to create this key : https://help.github.com/en/articles/generating-a-new-gpg-key
	GpgPublicKey string `json:"gpgPublicKey,omitempty"`

	// GPG email to create the EncryptionKeySecret with this data
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	// See here how to create this key : https://help.github.com/en/articles/generating-a-new-gpg-key
	GpgEmail string `json:"gpgEmail,omitempty"`

	// GPG trust model to create the EncryptionKeySecret with this data. the default value is true when it is empty.
	// More info: https://github.com/aerogear/mobile-security-service-operator#configuring-the-backup-service
	// See here how to create this key : https://help.github.com/en/articles/generating-a-new-gpg-key
	GpgTrustModel string `json:"gpgTrustModel,omitempty"`
}

// MobileSecurityServiceBackupStatus defines the observed state of MobileSecurityServiceBackup
// +k8s:openapi-gen=true
type MobileSecurityServiceBackupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.htm

	// Will be as "OK when all objects are created successfully
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	BackupStatus string `json:"backupStatus"`
	// Name of the CronJob object created and managed by it to schedule the backup job
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	CronJobName string `json:"cronJobName"`
	// Name of the secret object created with the database data to allow the backup image connect to the database
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	DBSecretName string `json:"dbSecretName"`
	// Data  of the secret object created with the database data to allow the backup image connect to the database
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	DBSecretData map[string]string `json:"dbSecretData"`
	// Name  of the secret object with the Aws data to allow send the backup files to the AWS storage
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	AWSSecretName string `json:"awsSecretName"`
	// Data  of the secret object with the Aws data to allow send the backup files to the AWS storage
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	AWSSecretData map[string]string `json:"awsSecretData"`
	// Namespace  of the secret object with the Aws data to allow send the backup files to the AWS storage
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	AwsCredentialsSecretNamespace string `json:"awsCredentialsSecretNamespace"`
	// Name  of the secret object with the Encryption GPG Key
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-type
	EncryptionKeySecretName string `json:"encryptionKeySecretName"`
	// Namespace of the secret object with the Encryption GPG Key
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	EncryptionKeySecretNamespace string `json:"encryptionKeySecretNamespace"`
	// Data of the secret object with the Encryption GPG Key
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	EncryptionKeySecretData map[string]string `json:"encryptionKeySecretData"`
	// Boolean value which has true when it has an EncryptionKey to be used to send the backup files
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	HasEncryptionKey bool `json:"hasEncryptionKey"`
	// Boolean value which has true when the Database Pod was found in order to create the secret with the database data to allow the backup image connect into it.
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	DatabasePodFound bool `json:"databasePodFound"`
	// Boolean value which has true when the Service Database Pod was found in order to create the secret with the database data to allow the backup image connect into it.
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	DatabaseServiceFound bool `json:"databaseServiceFound"`
	// Status of the CronJob object
	// More info: https://github.com/aerogear/mobile-security-service-operator#status-definition-per-types
	CronJobStatus v1beta1.CronJobStatus `json:"cronJobStatus"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceBackup is the Schema for the mobilesecurityservicedbbackups API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type MobileSecurityServiceBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MobileSecurityServiceBackupSpec   `json:"spec,omitempty"`
	Status MobileSecurityServiceBackupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MobileSecurityServiceBackupList contains a list of MobileSecurityServiceBackup
type MobileSecurityServiceBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MobileSecurityServiceBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MobileSecurityServiceBackup{}, &MobileSecurityServiceBackupList{})
}
