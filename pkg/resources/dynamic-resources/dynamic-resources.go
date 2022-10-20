package dynamic_resources

import (
	"context"

	kc "github.com/integr8ly/keycloak-client/pkg/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateUnstructuredWithGVK returns emtpy unstructured object with the GVK Meta, resource name and namespace are optional
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

// CreateUnstructuredListWithGVK returns emtpy unstructured list object with the GVK Meta, resource name and namespace are optional
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

// unstructuredListWithGVK adds GVK to the unstructured list object and item objects passed in
func unstructuredListWithGVK(unstructured unstructured.UnstructuredList, group, version, kind, listkind, apiVersion string) *unstructured.UnstructuredList {
	for _, item := range unstructured.Items {
		item.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   group,
			Version: version,
			Kind:    kind,
		})

		item.SetAPIVersion(apiVersion)
	}

	unstructured.SetKind(listkind)
	unstructured.SetAPIVersion(apiVersion)

	return &unstructured
}

// unstructuredWithGVK adds GVK to the unstructured object passed in
func unstructuredWithGVK(unstructured unstructured.Unstructured, group, version, kind, apiVersion string) *unstructured.Unstructured {
	unstructured.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	})

	unstructured.SetAPIVersion(apiVersion)

	return &unstructured
}

// ConvertKeycloakTypedToUnstructured converts keycloak typed object to unstructured and sets keycloak GVK on the object
func ConvertKeycloakTypedToUnstructured(keycloak *kc.Keycloak) (*unstructured.Unstructured, error) {
	kcUnstructed := &unstructured.Unstructured{}
	keycloak.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   kc.KeycloakGroup,
		Version: kc.KeycloakVersion,
		Kind:    kc.KeycloakKind,
	})

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(keycloak)
	if err != nil {
		return kcUnstructed, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, kcUnstructed)
	if err != nil {
		return kcUnstructed, err
	}

	return kcUnstructed, nil
}

// ConvertKeycloakUnstructuredToTyped converts unstructured object to keycloak typed and sets keycloak GVK on the object
func ConvertKeycloakUnstructuredToTyped(u unstructured.Unstructured) (*kc.Keycloak, error) {
	unstructuredKc := unstructuredWithGVK(u, kc.KeycloakGroup, kc.KeycloakVersion, kc.KeycloakKind, kc.KeycloakApiVersion)
	kcTyped := &kc.Keycloak{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKc.Object, &kcTyped)
	if err != nil {
		return kcTyped, err
	}

	return kcTyped, nil
}

// ConvertKeycloakListUnstructuredToTyped converts unstructured list object to typed keycloak list and sets GVK on object
func ConvertKeycloakListUnstructuredToTyped(u unstructured.UnstructuredList) (*kc.KeycloakList, error) {
	unstructuredKeycloakList := unstructuredListWithGVK(u, kc.KeycloakGroup, kc.KeycloakVersion, kc.KeycloakKind, kc.KeycloakListKind, kc.KeycloakApiVersion)
	kcTypedList := kc.KeycloakList{
		TypeMeta: v1.TypeMeta{
			Kind:       kc.KeycloakListKind,
			APIVersion: kc.KeycloakApiVersion,
		},
	}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakList.Object, &kcTypedList)
	if err != nil {
		return &kcTypedList, err
	}

	for _, item := range unstructuredKeycloakList.Items {
		kc := kc.Keycloak{
			TypeMeta: v1.TypeMeta{
				Kind:       kc.KeycloakKind,
				APIVersion: kc.KeycloakApiVersion,
			},
		}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &kc)
		if err != nil {
			return &kcTypedList, err
		}
		kcTypedList.Items = append(kcTypedList.Items, kc)
	}

	return &kcTypedList, nil
}

