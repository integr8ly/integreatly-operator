package threescale

import (
	"github.com/openshift/api/route/v1"
	"strings"
)

type TenantAccountsFilter struct {
	providers  []AccountDetail
	developers []developerRoute
}

type developerRoute struct {
	Url   string
	State string
}

func NewTenantAccountsFilter(accounts []AccountDetail) TenantAccountsFilter {
	f := TenantAccountsFilter{}
	f.setProviders(accounts)
	f.generateDeveloperRoutes(accounts)

	return f
}

func (f *TenantAccountsFilter) Provider(r v1.Route) bool {
	for _, account := range f.providers {
		if strings.HasSuffix(account.AdminBaseURL, r.Spec.Host) {
			return account.State == "approved"
		}
	}
	return false
}

func (f *TenantAccountsFilter) Developer(r v1.Route) bool {
	for _, account := range f.developers {
		if strings.HasSuffix(account.Url, r.Spec.Host) {
			return account.State == "approved"
		}
	}
	return false
}

func (f *TenantAccountsFilter) generateDeveloperRoutes(accounts []AccountDetail) {
	for _, account := range accounts {
		if strings.Contains(account.AdminBaseURL, "-admin.") {
			d := developerRoute{}
			d.State = account.State
			search := "-admin."
			i := strings.LastIndex(account.AdminBaseURL, search)

			d.Url = account.AdminBaseURL[:i] + strings.Replace(account.AdminBaseURL[i:], search, ".", 1)
			f.developers = append(f.developers, d)

		}
	}
}

func (f *TenantAccountsFilter) setProviders(accounts []AccountDetail) {
	f.providers = accounts
}
