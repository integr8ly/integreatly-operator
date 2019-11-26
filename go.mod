module github.com/integr8ly/integreatly-operator

go 1.13

require (
	github.com/RHsyseng/operator-utils v0.0.0-20191024171829-7e918ca09e5e
	github.com/aerogear/mobile-developer-console-operator v0.0.0-20191112104059-b2ac382d666a
	github.com/aerogear/mobile-security-service-operator v0.0.0-20191107125005-e74272a6c7ad
	github.com/aerogear/unifiedpush-operator v0.0.0-20191115172337-dde4a2e46a99
	github.com/eclipse/che-operator v0.0.0-20191115104040-d273007e7e91
	github.com/fabric8-launcher/launcher-operator v0.1.3
	github.com/ghodss/yaml v1.0.1-0.20180820084758-c7ce16629ff4
	github.com/go-openapi/spec v0.19.4
	github.com/google/uuid v1.1.1
	github.com/integr8ly/grafana-operator v2.0.0+incompatible
	github.com/matryer/moq v0.0.0-20191106032847-0e0395200ade // indirect
	github.com/openshift/api v3.9.1-0.20191031084152-11eee842dafd+incompatible
	github.com/openshift/client-go v0.0.0-20191022152013-2823239d2298
	github.com/openshift/cluster-samples-operator v0.0.0-20191113195805-9e879e661d71
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20190605231540-b8a4faf68e36
	github.com/operator-framework/operator-marketplace v0.0.0-20191110204144-8bb630bf8a91
	github.com/operator-framework/operator-sdk v0.12.1-0.20191112211508-82fc57de5e5b
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.5
	github.com/syndesisio/syndesis/install/operator v0.0.0-20191118082300-030400fcc362
	golang.org/x/net v0.0.0-20190827160401-ba9fcec4b297
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d
	sigs.k8s.io/controller-runtime v0.4.0
)

// Pinned to kubernetes-1.15.4
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190918201827-3de75813f604
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190918200908-1e17798da8c1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918202139-0b14c719ca62
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190918203125-ae665f80358a
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918202959-c340507a5d48
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190918200425-ed2f0867c778
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190817025403-3ae76f584e79
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190918203248-97c07dcbb623
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190918201136-c3a845f1fbb2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190918202837-c54ce30c680e
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190918202429-08c8357f8e2d
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190918202713-c34a54b3ec8e
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190918202550-958285cf3eef
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190918203421-225f0541b3ea
	k8s.io/metrics => k8s.io/metrics v0.0.0-20190918202012-3c1ca76f5bda
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190918201353-5cc279503896
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.3.0
)
