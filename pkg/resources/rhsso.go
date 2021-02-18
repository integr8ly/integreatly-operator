package resources

import (
	"context"
	"fmt"

	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	databaseSecretName         = "keycloak-db-secret"
	databaseSecretKeyDatabase  = "POSTGRES_DATABASE"
	databaseSecretKeyExtPort   = "POSTGRES_EXTERNAL_PORT"
	databaseSecretKeyExtHost   = "POSTGRES_EXTERNAL_ADDRESS"
	databaseSecretKeyPassword  = "POSTGRES_PASSWORD"
	databaseSecretKeyUsername  = "POSTGRES_USERNAME"
	databaseSecretKeySuperuser = "POSTGRES_SUPERUSER"
)

//ReconcileRHSSOPostgresCredentials Provisions postgres and creates external database secret based on Installation CR, secret will be nil while the postgres instance is provisioning
func ReconcileRHSSOPostgresCredentials(ctx context.Context, installation *integreatlyv1alpha1.RHMI, serverClient k8sclient.Client, name, ns, nsPostfix string) (*crov1.Postgres, error) {
	postgresNS := installation.Namespace
	postgres, err := croUtil.ReconcilePostgres(ctx, serverClient, nsPostfix, installation.Spec.Type, croUtil.TierProduction, name, postgresNS, name, postgresNS, func(cr metav1.Object) error {
		owner.AddIntegreatlyOwnerAnnotations(cr, installation)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to provision postgres instance while reconciling rhsso postgres credentials, %s: %w", name, err)
	}
	if postgres.Status.Phase != types.PhaseComplete {
		return nil, nil
	}
	postgresSec := &corev1.Secret{}
	err = serverClient.Get(ctx, k8sclient.ObjectKey{Name: postgres.Status.SecretRef.Name, Namespace: postgres.Status.SecretRef.Namespace}, postgresSec)
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres credential secret while reconciling rhsso postgres credentials, %s: %w", name, err)
	}
	// create secret using the default name which the keycloak operator expects
	keycloakSec := &corev1.Secret{
		ObjectMeta: controllerruntime.ObjectMeta{
			Name:      databaseSecretName,
			Namespace: ns,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, serverClient, keycloakSec, func() error {
		owner.AddIntegreatlyOwnerAnnotations(keycloakSec, installation)
		if keycloakSec.Data == nil {
			keycloakSec.Data = map[string][]byte{}
		}
		// based on https://github.com/keycloak/keycloak-operator/blob/d6203c6206bcf011023a289620f93d03cd755810/docs/external-database.asciidoc
		keycloakSec.Data[databaseSecretKeyDatabase] = postgresSec.Data["database"]
		keycloakSec.Data[databaseSecretKeyExtPort] = postgresSec.Data["port"]
		keycloakSec.Data[databaseSecretKeyExtHost] = postgresSec.Data["host"]
		keycloakSec.Data[databaseSecretKeyPassword] = postgresSec.Data["password"]
		keycloakSec.Data[databaseSecretKeyUsername] = postgresSec.Data["username"]
		keycloakSec.Data[databaseSecretKeySuperuser] = []byte("false")
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create keycloak external database secret, %s: %w", name, err)
	}
	return postgres, nil
}
