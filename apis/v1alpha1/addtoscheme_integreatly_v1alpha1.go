package v1alpha1

import (
	observabilityoperator "github.com/redhat-developer/observability-operator/v3/api/v1"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	envoyconfigv1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	discoveryservicev1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	prometheusmonitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"

	consolev1 "github.com/openshift/api/console/v1"

	crov1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"

	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"

	syndesisv1beta1 "github.com/syndesisio/syndesis/install/operator/pkg/apis/syndesis/v1beta1"

	threescalev1 "github.com/3scale/3scale-operator/apis/apps/v1alpha1"

	dr "github.com/integr8ly/integreatly-operator/pkg/resources/dynamic-resources"
	keycloak "github.com/integr8ly/keycloak-client/pkg/types"
	appsv1 "github.com/openshift/api/apps/v1"
	authv1 "github.com/openshift/api/authorization/v1"
	confv1 "github.com/openshift/api/config/v1"
	imagev1 "github.com/openshift/api/image/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	templatev1 "github.com/openshift/api/template/v1"
	usersv1 "github.com/openshift/api/user/v1"
	samplesv1 "github.com/openshift/cluster-samples-operator/pkg/apis/samples/v1"
	operatorsv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apimachinery/pkg/runtime"

	customdomainv1alpha1 "github.com/openshift/custom-domains-operator/api/v1alpha1"

	cloudcredentialv1 "github.com/openshift/api/operator/v1"
)

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

func init() {
	schemeKcbuilder := runtime.NewSchemeBuilder(addKnownKcTypes)
	addKcToScheme := schemeKcbuilder.AddToScheme
	AddToSchemes.Register(addKcToScheme)
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(
		AddToSchemes,
		operatorsv1alpha1.AddToScheme,
		operatorsv1.AddToScheme,
		authv1.AddToScheme,
		chev1.SchemeBuilder.AddToScheme,
		syndesisv1beta1.SchemeBuilder.AddToScheme,
		threescalev1.SchemeBuilder.AddToScheme,
		grafanav1alpha1.SchemeBuilder.AddToScheme,
		crov1.SchemeBuilder.AddToScheme,
		routev1.AddToScheme,
		appsv1.AddToScheme,
		imagev1.AddToScheme,
		oauthv1.AddToScheme,
		templatev1.AddToScheme,
		rbacv1.SchemeBuilder.AddToScheme,
		usersv1.AddToScheme,
		confv1.AddToScheme,
		samplesv1.SchemeBuilder.AddToScheme,
		prometheusmonitoringv1.SchemeBuilder.AddToScheme,
		projectv1.AddToScheme,
		consolev1.AddToScheme,
		envoyconfigv1.SchemeBuilder.AddToScheme,
		discoveryservicev1.SchemeBuilder.AddToScheme,
		apiextensionv1beta1.SchemeBuilder.AddToScheme,
		apiextensionv1.SchemeBuilder.AddToScheme,
		observabilityoperator.SchemeBuilder.AddToScheme,
		customdomainv1alpha1.AddToScheme,
		cloudcredentialv1.AddToScheme,
	)
}

func addKnownKcTypes(scheme *runtime.Scheme) error {
	// Add kc users kind to schema
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   keycloak.KeycloakUserGroup,
		Version: keycloak.KeycloakUserVersion,
		Kind:    keycloak.KeycloakUserListKind,
	},
		dr.CreateUnstructuredListWithGVK(keycloak.KeycloakUserGroup, keycloak.KeycloakUserKind, keycloak.KeycloakUserListKind, keycloak.KeycloakUserVersion, "", ""))

	// Add kc user kind to schema
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   keycloak.KeycloakUserGroup,
		Version: keycloak.KeycloakUserVersion,
		Kind:    keycloak.KeycloakUserKind,
	},
		dr.CreateUnstructuredWithGVK(keycloak.KeycloakUserGroup, keycloak.KeycloakUserKind, keycloak.KeycloakUserVersion, "", ""))

	// Add kc kind to schema
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   keycloak.KeycloakGroup,
		Version: keycloak.KeycloakVersion,
		Kind:    keycloak.KeycloakKind,
	},
		dr.CreateUnstructuredWithGVK(keycloak.KeycloakGroup, keycloak.KeycloakKind, keycloak.KeycloakVersion, "", ""))

	// Add kc list kind to schema
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   keycloak.KeycloakGroup,
		Version: keycloak.KeycloakVersion,
		Kind:    keycloak.KeycloakListKind,
	},
		dr.CreateUnstructuredListWithGVK(keycloak.KeycloakGroup, keycloak.KeycloakKind, keycloak.KeycloakListKind, keycloak.KeycloakVersion, "", ""))

	// Add kcr kind to schema
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   keycloak.KeycloakRealmGroup,
		Version: keycloak.KeycloakRealmVersion,
		Kind:    keycloak.KeycloakRealmKind,
	},
		dr.CreateUnstructuredWithGVK(keycloak.KeycloakRealmGroup, keycloak.KeycloakRealmKind, keycloak.KeycloakRealmVersion, "", ""))

	// Add kcr list kind to schema
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   keycloak.KeycloakRealmGroup,
		Version: keycloak.KeycloakRealmVersion,
		Kind:    keycloak.KeycloakRealmListKind,
	},
		dr.CreateUnstructuredListWithGVK(keycloak.KeycloakRealmGroup, keycloak.KeycloakRealmKind, keycloak.KeycloakRealmListKind, keycloak.KeycloakRealmVersion, "", ""))

	// Add kc client kind to schema
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   keycloak.KeycloakClientGroup,
		Version: keycloak.KeycloakClientVersion,
		Kind:    keycloak.KeycloakClientKind,
	},
		dr.CreateUnstructuredWithGVK(keycloak.KeycloakClientGroup, keycloak.KeycloakClientKind, keycloak.KeycloakClientVersion, "", ""))

	// Add kc client list kind to schema
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   keycloak.KeycloakClientGroup,
		Version: keycloak.KeycloakClientVersion,
		Kind:    keycloak.KeycloakClientListKind,
	},
		dr.CreateUnstructuredListWithGVK(keycloak.KeycloakClientGroup, keycloak.KeycloakClientKind, keycloak.KeycloakClientListKind, keycloak.KeycloakClientVersion, "", ""))

	// Add kc backup kind to schema
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   keycloak.KeycloakBackupGroup,
		Version: keycloak.KeycloakBackupVersion,
		Kind:    keycloak.KeycloakBackupKind,
	},
		dr.CreateUnstructuredWithGVK(keycloak.KeycloakBackupGroup, keycloak.KeycloakBackupKind, keycloak.KeycloakBackupVersion, "", ""))

	// Add kc backup list kind to schema
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   keycloak.KeycloakBackupGroup,
		Version: keycloak.KeycloakBackupVersion,
		Kind:    keycloak.KeycloakBackupsKind,
	},
		dr.CreateUnstructuredListWithGVK(keycloak.KeycloakBackupGroup, keycloak.KeycloakBackupKind, keycloak.KeycloakBackupsKind, keycloak.KeycloakBackupVersion, "", ""))

	return nil
}
