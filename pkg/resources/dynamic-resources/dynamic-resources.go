package dynamic_resources

import (
	"context"
	kc "github.com/integr8ly/keycloak-client/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

// Converts Keycloak list unstructured object to typed keycloak list and sets GVK on object
func ConvertKeycloakListUnstructuredToTyped(u unstructured.UnstructuredList) (*kc.KeycloakList, error) {
	unstructuredKeycloakList := unstructuredListWithGVK(u, kc.KeycloakGroup, kc.KeycloakVersion, kc.KeycloakListKind, kc.KeycloakApiVersion)
	kcTypedList := kc.KeycloakList{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakList.Object, &kcTypedList)
	if err != nil {
		return &kcTypedList, err
	}

	return &kcTypedList, nil
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
	keycloakRealmTyped := kc.KeycloakRealm{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakRealm.Object, &keycloakRealmTyped)
	if err != nil {
		return &keycloakRealmTyped, err
	}

	return &keycloakRealmTyped, nil
}

// Converts Keycloak Realm list unstructured object to typed keycloak users list and sets GVK on object
func ConvertKeycloakRealmListUnstructuredToTyped(u unstructured.UnstructuredList) (*kc.KeycloakRealmList, error) {
	unstructuredKeycloakRealmList := unstructuredListWithGVK(u, kc.KeycloakRealmGroup, kc.KeycloakRealmVersion, kc.KeycloakRealmListKind, kc.KeycloakRealmApiVersion)
	kcRealmTypedList := kc.KeycloakRealmList{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredKeycloakRealmList.Object, &kcRealmTypedList)
	if err != nil {
		return &kcRealmTypedList, err
	}

	return &kcRealmTypedList, nil
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

// Retrieves keycloak client based on kc client typed object meta
func GetKeycloakClient(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakClient) (*kc.KeycloakClient, error) {
	kcClientUnstructured := CreateUnstructuredWithGVK(kc.KeycloakClientGroup, kc.KeycloakClientKind, kc.KeycloakClientVersion, "", "")

	err := client.Get(ctx, k8sclient.ObjectKey{
		Namespace: kcTyped.Namespace,
		Name:      kcTyped.Name,
	}, kcClientUnstructured)
	if err != nil {
		return nil, err
	}

	kcClientUpdated, err := ConvertKeycloakClientUnstructuredToTyped(*kcClientUnstructured)
	if err != nil {
		return nil, err
	}

	return kcClientUpdated, nil
}

// Creates or updates keycloak client based on kc client typed
func CreateOrUpdateKeycloakClient(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakClient) (controllerutil.OperationResult, *kc.KeycloakClient, error) {
	kcClientUnstructured, err := ConvertKeycloakClientTypedToUnstructured(&kcTyped)
	if err != nil {
		return controllerutil.OperationResultNone, nil, err
	}

	kcClientEmpty := CreateUnstructuredWithGVK(kc.KeycloakClientGroup, kc.KeycloakClientKind, kc.KeycloakClientVersion, kcTyped.Name, kcTyped.Namespace)

	opRes, err := controllerutil.CreateOrUpdate(ctx, client, kcClientEmpty, func() error {
		kcClientEmpty.Object = kcClientUnstructured.Object
		return nil
	})

	kcClientUpdated, err := GetKeycloakClient(ctx, client, kcTyped)
	if err != nil {
		return controllerutil.OperationResultNone, nil, err
	}

	return opRes, kcClientUpdated, nil
}

// Retrieves keycloak client based on kc client typed object meta
func GetKeycloakClientList(ctx context.Context, client k8sclient.Client, options []k8sclient.ListOption, kcTyped kc.KeycloakClientList) (*kc.KeycloakClientList, error) {
	kcClientUnstructured := &unstructured.UnstructuredList{}

	err := client.List(ctx, kcClientUnstructured, options...)
	if err != nil {
		return nil, err
	}

	kcClientUpdated, err := ConvertKeycloakClientsUnstructuredToTyped(*kcClientUnstructured)
	if err != nil {
		return nil, err
	}

	return kcClientUpdated, nil
}

// Retrieves keycloak client based on kc client typed object meta
func GetKeycloakUser(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakUser) (*kc.KeycloakUser, error) {
	kcUserUnstructured := &unstructured.Unstructured{}

	if kcTyped.Namespace != "" && kcTyped.Name != "" {
		kcUserUnstructured = CreateUnstructuredWithGVK(kc.KeycloakUserGroup, kc.KeycloakUserKind, kc.KeycloakUserVersion, kcTyped.Name, kcTyped.Namespace)
	} else {
		kcUserUnstructured = CreateUnstructuredWithGVK(kc.KeycloakUserGroup, kc.KeycloakUserKind, kc.KeycloakUserVersion, "", "")
	}

	err := client.Get(ctx, k8sclient.ObjectKey{
		Namespace: kcTyped.Namespace,
		Name:      kcTyped.Name,
	}, kcUserUnstructured)
	if err != nil {
		return nil, err
	}

	kcUserUpdated, err := ConvertKeycloakUserUnstructuredToTyped(*kcUserUnstructured)
	if err != nil {
		return nil, err
	}

	return kcUserUpdated, nil
}

// Creates or updates keycloak user based on kc user typed
func CreateOrUpdateKeycloakUser(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakUser) (controllerutil.OperationResult, *kc.KeycloakUser, error) {
	kcUserUnstructured, err := ConvertKeycloakUserTypedToUnstructured(&kcTyped)
	if err != nil {
		return controllerutil.OperationResultNone, nil, err
	}

	kcUserUnstructuredEmpty := CreateUnstructuredWithGVK(kc.KeycloakUserGroup, kc.KeycloakUserKind, kc.KeycloakUserVersion, kcTyped.Name, kcTyped.Namespace)

	opRes, err := controllerutil.CreateOrUpdate(ctx, client, kcUserUnstructuredEmpty, func() error {
		kcUserUnstructuredEmpty.Object = kcUserUnstructured.Object
		return nil
	})

	kcUserUpdated, err := GetKeycloakUser(ctx, client, kcTyped)
	if err != nil {
		return controllerutil.OperationResultNone, nil, err
	}

	return opRes, kcUserUpdated, nil
}

// Retrieves keycloak user list based on list opts and kc user list typed object
func GetKeycloakUserList(ctx context.Context, client k8sclient.Client, options []k8sclient.ListOption, kcTyped kc.KeycloakUserList) (*kc.KeycloakUserList, error) {
	kcUsersUnstructured := &unstructured.UnstructuredList{}

	err := client.List(ctx, kcUsersUnstructured, options...)
	if err != nil {
		return nil, err
	}

	kcUsersUpdated, err := ConvertKeycloakUsersUnstructuredToTyped(*kcUsersUnstructured)
	if err != nil {
		return nil, err
	}

	return kcUsersUpdated, nil
}

// Retrieves keycloak Realm based on kc client typed object meta
func GetKeycloakRealm(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakRealm) (*kc.KeycloakRealm, error) {
	kcRealmUnstructured := CreateUnstructuredWithGVK(kc.KeycloakRealmGroup, kc.KeycloakRealmKind, kc.KeycloakRealmVersion, "", "")

	err := client.Get(ctx, k8sclient.ObjectKey{
		Namespace: kcTyped.Namespace,
		Name:      kcTyped.Name,
	}, kcRealmUnstructured)
	if err != nil {
		return nil, err
	}

	kcRealmUpdated, err := ConvertKeycloakRealmUnstructuredToTyped(*kcRealmUnstructured)
	if err != nil {
		return nil, err
	}

	return kcRealmUpdated, nil
}

// Creates or updates keycloak Realm based on kc Realm typed
func CreateOrUpdateKeycloakRealm(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakRealm) (controllerutil.OperationResult, *kc.KeycloakRealm, error) {
	kcRealmUnstructured, err := ConvertKeycloakRealmTypedToUnstructured(&kcTyped)
	if err != nil {
		return controllerutil.OperationResultNone, nil, err
	}

	kcRealmEmpty := CreateUnstructuredWithGVK(kc.KeycloakRealmGroup, kc.KeycloakRealmKind, kc.KeycloakRealmVersion, kcTyped.Name, kcTyped.Namespace)

	opRes, err := controllerutil.CreateOrUpdate(ctx, client, kcRealmEmpty, func() error {
		kcRealmEmpty.Object = kcRealmUnstructured.Object
		return nil
	})

	kcRealmUpdated, err := GetKeycloakRealm(ctx, client, kcTyped)
	if err != nil {
		return controllerutil.OperationResultNone, nil, err
	}

	return opRes, kcRealmUpdated, nil
}

// Retrieves keycloak Realm based on kc Realm typed object meta
func GetKeycloakRealmList(ctx context.Context, client k8sclient.Client, options []k8sclient.ListOption, kcTyped kc.KeycloakRealmList) (*kc.KeycloakRealmList, error) {
	kcRealmUnstructured := &unstructured.UnstructuredList{}

	err := client.List(ctx, kcRealmUnstructured, options...)
	if err != nil {
		return nil, err
	}

	kcRealmUpdated, err := ConvertKeycloakRealmListUnstructuredToTyped(*kcRealmUnstructured)
	if err != nil {
		return nil, err
	}

	return kcRealmUpdated, nil
}

// Retrieves keycloak based on kc typed object meta
func GetKeycloak(ctx context.Context, client k8sclient.Client, kcTyped kc.Keycloak) (*kc.Keycloak, error) {
	kcUnstructured := CreateUnstructuredWithGVK(kc.KeycloakGroup, kc.KeycloakKind, kc.KeycloakVersion, "", "")

	err := client.Get(ctx, k8sclient.ObjectKey{
		Namespace: kcTyped.Namespace,
		Name:      kcTyped.Name,
	}, kcUnstructured)
	if err != nil {
		return nil, err
	}

	kcUpdated, err := ConvertKeycloakUnstructuredToTyped(*kcUnstructured)
	if err != nil {
		return nil, err
	}

	return kcUpdated, nil
}

// Creates or updates keycloak client based on kc client typed
func CreateOrUpdateKeycloak(ctx context.Context, client k8sclient.Client, kcTyped kc.Keycloak) (controllerutil.OperationResult, *kc.Keycloak, error) {
	kcUnstructured, err := ConvertKeycloakTypedToUnstructured(&kcTyped)
	if err != nil {
		return controllerutil.OperationResultNone, nil, err
	}

	kcEmpty := CreateUnstructuredWithGVK(kc.KeycloakGroup, kc.KeycloakKind, kc.KeycloakVersion, kcTyped.Name, kcTyped.Namespace)

	opRes, err := controllerutil.CreateOrUpdate(ctx, client, kcEmpty, func() error {
		kcEmpty.Object = kcUnstructured.Object
		return nil
	})

	kcUpdated, err := GetKeycloak(ctx, client, kcTyped)
	if err != nil {
		return controllerutil.OperationResultNone, nil, err
	}

	return opRes, kcUpdated, nil
}

// Retrieves keycloak based on kc typed object meta
func GetKeycloakList(ctx context.Context, client k8sclient.Client, options []k8sclient.ListOption, kcTyped kc.KeycloakList) (*kc.KeycloakList, error) {
	kcUnstructured := &unstructured.UnstructuredList{}

	err := client.List(ctx, kcUnstructured, options...)
	if err != nil {
		return nil, err
	}

	kcUpdated, err := ConvertKeycloakListUnstructuredToTyped(*kcUnstructured)
	if err != nil {
		return nil, err
	}

	return kcUpdated, nil
}

// Deletes keycloak based on kc typed object meta
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

// Deletes User based on kc typed object meta
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

// Deletes Realm based on kc typed object meta
func DeleteRealm(ctx context.Context, client k8sclient.Client, kcTyped kc.KeycloakRealm) error {
	kcTyped.SetFinalizers([]string{})
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
