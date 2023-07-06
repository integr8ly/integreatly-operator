package utils

import (
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// NamespacePredicate is a reusable predicate to watch only resources on a given
// namespace
func NamespacePredicate(namespace string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(m k8sclient.Object) bool {
		return m.GetNamespace() == namespace
	})
}

// NamePredicate is a reusable predicate to watch only resources on a given
// name
func NamePredicate(name string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(m k8sclient.Object) bool {
		return m.GetName() == name
	})
}
