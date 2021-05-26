module github.com/integr8ly/integreatly-operator

go 1.16

require (
	github.com/3scale/3scale-operator v0.2.1-0.20210312114906-e612e89addbf
	github.com/3scale/marin3r v0.7.0
	github.com/Apicurio/apicurio-registry-operator v0.0.0-20200903111206-f9f14054bc16
	github.com/Masterminds/semver v1.5.0
	github.com/PuerkitoBio/goquery v1.6.1
	github.com/RHsyseng/operator-utils v0.0.0-20200709142328-d5a5812a443f
	github.com/aerogear/unifiedpush-operator v0.5.0
	github.com/antchfx/xmlquery v1.3.5
	github.com/apicurio/apicurio-operators/apicurito v0.0.0-20200123142409-83e0a91dd6be
	github.com/aws/aws-sdk-go v1.35.23
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/prometheus-operator v0.40.0
	github.com/eclipse/che-operator v0.0.0-20201214125341-cce874092f25
	github.com/envoyproxy/go-control-plane v0.9.7
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/spec v0.19.12
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.4.0
	github.com/google/go-querystring v1.0.0
	github.com/googleapis/gnostic v0.5.3 // indirect
	github.com/headzoo/surf v1.0.0
	github.com/headzoo/ut v0.0.0-20181013193318-a13b5a7a02ca // indirect
	github.com/integr8ly/application-monitoring-operator v1.6.0
	github.com/integr8ly/cloud-resource-operator v0.24.0
	github.com/integr8ly/grafana-operator v2.0.0+incompatible
	github.com/integr8ly/grafana-operator/v3 v3.10.1
	github.com/integr8ly/keycloak-client v0.1.4
	github.com/keycloak/keycloak-operator v0.0.0-20210506103913-57d81e278bcb
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.2
	github.com/openshift/api v3.9.1-0.20191031084152-11eee842dafd+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/openshift/cluster-samples-operator v0.0.0-20191113195805-9e879e661d71
	github.com/operator-framework/operator-lifecycle-manager v0.17.0
	// github.com/operator-framework/operator-marketplace v0.0.0-20200919233811-2d6d71892437
	github.com/operator-framework/operator-registry v1.14.3
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.6.0
	github.com/syndesisio/syndesis/install/operator v0.0.0-20201210151747-8264b9904eab
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	google.golang.org/protobuf v1.24.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.20.0
	k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v12.0.0+incompatible
	// k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.38.3
	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210105115604-44119421ec6b
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v1.2.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.3
)

// replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v1.2.0

// replace github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.38.3

replace (
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.19.6
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/operator-framework/api => github.com/operator-framework/api v0.1.1
	github.com/operator-framework/operator-lifecycle-manager => github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200321030439-57b580e57e88
	github.com/operator-framework/operator-registry => github.com/operator-framework/operator-registry v1.6.2-0.20200330184612-11867930adb5
	k8s.io/api => k8s.io/api v0.0.0-20200331211856-3c24962070e9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.4
	k8s.io/apiserver => k8s.io/apiserver v0.19.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.4
	k8s.io/client-go => k8s.io/client-go v0.0.0-20200331211856-d847b4c9642e
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.4
	k8s.io/code-generator => k8s.io/code-generator v0.19.4
	k8s.io/component-base => k8s.io/component-base v0.19.4
	k8s.io/cri-api => k8s.io/cri-api v0.19.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.4
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.1.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210113000636-45edf8a2a574
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.4
	k8s.io/kubectl => k8s.io/kubectl v0.19.4
	k8s.io/kubelet => k8s.io/kubelet v0.19.4
	k8s.io/kubernetes => k8s.io/kubernetes v1.19.4
	k8s.io/metrics => k8s.io/metrics v0.19.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.4
)

replace github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.14

replace github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
