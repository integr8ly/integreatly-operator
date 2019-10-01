package nspredicate

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/event"
)

// NewNamespacePredicateFromEnvVar looks up the provided environment
// variable, parses the value, and returns a NamespacePredicate
func NewFromEnvVar(envVar string) (NamespacePredicate, error) {
	namespaces, found := os.LookupEnv(envVar)
	if !found {
		return NamespacePredicate{}, fmt.Errorf("Environment variable %s not found", envVar)
	}
	return NamespacePredicate{Namespaces: strings.Split(namespaces, ",")}, nil
}

type NamespacePredicate struct {
	Namespaces []string
}

func (p NamespacePredicate) Create(e event.CreateEvent) bool {
	return p.isValidNamespace(e.Meta.GetNamespace())
}

func (p NamespacePredicate) Delete(e event.DeleteEvent) bool {
	return p.isValidNamespace(e.Meta.GetNamespace())
}

func (p NamespacePredicate) Update(e event.UpdateEvent) bool {
	return p.isValidNamespace(e.MetaOld.GetNamespace())
}

func (p NamespacePredicate) Generic(e event.GenericEvent) bool {
	return p.isValidNamespace(e.Meta.GetNamespace())
}

func (p NamespacePredicate) isValidNamespace(namespace string) bool {
	for _, ns := range p.Namespaces {
		if ns == namespace {
			return true
		}
	}
	return false
}
