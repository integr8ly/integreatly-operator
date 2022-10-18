package dynamic_resources

import (
	kc "github.com/integr8ly/keycloak-client/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Sets up the GVK Meta on unstructured object
func CreateUnstructuredWithGVK(group, kind, version, resourceName, namespace string) *unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Kind:    kind,
		Version: version,
	})
	u.SetAPIVersion(group + "/" + version)

	if resourceName != "" {
		u.SetName(resourceName)
	}
	if namespace != "" {
		u.SetNamespace(namespace)
	}

	return &u
}

// Sets up the GVK Meta on unstructured objects
func CreateUnstructuredListWithGVK(group, itemKind, listKind, version, resourceName, namespace string) *unstructured.UnstructuredList {
	u := unstructured.UnstructuredList{}

	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    listKind,
	})
	u.SetAPIVersion(group + "/" + version)
	for _, item := range u.Items {
		item.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   group,
			Version: version,
			Kind:    itemKind,
		})
		item.SetAPIVersion(group + "/" + version)

	}

	return &u
}

// Converts Keycloak Typed object to unstructured and sets GVK on the object
func ConvertKeycloakTypedToUnstructured(keycloak *kc.Keycloak) (*unstructured.Unstructured, error) {
	kcUnstructed := &unstructured.Unstructured{}
	keycloak.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   kc.KeycloakGroup,
		Version: kc.KeycloakVersion,
		Kind:    kc.KeycloakKind,
	})
	// convert typed to map
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(keycloak)
	if err != nil {
		return kcUnstructed, err
	}

	// convert map to unstructured type
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, kcUnstructed)
	if err != nil {
		return kcUnstructed, err
	}

	return kcUnstructed, nil
}

// Converts Keycloak unstructured to typed and sets GVK on object
func ConvertKeycloakUnstructuredToTyped(u unstructured.Unstructured) (*kc.Keycloak, error) {
	unstructuredKc := unstructuredWithGVK(u, kc.KeycloakGroup, kc.KeycloakVersion, kc.KeycloakKind, kc.KeycloakApiVersion)
	kcTyped := &kc.Keycloak{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKc.Object, &kcTyped)
	if err != nil {
		return kcTyped, err
	}

	return kcTyped, nil
}

// Converts Keycloak Realm Typed object to unstructured and sets GVK on object
func ConvertKeycloakRealmTypedToUnstructured(kcRealm *kc.KeycloakRealm) (*unstructured.Unstructured, error) {
	unstructuredRealm := &unstructured.Unstructured{}
	kcRealm.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   kc.KeycloakRealmGroup,
		Version: kc.KeycloakRealmVersion,
		Kind:    kc.KeycloakRealmKind,
	})
	// convert typed to map
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(kcRealm)
	if err != nil {
		return unstructuredRealm, err
	}

	// convert map to unstructured type
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, unstructuredRealm)
	if err != nil {
		return unstructuredRealm, err
	}

	return unstructuredRealm, nil
}

// Converts Keycloak Realm Unstructured object to structured and sets GVK on object
func ConvertKeycloakRealmUnstructuredToTyped(u unstructured.Unstructured) (*kc.KeycloakRealm, error) {
	unstructuredKeycloakRealm := unstructuredWithGVK(u, kc.KeycloakRealmGroup, kc.KeycloakRealmVersion, kc.KeycloakRealmKind, kc.KeycloakRealmApiVersion)
	keycloakRealmTyped := &kc.KeycloakRealm{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakRealm.Object, &keycloakRealmTyped)
	if err != nil {
		return keycloakRealmTyped, err
	}

	return keycloakRealmTyped, nil
}

// Converts Keycloak Users unstructured object to typed keycloak users list and sets up GVK on the object
func ConvertKeycloakUsersUnstructuredToTyped(u unstructured.UnstructuredList) (*kc.KeycloakUserList, error) {
	unstructuredKeycloakUserList := unstructuredListWithGVK(u, kc.KeycloakUserGroup, kc.KeycloakUserVersion, kc.KeycloakUserListKind, kc.KeycloakUserApiVersion)
	kcUsersTyped := kc.KeycloakUserList{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakUserList.Object, &kcUsersTyped)
	if err != nil {
		return &kcUsersTyped, err
	}

	return &kcUsersTyped, nil
}

// Converts Keycloak Users typed object to unstructured object and sets GVK on each item
func ConvertKeycloakUsersTypedToUnstructured(kcUserList *kc.KeycloakUserList) (*unstructured.UnstructuredList, error) {
	unstructuredUsers := &unstructured.UnstructuredList{}
	kcUserList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   kc.KeycloakUserGroup,
		Version: kc.KeycloakUserVersion,
		Kind:    kc.KeycloakUserListKind,
	})
	for _, item := range kcUserList.Items {
		item.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   kc.KeycloakUserGroup,
			Version: kc.KeycloakUserVersion,
			Kind:    kc.KeycloakUserKind,
		})
		item.APIVersion = kc.KeycloakUserApiVersion
	}
	unstructuredKeycloakUserList := unstructuredListWithGVK(*unstructuredUsers, kc.KeycloakUserGroup, kc.KeycloakUserVersion, kc.KeycloakUserListKind, kc.KeycloakUserApiVersion)
	// convert typed to map
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kcUserList)
	if err != nil {
		return unstructuredKeycloakUserList, err
	}

	// convert map to unstructured type
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, &unstructuredKeycloakUserList)
	if err != nil {
		return unstructuredKeycloakUserList, err
	}

	return unstructuredKeycloakUserList, nil
}

