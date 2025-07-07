package resources

import (
	"encoding/json"
	"fmt"

	"github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	codecs serializer.CodecFactory
)

func LoadKubernetesResource(jsonData []byte, namespace string) (runtime.Object, error) {
	u := unstructured.Unstructured{}

	err := u.UnmarshalJSON(jsonData)
	if err != nil {
		return nil, err
	}
	u.SetNamespace(namespace)
	setIntegreatlyLabel(&u)
	if u.GetObjectKind().GroupVersionKind().Kind == "ImageStream" {
		u.SetAPIVersion("image.openshift.io/v1")
	}

	if u.GetObjectKind().GroupVersionKind().Kind == "Template" {
		u.SetAPIVersion("template.openshift.io/v1")
	}

	return RuntimeObjectFromUnstructured(&u)
}

func RuntimeObjectFromUnstructured(u *unstructured.Unstructured) (runtime.Object, error) {
	gvk := u.GroupVersionKind()
	decoder := codecs.UniversalDecoder(gvk.GroupVersion())

	b, err := u.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("error running MarshalJSON on unstructured object: %v", err)
	}

	ro, _, err := decoder.Decode(b, &gvk, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode json data with gvk(%v): %v", gvk.String(), err)
	}
	return ro, nil
}

func UnstructuredFromRuntimeObject(ro runtime.Object) (*unstructured.Unstructured, error) {
	b, err := json.Marshal(ro)
	if err != nil {
		return nil, fmt.Errorf("error running MarshalJSON on runtime object: %v", err)
	}
	var u unstructured.Unstructured
	if err := json.Unmarshal(b, &u.Object); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json into unstructured object: %v", err)
	}
	return &u, nil
}

func setIntegreatlyLabel(u *unstructured.Unstructured) {
	labels := u.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels["integreatly"] = "true"
	u.SetLabels(labels)
}

func init() {
	codecsScheme := runtime.NewScheme()
	err := scheme.AddToScheme(codecsScheme)
	if err != nil {
		fmt.Printf("failed adding to scheme with error: %v", err)
		return
	}

	err = v1alpha1.AddToSchemes.AddToScheme(codecsScheme)
	if err != nil {
		fmt.Printf("failed adding to scheme with error: %v", err)
		return
	}

	codecs = serializer.NewCodecFactory(codecsScheme)
}
