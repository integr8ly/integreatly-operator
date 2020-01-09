package owner

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	INTEGREATLY_OWNER_NAMESPACE = "integreatly-namespace"
	INTEGREATLY_OWNER_NAME      = "integreatly-name"
)

func AddIntegreatlyOwnerAnnotations(obj metav1.Object, owner metav1.Object) metav1.Object {
	if obj == nil || owner == nil {
		return nil
	}
	ant := obj.GetAnnotations()
	if ant == nil {
		ant = map[string]string{}
	}
	ant[INTEGREATLY_OWNER_NAME] = owner.GetName()
	ant[INTEGREATLY_OWNER_NAMESPACE] = owner.GetNamespace()
	obj.SetAnnotations(ant)
	return obj
}
