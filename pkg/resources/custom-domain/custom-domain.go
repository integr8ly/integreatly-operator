package custom_domain

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	customdomainv1alpha1 "github.com/openshift/custom-domains-operator/api/v1alpha1"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const CustomDomainDomain = "custom-domain_domain"

func GetDomain(ctx context.Context, serverClient client.Client, installation *v1alpha1.RHMI) (bool, string, error) {

	if installation == nil {
		return false, "", fmt.Errorf("nil pointer passed in for installation parameter")
	}

	parameter, ok, err := addon.GetParameter(ctx, serverClient, installation.Namespace, CustomDomainDomain)

	if err != nil {
		return false, "", err
	}

	if ok {
		parameter := string(parameter)
		parameter = strings.TrimSpace(parameter)
		valid := IsValidDomain(parameter)

		if valid {
			return true, parameter, nil
		}

		return true, parameter, fmt.Errorf("not valid domain \"%v\"", parameter)
	}

	return false, "", nil
}

func IsValidDomain(domain string) bool {
	if len(domain) == 0 {
		return false
	}

	u, err := url.ParseRequestURI(fmt.Sprintf("https://%s/", domain))

	if err == nil && u.Host == domain {
		return true
	}
	return false
}

func HasValidCustomDomainCR(ctx context.Context, serverClient client.Client, domain string) (bool, error) {
	ok := IsValidDomain(domain)
	if !ok {
		return false, fmt.Errorf("invalid domain string passed: \"%s\"", domain)
	}

	customDomains := &customdomainv1alpha1.CustomDomainList{}

	err := serverClient.List(ctx, customDomains)
	if err != nil {
		return false, err
	}

	for _, item := range customDomains.Items {
		if item.Spec.Domain == domain {
			if item.Status.State == customdomainv1alpha1.CustomDomainStateReady {
				return true, nil
			}
			return false, fmt.Errorf("custom domain CR in failing state for: \"%s\"", domain)
		}
	}

	return false, fmt.Errorf("no custom domain CR found for: \"%s\"", domain)
}

func UpdateErrorAndMetric(installation *v1alpha1.RHMI, log logger.Logger, err error, message string) {
	if err == nil {
		installation.Status.CustomDomain.Error = ""
		metrics.SetCustomDomain("active", 0)
	} else {
		log.Error(message, err)
		installation.Status.CustomDomain.Error = err.Error()
		metrics.SetCustomDomain("active", 1)
	}
}
