module github.com/integr8ly/integreatly-operator

go 1.13

require (
	github.com/3scale/3scale-operator v0.5.0
	github.com/3scale/marin3r v0.5.1
	github.com/Apicurio/apicurio-registry-operator v0.0.0-20200604085617-be91f2b38134
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Masterminds/semver v1.5.0
	github.com/PuerkitoBio/goquery v1.5.1
	github.com/RHsyseng/operator-utils v0.0.0-20200107144857-313dbcf0e3bd
	github.com/aerogear/unifiedpush-operator v0.5.0
	// No tags on the apicurio repo so needed to use a commit hash
	github.com/apicurio/apicurio-operators/apicurito v0.0.0-20200123142409-83e0a91dd6be
	github.com/aws/aws-sdk-go v1.25.50
	github.com/blang/semver v3.5.1+incompatible
	github.com/bugsnag/bugsnag-go v1.5.0 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/coreos/prometheus-operator v0.38.0
	github.com/docker/go-metrics v0.0.0-20181218153428-b84716841b82 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/eclipse/che-operator v0.0.0-20191122191946-81d08d3f0fde
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/spec v0.19.6
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/google/go-querystring v1.0.0
	github.com/gorilla/handlers v1.4.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/headzoo/surf v1.0.0
	github.com/headzoo/ut v0.0.0-20181013193318-a13b5a7a02ca // indirect
	github.com/integr8ly/application-monitoring-operator v1.1.1
	github.com/integr8ly/cloud-resource-operator v0.16.1
	github.com/integr8ly/grafana-operator/v3 v3.0.2-0.20200103111057-03d7fa884db4
	github.com/integr8ly/keycloak-client v0.1.2
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/keycloak/keycloak-operator v0.0.0-20200518131634-204a6a8d6ee0
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/matryer/moq v0.0.0-20200310130814-7721994d1b54
	github.com/opencontainers/runc v1.0.0-rc9 // indirect
	github.com/openshift/api v3.9.1-0.20191031084152-11eee842dafd+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/openshift/cluster-samples-operator v0.0.0-20191113195805-9e879e661d71
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200321030439-57b580e57e88
	github.com/operator-framework/operator-marketplace v0.0.0-20191105191618-530c85d41ce7
	github.com/operator-framework/operator-registry v1.6.2-0.20200330184612-11867930adb5
	github.com/operator-framework/operator-sdk v0.17.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/syndesisio/syndesis/install/operator v0.0.0-20200406175937-7e2945a5ee43
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.6 // indirect
	golang.org/x/lint v0.0.0-20200130185559-910be7a94367 // indirect
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.18.0
	k8s.io/apiextensions-apiserver v0.17.4
	k8s.io/apimachinery v0.17.12
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
	sigs.k8s.io/controller-runtime v0.5.2
)

// Pinned to kubernetes-1.16.2
// To bump the versions below run bump-k8s-dependency-versions.sh vX.Y.Z
replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc8
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20191022152013-2823239d2298
	k8s.io/api => k8s.io/api v0.17.12
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.12
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.13-rc.0
	k8s.io/apiserver => k8s.io/apiserver v0.17.12
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.12
	k8s.io/client-go => k8s.io/client-go v0.17.12
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.12
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.12
	k8s.io/code-generator => k8s.io/code-generator v0.17.13-rc.0
	k8s.io/component-base => k8s.io/component-base v0.17.12
	k8s.io/cri-api => k8s.io/cri-api v0.17.13-rc.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.12
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.12
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.12
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.12
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.12
	k8s.io/kubectl => k8s.io/kubectl v0.17.12
	k8s.io/kubelet => k8s.io/kubelet v0.17.12
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.12
	k8s.io/metrics => k8s.io/metrics v0.17.12
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.12
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.2
)

// Pinned to operator-sdk v0.15.1
replace (
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.17.1
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.4.0
)

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20190924102528-32369d4db2ad // Required until https://github.com/operator-framework/operator-lifecycle-manager/pull/1241 is resolved

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm

replace k8s.io/node-api => k8s.io/node-api v0.17.12

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.12

replace k8s.io/sample-controller => k8s.io/sample-controller v0.17.12