// ConvertKeycloakRealmTypedToUnstructured converts keycloak realm list typed object to unstructured list and sets GVK on each item
func ConvertKeycloakListTypedToUnstructuredList(kcList *kc.KeycloakList) (*unstructured.UnstructuredList, error) {
	unstructuredKeycloakList := CreateUnstructuredListWithGVK(kc.KeycloakGroup, kc.KeycloakKind, kc.KeycloakListKind, kc.KeycloakVersion, "", "")
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kcList)
	if err != nil {
		return unstructuredKeycloakList, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, unstructuredKeycloakList)
	if err != nil {
		return unstructuredKeycloakList, err
	}

	for _, keycloak := range kcList.Items {
		unstructuredKeycloak := CreateUnstructuredWithGVK(kc.KeycloakRealmGroup, kc.KeycloakRealmKind, kc.KeycloakRealmVersion, "", "")
		realmUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&keycloak)
		if err != nil {
			return unstructuredKeycloakList, err
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(realmUnstructured, unstructuredKeycloak)
		if err != nil {
			return unstructuredKeycloakList, err
		}
		unstructuredKeycloakList.Items = append(unstructuredKeycloakList.Items, *unstructuredKeycloak)
	}

	return unstructuredKeycloakList, nil
}

// GetKeycloak retrieves keycloak based on keycloak typed object meta, if object is not found, input is returned
func GetKeycloak(ctx context.Context, client k8sclient.Client, kcTyped kc.Keycloak) (*kc.Keycloak, error) {
	kcUnstructured := CreateUnstructuredWithGVK(kc.KeycloakGroup, kc.KeycloakKind, kc.KeycloakVersion, kcTyped.Name, kcTyped.Namespace)

	err := client.Get(ctx, k8sclient.ObjectKey{
		Namespace: kcTyped.Namespace,
		Name:      kcTyped.Name,
	}, kcUnstructured)
	if err != nil {
		return &kcTyped, err
	}

	kcUpdated, err := ConvertKeycloakUnstructuredToTyped(*kcUnstructured)
	if err != nil {
		return nil, err
	}

	return kcUpdated, nil
}

// GetKeycloakList retrieves keycloak list based on typed keycloak list and list option provided
func GetKeycloakList(ctx context.Context, client k8sclient.Client, options []k8sclient.ListOption, kcTyped kc.KeycloakList) (*kc.KeycloakList, error) {
	kcUnstructured := CreateUnstructuredListWithGVK(kc.KeycloakGroup, kc.KeycloakKind, kc.KeycloakListKind, kc.KeycloakVersion, "", "")

	err := client.List(ctx, kcUnstructured, options...)
	if err != nil {
		return &kcTyped, err
	}

	kcUpdated, err := ConvertKeycloakListUnstructuredToTyped(*kcUnstructured)
	if err != nil {
		return nil, err
	}

	return kcUpdated, nil
}

// DeleteKeycloak deletes the keycloak resource
func DeleteKeycloak(ctx context.Context, client k8sclient.Client, kcTyped kc.Keycloak) error {
	kcUnstructured, err := ConvertKeycloakTypedToUnstructured(&kcTyped)
	if err != nil {
		return err
	}

	err = client.Delete(ctx, kcUnstructured)
	if err != nil {
		return err
	}

	return nil
}

// ConvertKeycloakRealmTypedToUnstructured converts keycloak realm typed object to unstructured and sets GVK on object
func ConvertKeycloakRealmTypedToUnstructured(kcRealm *kc.KeycloakRealm) (*unstructured.Unstructured, error) {
	unstructuredRealm := &unstructured.Unstructured{}
	kcRealm.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   kc.KeycloakRealmGroup,
		Version: kc.KeycloakRealmVersion,
		Kind:    kc.KeycloakRealmKind,
	})
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(kcRealm)
	if err != nil {
		return unstructuredRealm, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, unstructuredRealm)
	if err != nil {
		return unstructuredRealm, err
	}

	return unstructuredRealm, nil
}

// ConvertKeycloakRealmUnstructuredToTyped converts unstructured object to keycloak realm and sets GVK on object
func ConvertKeycloakRealmUnstructuredToTyped(u unstructured.Unstructured) (*kc.KeycloakRealm, error) {
	unstructuredKeycloakRealm := unstructuredWithGVK(u, kc.KeycloakRealmGroup, kc.KeycloakRealmVersion, kc.KeycloakRealmKind, kc.KeycloakRealmApiVersion)
	keycloakRealmTyped := kc.KeycloakRealm{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakRealm.Object, &keycloakRealmTyped)
	if err != nil {
		return &keycloakRealmTyped, err
	}

	return &keycloakRealmTyped, nil
}

