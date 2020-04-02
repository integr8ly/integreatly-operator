package functional

import (
	"testing"

	"github.com/integr8ly/integreatly-operator/test/common"
	runtimeConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestIntegreatly(t *testing.T) {
	config, err := runtimeConfig.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	testingContext, err := common.NewTestingContext(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Integreatly Happy Path Tests", func(t *testing.T) {
		for _, test := range common.ALL_TESTS {
			t.Run(test.Description, func(t *testing.T) {
				test.Test(t, testingContext)
			})
		}
		for _, test := range common.AFTER_INSTALL_TESTS {
			t.Run(test.Description, func(t *testing.T) {
				test.Test(t, testingContext)
			})
		}
		for _, test := range FUNCTIONAL_TESTS {
			t.Run(test.Description, func(t *testing.T) {
				testingContext, err = common.NewTestingContext(config)
				if err != nil {
					t.Fatal("failed to create testing context", err)
				}
				test.Test(t, testingContext)
			})
		}
	})
}
