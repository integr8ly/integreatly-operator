/*
Strategy Setup

A utility to abstract the various strategy map ConfigMaps from the service using CRO

Problem Statement:
 - We require strategy map ConfigMaps to be in place to provide configuration used to provision cloud resources
 - Each provider overrides null configuration with expected specific defaults
 - Non null configuration provided via strategy map is consumed by CRO as configuration for the provisioning of cloud resources
 - Strategy maps provide the source of truth for the expected state of a cloud resource
 - Strategy maps are used to update and modify the state of provisioned cloud resources

This utility provides the abstraction necessary, provisioning a strategy map for the infrastructure in which the operator is currently running in

We accept start times for maintenance and backup times as a level of abstraction
Building the correct maintenance and backup times necessary for specific cloud providers
*/

package client

import (
	"context"
	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// exported tiers to be used by RHOAM operator
	TierProduction  = "production"
	TierDevelopment = "development"

	postgresStratKey    = "postgres"
	redisStratKey       = "redis"
	blobstorageStratKey = "blobstorage"
)

type StrategyTimeConfig struct {
	BackupStartTime      string
	MaintenanceStartTime string
}

// ReconcileStrategyMaps to be used to reconcile strategy maps expected in RHOAM installs
// A single function which can check the infrastructure and provision the correct strategy config map
func ReconcileStrategyMaps(ctx context.Context, client client.Client, timeConfig *StrategyTimeConfig, tier, namespace string) error {
	// reconciles aws specific strategy map
	if err := reconcileAWSStrategyMap(ctx, client, timeConfig, tier, namespace); err != nil {
		return errorUtil.Wrapf(err, "failed to reconcile aws strategy map")
	}
	return nil
}