// ConvertKeycloakRealmListUnstructuredToTyped converts unstructured list to keycloak realm list and sets GVK on object
func ConvertKeycloakRealmListUnstructuredToTyped(u unstructured.UnstructuredList) (*kc.KeycloakRealmList, error) {
	unstructuredKeycloakRealmList := unstructuredListWithGVK(u, kc.KeycloakRealmGroup, kc.KeycloakRealmVersion, kc.KeycloakRealmKind, kc.KeycloakRealmListKind, kc.KeycloakRealmApiVersion)
	kcRealmTypedList := kc.KeycloakRealmList{
		TypeMeta: v1.TypeMeta{
			Kind:       kc.KeycloakRealmListKind,
			APIVersion: kc.KeycloakRealmApiVersion,
		},
	}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakRealmList.Object, &kcRealmTypedList)
	if err != nil {
		return &kcRealmTypedList, err
	}

	for _, item := range unstructuredKeycloakRealmList.Items {
		kcRealm := kc.KeycloakRealm{
			TypeMeta: v1.TypeMeta{
				Kind:       kc.KeycloakRealmKind,
				APIVersion: kc.KeycloakRealmApiVersion,
			},
		}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &kcRealm)
		if err != nil {
			return &kcRealmTypedList, err
		}
		kcRealmTypedList.Items = append(kcRealmTypedList.Items, kcRealm)
	}

	return &kcRealmTypedList, nil
}

// ConvertKeycloakRealmTypedToUnstructured converts keycloak realm list typed object to unstructured list and sets GVK on each item
func ConvertKeycloakRealmListTypedToUnstructuredList(kcRealmList *kc.KeycloakRealmList) (*unstructured.UnstructuredList, error) {
	unstructuredKeycloakRealmList := CreateUnstructuredListWithGVK(kc.KeycloakRealmGroup, kc.KeycloakRealmKind, kc.KeycloakRealmListKind, kc.KeycloakRealmVersion, "", "")
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kcRealmList)
	if err != nil {
		return unstructuredKeycloakRealmList, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, unstructuredKeycloakRealmList)
	if err != nil {
		return unstructuredKeycloakRealmList, err
	}

	for _, realm := range kcRealmList.Items {
		unstructuredKeycloakRealm := CreateUnstructuredWithGVK(kc.KeycloakRealmGroup, kc.KeycloakRealmKind, kc.KeycloakRealmVersion, "", "")
		realmUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&realm)
		if err != nil {
			return unstructuredKeycloakRealmList, err
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(realmUnstructured, unstructuredKeycloakRealm)
		if err != nil {
			return unstructuredKeycloakRealmList, err
		}
		unstructuredKeycloakRealmList.Items = append(unstructuredKeycloakRealmList.Items, *unstructuredKeycloakRealm)
	}

	return unstructuredKeycloakRealmList, nil
}

// GetKeycloakRealm retrieves keycloak realm based on keycloak realm typed object meta
func GetKeycloakRealm(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakRealm) (*kc.KeycloakRealm, error) {
	kcRealmUnstructured := CreateUnstructuredWithGVK(kc.KeycloakRealmGroup, kc.KeycloakRealmKind, kc.KeycloakRealmVersion, kcTyped.Name, kcTyped.Namespace)

	err := client.Get(ctx, k8sclient.ObjectKey{
		Namespace: kcTyped.Namespace,
		Name:      kcTyped.Name,
	}, kcRealmUnstructured)
	if err != nil {
		return &kcTyped, err
	}

	kcRealmUpdated, err := ConvertKeycloakRealmUnstructuredToTyped(*kcRealmUnstructured)
	if err != nil {
		return nil, err
	}

	return kcRealmUpdated, nil
}