// Converts Keycloak User Unstructured object to typed Keycloak user object and sets GVK on object
func ConvertKeycloakUserUnstructuredToTyped(u unstructured.Unstructured) (*kc.KeycloakUser, error) {
	unstructuredKeycloakUser := unstructuredWithGVK(u, kc.KeycloakUserGroup, kc.KeycloakUserVersion, kc.KeycloakUserKind, kc.KeycloakUserApiVersion)
	kcUserTyped := kc.KeycloakUser{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakUser.Object, &kcUserTyped)
	if err != nil {
		return &kcUserTyped, err
	}

	return &kcUserTyped, nil
}

// Converts Keycloak User typed object to keycloak user unstructured object
func ConvertKeycloakUserTypedToUnstructured(kcUser *kc.KeycloakUser) (*unstructured.Unstructured, error) {
	unstructuredUser := &unstructured.Unstructured{}
	kcUser.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   kc.KeycloakUserGroup,
		Version: kc.KeycloakUserVersion,
		Kind:    kc.KeycloakUserKind,
	})
	kcUser.APIVersion = kc.KeycloakUserApiVersion
	// convert typed to map
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kcUser)
	if err != nil {
		return unstructuredUser, err
	}

	// convert map to unstructured type
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, unstructuredUser)
	if err != nil {
		return unstructuredUser, err
	}

	return unstructuredUser, nil
}

// Converts Keycloak Client typed to unstructured and sets GVK on object
func ConvertKeycloakClientTypedToUnstructured(kcClient *kc.KeycloakClient) (*unstructured.Unstructured, error) {
	unstructuredClient := &unstructured.Unstructured{}
	kcClient.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   kc.KeycloakClientGroup,
		Version: kc.KeycloakClientVersion,
		Kind:    kc.KeycloakClientKind,
	})
	kcClient.APIVersion = kc.KeycloakClientApiVersion
	// convert typed to map
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kcClient)
	if err != nil {
		return unstructuredClient, err
	}
	// convert map to unstructured type
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, &unstructuredClient)
	if err != nil {
		return unstructuredClient, err
	}

	return unstructuredClient, nil
}

// Converts keycloak client unstructured to typed and sets GVK on object
func ConvertKeycloakClientUnstructuredToTyped(u unstructured.Unstructured) (*kc.KeycloakClient, error) {
	unstructuredKeycloakClient := unstructuredWithGVK(u, kc.KeycloakClientGroup, kc.KeycloakClientVersion, kc.KeycloakClientKind, kc.KeycloakClientApiVersion)
	kcClientTyped := kc.KeycloakClient{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakClient.Object, &kcClientTyped)
	if err != nil {
		return &kcClientTyped, err
	}

	return &kcClientTyped, nil
}

// Converts Keycloak Users unstructured object to typed keycloak users list and sets GVK on object
func ConvertKeycloakClientsUnstructuredToTyped(u unstructured.UnstructuredList) (*kc.KeycloakClientList, error) {
	unstructuredKeycloakClientList := unstructuredListWithGVK(u, kc.KeycloakClientGroup, kc.KeycloakClientVersion, kc.KeycloakClientListKind, kc.KeycloakClientApiVersion)
	kcClientTypedList := kc.KeycloakClientList{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakClientList.Object, &kcClientTypedList)
	if err != nil {
		return &kcClientTypedList, err
	}

	return &kcClientTypedList, nil
}

// Converts Keycloak Users typed object to unstructured object
func ConvertKeycloakClientsTypedToUnstructured(kcUser *kc.KeycloakClientList) (*unstructured.UnstructuredList, error) {
	unstructuredClients := &unstructured.UnstructuredList{}
	unstructuredKeycloakClients := unstructuredListWithGVK(*unstructuredClients, kc.KeycloakClientGroup, kc.KeycloakClientVersion, kc.KeycloakClientKind, kc.KeycloakClientApiVersion)
	// convert typed to map
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kcUser)
	if err != nil {
		return unstructuredKeycloakClients, err
	}

	// convert map to unstructured type
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, unstructuredKeycloakClients)
	if err != nil {
		return unstructuredKeycloakClients, err
	}

	return unstructuredKeycloakClients, nil
}

func unstructuredWithGVK(unstructured unstructured.Unstructured, group, version, kind, apiVersion string) *unstructured.Unstructured {
	unstructured.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	})

	unstructured.SetAPIVersion(apiVersion)

	return &unstructured
}

func unstructuredListWithGVK(unstructured unstructured.UnstructuredList, group, version, kind, apiVersion string) *unstructured.UnstructuredList {
	for _, item := range unstructured.Items {
		item.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   group,
			Version: version,
			Kind:    kind,
		})

		item.SetAPIVersion(apiVersion)
	}

	return &unstructured
}
