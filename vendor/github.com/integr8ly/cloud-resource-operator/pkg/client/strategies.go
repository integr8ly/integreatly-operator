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

	"github.com/integr8ly/cloud-resource-operator/pkg/client/aws"
	stratType "github.com/integr8ly/cloud-resource-operator/pkg/client/types"
	"github.com/integr8ly/cloud-resource-operator/pkg/resources"
	configv1 "github.com/openshift/api/config/v1"
	errorUtil "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// exported tiers to be used by RHOAM operator
	TierProduction  = "production"
	TierDevelopment = "development"
)

// ReconcileStrategyMaps to be used to reconcile strategy maps expected in RHOAM installs
// A single function which can check the infrastructure and provision the correct strategy config map
func ReconcileStrategyMaps(ctx context.Context, client client.Client, timeConfig *stratType.StrategyTimeConfig, tier, namespace string) error {
	platformType, err := resources.GetPlatformType(ctx, client)
	if err != nil {
		return err
	}
	var strategyProvider stratType.StrategyProvider
	switch platformType {
	case configv1.AWSPlatformType:
		// reconciles aws specific strategy map
		strategyProvider = aws.NewAWSStrategyProvider(client, tier, namespace)
	default:
		return errorUtil.New("Unsupported platform type")
	}
	if err := strategyProvider.ReconcileStrategyMap(ctx, client, timeConfig); err != nil {
		return errorUtil.Wrapf(err, "failed to reconcile %s strategy map", platformType)
	}
	return nil
}