// GetKeycloakRealmList retrieves keycloak realm list based on typed keycloak realm and list option provided
func GetKeycloakRealmList(ctx context.Context, client k8sclient.Client, options []k8sclient.ListOption, kcTyped kc.KeycloakRealmList) (*kc.KeycloakRealmList, error) {
	kcRealmUnstructured := CreateUnstructuredListWithGVK(kc.KeycloakRealmGroup, kc.KeycloakRealmKind, kc.KeycloakRealmListKind, kc.KeycloakRealmVersion, "", "")

	err := client.List(ctx, kcRealmUnstructured, options...)
	if err != nil {
		return &kcTyped, err
	}

	kcRealmUpdated, err := ConvertKeycloakRealmListUnstructuredToTyped(*kcRealmUnstructured)
	if err != nil {
		return nil, err
	}

	return kcRealmUpdated, nil
}

// DeleteRealm deletes the keycloak realm resource
func DeleteRealm(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakRealm) error {
	kcRealmUnstructured, err := ConvertKeycloakRealmTypedToUnstructured(&kcTyped)
	if err != nil {
		return err
	}

	err = client.Delete(ctx, kcRealmUnstructured)
	if err != nil {
		return err
	}

	return nil
}

// ConvertKeycloakUserUnstructuredToTyped converts keycloak user unstructured object to typed keycloak user object and sets GVK on object
func ConvertKeycloakUserUnstructuredToTyped(u unstructured.Unstructured) (*kc.KeycloakUser, error) {
	unstructuredKeycloakUser := unstructuredWithGVK(u, kc.KeycloakUserGroup, kc.KeycloakUserVersion, kc.KeycloakUserKind, kc.KeycloakUserApiVersion)
	kcUserTyped := kc.KeycloakUser{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakUser.Object, &kcUserTyped)
	if err != nil {
		return &kcUserTyped, err
	}

	return &kcUserTyped, nil
}

// ConvertKeycloakUserTypedToUnstructured converts keycloak user typed object to unstructured object and sets GVK on object
func ConvertKeycloakUserTypedToUnstructured(kcUser *kc.KeycloakUser) (*unstructured.Unstructured, error) {
	unstructuredUser := &unstructured.Unstructured{}
	kcUser.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   kc.KeycloakUserGroup,
		Version: kc.KeycloakUserVersion,
		Kind:    kc.KeycloakUserKind,
	})
	kcUser.APIVersion = kc.KeycloakUserApiVersion
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kcUser)
	if err != nil {
		return unstructuredUser, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, unstructuredUser)
	if err != nil {
		return unstructuredUser, err
	}

	return unstructuredUser, nil
}

// ConvertKeycloakUserListUnstructuredToTyped converts unstructured list object to typed keycloak user list and sets up GVK on the object
func ConvertKeycloakUserListUnstructuredToTyped(u unstructured.UnstructuredList) (*kc.KeycloakUserList, error) {
	unstructuredKeycloakUserList := unstructuredListWithGVK(u, kc.KeycloakUserGroup, kc.KeycloakUserVersion, kc.KeycloakUserKind, kc.KeycloakUserListKind, kc.KeycloakUserApiVersion)
	kcUserTypedList := kc.KeycloakUserList{
		TypeMeta: v1.TypeMeta{
			Kind:       kc.KeycloakUserListKind,
			APIVersion: kc.KeycloakUserApiVersion,
		},
	}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakUserList.Object, &kcUserTypedList)
	if err != nil {
		return &kcUserTypedList, err
	}

	for _, item := range unstructuredKeycloakUserList.Items {
		kcUser := kc.KeycloakUser{
			TypeMeta: v1.TypeMeta{
				Kind:       kc.KeycloakUserKind,
				APIVersion: kc.KeycloakUserApiVersion,
			},
		}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &kcUser)
		if err != nil {
			return &kcUserTypedList, err
		}
		kcUserTypedList.Items = append(kcUserTypedList.Items, kcUser)
	}

	return &kcUserTypedList, nil
}

