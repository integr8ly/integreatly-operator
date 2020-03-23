package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	version   = "v1alpha1"
	groupName = "integreatly.org"
)

var (
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
	// SchemeGroupVersion is the group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: groupName, Version: version}
)

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&WebApp{},
		&WebAppList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

func init() {
	SchemeBuilder.Register(&WebApp{}, &WebAppList{})
}
