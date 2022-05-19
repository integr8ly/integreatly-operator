package utils

import (
	"fmt"
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// JUnitFileName Allow adding a prefix into the junit file name.
// prefix using the "TEST_PREFIX" env var
func JUnitFileName(suiteName string) string {
	testPrefix := os.Getenv("TEST_PREFIX")
	if len(testPrefix) > 0 {
		return fmt.Sprintf("junit-%s-%s.xml", testPrefix, suiteName)
	}
	return fmt.Sprintf("junit-%s.xml", suiteName)
}

// SpecDescription Allow adding a prefix into the test spec description.
// prefix using the "TEST_PREFIX" env var
func SpecDescription(spec string) string {
	testPrefix := os.Getenv("TEST_PREFIX")
	if len(testPrefix) > 0 {
		return fmt.Sprintf("%s %s", testPrefix, spec)
	}
	return spec
}

//OperatorIsHiveManaged Returns true if addon is manged by hive
func OperatorIsHiveManaged(client k8sclient.Client, inst *v1alpha1.RHMI) (bool, error) {
	ns := &v1.Namespace{
		ObjectMeta: v12.ObjectMeta{
			Name: inst.Namespace,
		},
	}
	err := client.Get(context.TODO(), k8sclient.ObjectKey{Name: ns.Name}, ns)
	if err != nil {
		return false, fmt.Errorf("could not retrieve %s namespace: ", err)
	}

	labels := ns.GetLabels()
	value, ok := labels["hive.openshift.io/managed"]
	if ok {
		if value == "true" {
			logrus.Info("operator is hive managed")
			return true, nil
		}
	}

	return false, nil
}
