package custom_domain

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/metrics"
	ingressController "github.com/openshift/api/operator/v1"
	customdomainv1alpha1 "github.com/openshift/custom-domains-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const CustomDomainDomain = "custom-domain_domain"

func GetDomain(ctx context.Context, serverClient client.Client, installation *v1alpha1.RHMI) (bool, string, error) {

	if installation == nil {
		return false, "", fmt.Errorf("nil pointer passed in for installation parameter")
	}

	parameter, ok, err := addon.GetStringParameter(ctx, serverClient, installation.Namespace, CustomDomainDomain)

	if err != nil {
		return false, "", err
	}

	if ok {
		parameter = strings.TrimSpace(parameter)
		valid := IsValidDomain(parameter)

		if valid {
			return true, parameter, nil
		}

		// Not returning false here to allow monitoring stack to install - will fail on 3scale installation stage
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

func HasValidCustomDomainCR(ctx context.Context, serverClient client.Client, domain string) (bool, string, error) {
	ok := IsValidDomain(domain)
	if !ok {
		return false, "", fmt.Errorf("invalid domain string passed: \"%s\"", domain)
	}

	customDomains := &customdomainv1alpha1.CustomDomainList{}

	err := serverClient.List(ctx, customDomains)
	if err != nil {
		return false, "", err
	}

	for _, item := range customDomains.Items {
		if item.Spec.Domain == domain {
			if item.Status.State == customdomainv1alpha1.CustomDomainStateReady {
				return true, item.Name, nil
			}
			return false, item.Name, fmt.Errorf("custom domain CR in failing state for: \"%s\"", domain)
		}
	}

	return false, "", fmt.Errorf("no custom domain CR found for: \"%s\"", domain)
}

func HasValidIngressControllerCR(ctx context.Context, serverClient client.Client, customDomainName, domain string) (bool, error) {
	ok := IsValidDomain(domain)
	if !ok {
		return false, fmt.Errorf("invalid domain string passed: \"%s\"", domain)
	}

	if customDomainName == "" {
		ingressControllers := &ingressController.IngressControllerList{}

		err := serverClient.List(ctx, ingressControllers)
		if err != nil {
			return false, err
		}

		for _, item := range ingressControllers.Items {
			if item.Spec.Domain == domain {
				for _, condition := range item.Status.Conditions {
					if condition.Type == "Available" && condition.Status == "True" {
						return true, nil
					}
				}

				return false, fmt.Errorf("ingress controller CR in failing state for: \"%s\"", domain)
			}
		}
	} else {
		ingressControllerCR := &ingressController.IngressController{}
		key := client.ObjectKey{
			Name:      customDomainName,
			Namespace: "openshift-ingress-operator",
		}

		err := serverClient.Get(ctx, key, ingressControllerCR)
		if err != nil {
			return false, fmt.Errorf("no ingress controller CR found for: \"%s\"", domain)
		}

		for _, condition := range ingressControllerCR.Status.Conditions {
			if condition.Type == "Available" && condition.Status == "True" {
				return true, nil
			}
		}

		return false, fmt.Errorf("ingress controller CR in failing state for: \"%s\"", domain)
	}

	return false, fmt.Errorf("no ingress controller CR found for: \"%s\"", domain)
}

func UpdateErrorAndCustomDomainMetric(installation *v1alpha1.RHMI, active bool, err error) {
	if err != nil {
		installation.Status.CustomDomain.Error = err.Error()
		metrics.SetCustomDomain(active, 1)
		return
	}
	installation.Status.CustomDomain.Error = ""
	metrics.SetCustomDomain(active, 0)
}

func GetIngressRouterService(ctx context.Context, serverClient client.Client, svcName string) (*v1.Service, error) {
	ingressRouterService := &v1.Service{}
	key := client.ObjectKey{
		Name:      svcName,
		Namespace: "openshift-ingress",
	}
	err := serverClient.Get(ctx, key, ingressRouterService)
	if err != nil {
		return nil, err
	}
	return ingressRouterService, nil
}

func GetIngressRouterIPs(ingress []v1.LoadBalancerIngress) ([]net.IP, error) {
	var ips []net.IP
	if hostname := ingress[0].Hostname; hostname != "" {
		var err error
		ips, err = net.LookupIP(hostname)
		if err != nil {
			return nil, fmt.Errorf("unable to perform ip lookup for hostname %s: %v", hostname, err)
		}
	} else {
		for i := range ingress {
			ips = append(ips, net.ParseIP(ingress[i].IP))
		}
	}
	return ips, nil
}

func IsCustomDomain(installation *v1alpha1.RHMI) bool {
	domainStatus := installation.Status.CustomDomain
	if domainStatus == nil {
		return false
	}
	return domainStatus.Enabled
}