// ConvertKeycloakUsersTypedToUnstructured converts keycloak users list typed object to unstructured list and sets GVK on each item
func ConvertKeycloakUserListTypedToUnstructured(kcUserList *kc.KeycloakUserList) (*unstructured.UnstructuredList, error) {
	unstructuredKeycloakUserList := CreateUnstructuredListWithGVK(kc.KeycloakUserGroup, kc.KeycloakClientKind, kc.KeycloakListKind, kc.KeycloakClientVersion, "", "")
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kcUserList)
	if err != nil {
		return unstructuredKeycloakUserList, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, unstructuredKeycloakUserList)
	if err != nil {
		return unstructuredKeycloakUserList, err
	}

	for _, user := range kcUserList.Items {
		unstructuredKeycloakUser := CreateUnstructuredWithGVK(kc.KeycloakUserGroup, kc.KeycloakUserKind, kc.KeycloakUserVersion, "", "")
		userUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&user)
		if err != nil {
			return unstructuredKeycloakUserList, err
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(userUnstructured, unstructuredKeycloakUser)
		if err != nil {
			return unstructuredKeycloakUserList, err
		}
		unstructuredKeycloakUserList.Items = append(unstructuredKeycloakUserList.Items, *unstructuredKeycloakUser)
	}

	return unstructuredKeycloakUserList, nil
}

// GetKeycloakUser retrieves keycloak user based on keycloak user typed object meta and sets GVK
func GetKeycloakUser(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakUser) (*kc.KeycloakUser, error) {
	kcUserUnstructured := CreateUnstructuredWithGVK(kc.KeycloakUserGroup, kc.KeycloakUserKind, kc.KeycloakUserVersion, kcTyped.Name, kcTyped.Namespace)

	err := client.Get(ctx, k8sclient.ObjectKey{
		Namespace: kcTyped.Namespace,
		Name:      kcTyped.Name,
	}, kcUserUnstructured)
	if err != nil {
		return &kcTyped, err
	}

	kcUserUpdated, err := ConvertKeycloakUserUnstructuredToTyped(*kcUserUnstructured)
	if err != nil {
		return nil, err
	}

	return kcUserUpdated, nil
}

// GetKeycloakUserList retrieves keycloak user list based on list opts and keycloak user list typed object
func GetKeycloakUserList(ctx context.Context, client k8sclient.Client, options []k8sclient.ListOption, kcTyped kc.KeycloakUserList) (*kc.KeycloakUserList, error) {
	kcUsersUnstructured := CreateUnstructuredListWithGVK(kc.KeycloakUserGroup, kc.KeycloakUserKind, kc.KeycloakUserListKind, kc.KeycloakUserVersion, "", "")

	err := client.List(ctx, kcUsersUnstructured, options...)
	if err != nil {
		return &kcTyped, err
	}

	kcUsersUpdated, err := ConvertKeycloakUserListUnstructuredToTyped(*kcUsersUnstructured)
	if err != nil {
		return nil, err
	}

	return kcUsersUpdated, nil
}

// DeleteUser deletes the keycloak user resource
func DeleteUser(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakUser) error {
	kcUserUnstructured, err := ConvertKeycloakUserTypedToUnstructured(&kcTyped)
	if err != nil {
		return err
	}

	err = client.Delete(ctx, kcUserUnstructured)
	if err != nil {
		return err
	}

	return nil
}

// ConvertKeycloakClientTypedToUnstructured converts keycloak client typed to unstructured and sets GVK on object
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

// ConvertKeycloakClientUnstructuredToTyped converts unstructured to keycloak client typed and sets GVK on object
func ConvertKeycloakClientUnstructuredToTyped(u unstructured.Unstructured) (*kc.KeycloakClient, error) {
	unstructuredKeycloakClient := unstructuredWithGVK(u, kc.KeycloakClientGroup, kc.KeycloakClientVersion, kc.KeycloakClientKind, kc.KeycloakClientApiVersion)
	kcClientTyped := kc.KeycloakClient{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakClient.Object, &kcClientTyped)
	if err != nil {
		return &kcClientTyped, err
	}

	return &kcClientTyped, nil
}

