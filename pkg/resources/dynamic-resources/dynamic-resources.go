package dynamic_resources

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func ConvertKeycloakTypedToUnstructured(kc Keycloak) (unstructured.Unstructured, error) {
	unstructuredKeycloak := unstructured.Unstructured{}
	// convert typed to map
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kc)
	if err != nil {
		return unstructuredKeycloak, err
	}

	// convert map to unstructured type
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, &unstructuredKeycloak)
	if err != nil {
		return unstructuredKeycloak, err
	}

	return unstructuredKeycloak, nil
}

func ConvertKeycloakUnstructuredToTyped(u unstructured.Unstructured) (Keycloak, error) {
	kcTyped := Keycloak{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &kcTyped)
	if err != nil {
		return kcTyped, err
	}

	return kcTyped, nil
}

func SetupGVKandMetaOnUnstructured(u unstructured.Unstructured, group, resource, version, resourceName, namespace string) unstructured.Unstructured {
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Kind:    resource,
		Version: version,
	})
	u.SetName(resourceName)
	u.SetNamespace(namespace)

	return u
}