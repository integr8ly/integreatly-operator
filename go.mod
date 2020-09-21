module github.com/integr8ly/integreatly-operator

go 1.13

require (
	github.com/3scale/3scale-operator v0.5.0
	github.com/3scale/marin3r v0.5.1
	github.com/Apicurio/apicurio-registry-operator v0.0.0-20200903111206-f9f14054bc16
	github.com/Masterminds/semver v1.5.0
	github.com/PuerkitoBio/goquery v1.5.1
	github.com/RHsyseng/operator-utils v1.4.4
	github.com/aerogear/unifiedpush-operator v0.5.0
	github.com/apicurio/apicurio-operators/apicurito v0.0.0-20200123142409-83e0a91dd6be
	github.com/aws/aws-sdk-go v1.34.5
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/prometheus-operator v0.40.0
	github.com/eclipse/che-operator v0.0.0-20191122191946-81d08d3f0fde
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/spec v0.19.9
	github.com/google/go-querystring v1.0.0
	github.com/headzoo/surf v1.0.0
	github.com/headzoo/ut v0.0.0-20181013193318-a13b5a7a02ca // indirect
	github.com/integr8ly/application-monitoring-operator v1.1.1
	github.com/integr8ly/cloud-resource-operator v0.16.1
	github.com/integr8ly/grafana-operator/v3 v3.4.0
	github.com/integr8ly/keycloak-client v0.1.2
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/keycloak/keycloak-operator v0.0.0-20200518131634-204a6a8d6ee0
	github.com/matryer/moq v0.1.3
	github.com/openshift/api v3.9.1-0.20191031084152-11eee842dafd+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/openshift/cluster-samples-operator v0.0.0-20191113195805-9e879e661d71
	github.com/operator-framework/operator-lifecycle-manager v3.11.0+incompatible
	github.com/operator-framework/operator-marketplace v0.0.0-20200919233811-2d6d71892437
	github.com/operator-framework/operator-registry v1.12.2
	github.com/operator-framework/operator-sdk v0.19.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/syndesisio/syndesis/install/operator v0.0.0-20200921104849-b99c54c8a481
	golang.org/x/net v0.0.0-20200625001655-4c5254603344
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	sigs.k8s.io/controller-runtime v0.6.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.6
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20191022152013-2823239d2298
	github.com/operator-framework/api => github.com/operator-framework/api v0.1.1
	github.com/operator-framework/operator-lifecycle-manager => github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200321030439-57b580e57e88
	github.com/operator-framework/operator-registry => github.com/operator-framework/operator-registry v1.6.2-0.20200330184612-11867930adb5
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.17.1
	k8s.io/api => k8s.io/api v0.17.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.5
	k8s.io/apiserver => k8s.io/apiserver v0.17.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.4
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.4
	k8s.io/code-generator => k8s.io/code-generator v0.17.4
	k8s.io/component-base => k8s.io/component-base v0.17.4
	k8s.io/cri-api => k8s.io/cri-api v0.17.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.4
	k8s.io/kubectl => k8s.io/kubectl v0.17.4
	k8s.io/kubelet => k8s.io/kubelet v0.17.4
	k8s.io/kubernetes => k8s.io/kubernetes v1.17.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.4
	k8s.io/metrics => k8s.io/metrics v0.17.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.4
)

replace sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.5.2

replace github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.38.3