// onvertKeycloakClientListUnstructuredToTyped converts unstructured list to typed keycloak user list and sets GVK on object
func ConvertKeycloakClientListUnstructuredToTyped(u unstructured.UnstructuredList) (*kc.KeycloakClientList, error) {
	unstructuredKeycloakClientList := unstructuredListWithGVK(u, kc.KeycloakClientGroup, kc.KeycloakClientVersion, kc.KeycloakClientKind, kc.KeycloakClientListKind, kc.KeycloakClientApiVersion)
	kcClientTypedList := kc.KeycloakClientList{
		TypeMeta: v1.TypeMeta{
			Kind:       kc.KeycloakClientListKind,
			APIVersion: kc.KeycloakClientApiVersion,
		},
		Items: []kc.KeycloakClient{},
	}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakClientList.Object, &kcClientTypedList)
	if err != nil {
		return &kcClientTypedList, err
	}

	for _, item := range unstructuredKeycloakClientList.Items {
		kcClient := kc.KeycloakClient{
			TypeMeta: v1.TypeMeta{
				Kind:       kc.KeycloakClientKind,
				APIVersion: kc.KeycloakClientApiVersion,
			},
		}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &kcClient)
		if err != nil {
			return &kcClientTypedList, err
		}
		kcClientTypedList.Items = append(kcClientTypedList.Items, kcClient)
	}

	return &kcClientTypedList, nil
}

// ConvertKeycloakClientListTypedToUnstructured converts keycloak client list typed object to unstructured list and sets GVK
func ConvertKeycloakClientListTypedToUnstructured(kcClientList *kc.KeycloakClientList) (*unstructured.UnstructuredList, error) {
	unstructuredKeycloakClientList := CreateUnstructuredListWithGVK(kc.KeycloakClientGroup, kc.KeycloakClientKind, kc.KeycloakClientListKind, kc.KeycloakClientVersion, "", "")
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&kcClientList)
	if err != nil {
		return unstructuredKeycloakClientList, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, unstructuredKeycloakClientList)
	if err != nil {
		return unstructuredKeycloakClientList, err
	}

	for _, client := range kcClientList.Items {
		unstructuredKeycloakClient := CreateUnstructuredWithGVK(kc.KeycloakClientGroup, kc.KeycloakClientKind, kc.KeycloakClientVersion, "", "")
		clientUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&client)
		if err != nil {
			return unstructuredKeycloakClientList, err
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(clientUnstructured, unstructuredKeycloakClient)
		if err != nil {
			return unstructuredKeycloakClientList, err
		}
		unstructuredKeycloakClientList.Items = append(unstructuredKeycloakClientList.Items, *unstructuredKeycloakClient)
	}

	return unstructuredKeycloakClientList, nil
}

// GetKeycloakClient retrieves keycloak client based on keycloak client typed object meta
func GetKeycloakClient(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakClient) (*kc.KeycloakClient, error) {
	kcClientUnstructured := CreateUnstructuredWithGVK(kc.KeycloakClientGroup, kc.KeycloakClientKind, kc.KeycloakClientVersion, kcTyped.Name, kcTyped.Namespace)

	err := client.Get(ctx, k8sclient.ObjectKey{
		Namespace: kcTyped.Namespace,
		Name:      kcTyped.Name,
	}, kcClientUnstructured)
	if err != nil {
		return &kcTyped, err
	}

	kcClientUpdated, err := ConvertKeycloakClientUnstructuredToTyped(*kcClientUnstructured)
	if err != nil {
		return nil, err
	}

	return kcClientUpdated, nil
}

// GetKeycloakClientList retrieves keycloak client list typed based on keycloak client list typed object meta and list options
func GetKeycloakClientList(ctx context.Context, client k8sclient.Client, options []k8sclient.ListOption, kcTyped kc.KeycloakClientList) (*kc.KeycloakClientList, error) {
	kcClientUnstructured := CreateUnstructuredListWithGVK(kc.KeycloakClientGroup, kc.KeycloakClientKind, kc.KeycloakClientListKind, kc.KeycloakClientVersion, "", "")

	err := client.List(ctx, kcClientUnstructured, options...)
	if err != nil {
		return &kcTyped, err
	}

	kcClientUpdated, err := ConvertKeycloakClientListUnstructuredToTyped(*kcClientUnstructured)
	if err != nil {
		return nil, err
	}

	return kcClientUpdated, nil
}

// DeleteClient deletes the keycloak client resource
func DeleteClient(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakClient) error {
	kcClientUnstructured, err := ConvertKeycloakClientTypedToUnstructured(&kcTyped)
	if err != nil {
		return err
	}

	err = client.Delete(ctx, kcClientUnstructured)
	if err != nil {
		return err
	}

	return nil
}
