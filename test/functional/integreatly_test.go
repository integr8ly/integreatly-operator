package functional

import (
	"os"
	"testing"

	"github.com/integr8ly/integreatly-operator/test/common"
	runtimeConfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestIntegreatly(t *testing.T) {
	config, err := runtimeConfig.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Integreatly Happy Path Tests", func(t *testing.T) {
		for _, test := range common.ALL_TESTS {
			t.Run(test.Description, func(t *testing.T) {
				testingContext, err := common.NewTestingContext(config)
				if err != nil {
					t.Fatal("failed to create testing context", err)
				}
				test.Test(t, testingContext)
			})
		}
		for _, test := range common.HAPPY_PATH_TESTS {
			t.Run(test.Description, func(t *testing.T) {
				testingContext, err := common.NewTestingContext(config)
				if err != nil {
					t.Fatal("failed to create testing context", err)
				}
				test.Test(t, testingContext)
			})
		}
		for _, test := range FUNCTIONAL_TESTS {
			t.Run(test.Description, func(t *testing.T) {
				testingContext, err := common.NewTestingContext(config)
				if err != nil {
					t.Fatal("failed to create testing context", err)
				}
				test.Test(t, testingContext)
			})
		}
	})

	t.Run("Integreatly Destructive Tests", func(t *testing.T) {
		// Do not execute these tests unless DESTRUCTIVE is set to true
		if os.Getenv("DESTRUCTIVE") != "true" {
			t.Skip("Skipping Destructive tests as DESTRUCTIVE env var is not set to true")
		}

		for _, test := range common.DESTRUCTIVE_TESTS {
			t.Run(test.Description, func(t *testing.T) {
				testingContext, err := common.NewTestingContext(config)
				if err != nil {
					t.Fatal("failed to create testing context", err)
				}
				test.Test(t, testingContext)
			})
		}
	})
}
