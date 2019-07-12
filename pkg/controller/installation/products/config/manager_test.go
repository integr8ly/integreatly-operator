package config

import (
	"context"
	"strings"
	"testing"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	mockProductName   = "mock"
	mockConfigMapName = "test"
	mockNamespaceName = "test"
)

type mockConfig struct {
	productName string
	config      ProductConfig
}

func (c *mockConfig) GetProductName() v1alpha1.ProductName {
	return v1alpha1.ProductName(c.productName)
}

func (c *mockConfig) Read() ProductConfig {
	return c.config
}

func TestWriteConfig(t *testing.T) {
	tests := []struct {
		productName       string
		existingResources []runtime.Object
		toWrite           ConfigReadable
		expected          map[string]string
	}{
		// Test basic adding config
		{
			productName: mockProductName,
			existingResources: []runtime.Object{&v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      mockConfigMapName,
					Namespace: mockNamespaceName,
				},
			}},
			toWrite: newMockConfig(mockProductName, map[string]string{
				"testKey": "testVal",
			}),
			expected: map[string]string{
				"testKey": "testVal",
			},
		},
		// Test overwrite config
		{
			productName: mockProductName,
			existingResources: []runtime.Object{&v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      mockConfigMapName,
					Namespace: mockNamespaceName,
				},
				Data: map[string]string{
					"testKey1": "testVal1",
					"testKey2": "testVal2",
				},
			}},
			toWrite: newMockConfig(mockProductName, map[string]string{
				"testKey1": "newTestVal",
			}),
			expected: map[string]string{
				"testKey1": "newTestVal",
			},
		},
		// Test create configmap if one doesn't exist
		{
			productName:       mockProductName,
			existingResources: []runtime.Object{},
			toWrite: newMockConfig(mockProductName, map[string]string{
				"testKey": "testVal",
			}),
			expected: map[string]string{
				"testKey": "testVal",
			},
		},
	}
	for _, test := range tests {
		fakeClient := fake.NewFakeClient(test.existingResources...)

		mgr, err := NewManager(fakeClient, mockNamespaceName, mockConfigMapName)
		if err != nil {
			t.Fatalf("could not create manager %v", err)
		}
		if err = mgr.WriteConfig(test.toWrite); err != nil {
			t.Fatalf("could not write config %v", err)
		}
		readCfgMap := &v1.ConfigMap{
			ObjectMeta: v12.ObjectMeta{
				Name:      mockConfigMapName,
				Namespace: mockNamespaceName,
			},
		}
		fakeClient.Get(context.TODO(), client.ObjectKey{Name: mockConfigMapName, Namespace: mockNamespaceName}, readCfgMap)

		decoder := yaml.NewDecoder(strings.NewReader(readCfgMap.Data[test.productName]))
		testCfg := map[string]string{}
		decoder.Decode(testCfg)

		for key, value := range test.expected {
			if strings.Compare(testCfg[key], value) != 0 {
				t.Fatalf("expected %s but got %s for key %s", value, testCfg[key], key)
			}
		}
	}
}

func TestReadConfigForProduct(t *testing.T) {
	tests := []struct {
		productName       string
		existingResources []runtime.Object
		expected          map[string]string
	}{
		{
			productName: mockProductName,
			existingResources: []runtime.Object{&v1.ConfigMap{
				ObjectMeta: v12.ObjectMeta{
					Name:      mockConfigMapName,
					Namespace: mockNamespaceName,
				},
				Data: map[string]string{
					"mock": "testKey: testVal",
				},
			}},
			expected: map[string]string{
				"testKey": "testVal",
			},
		},
		{
			productName:       mockProductName,
			existingResources: []runtime.Object{},
			expected:          map[string]string{},
		},
	}

	for _, test := range tests {
		fakeClient := fake.NewFakeClient(test.existingResources...)
		mgr, err := NewManager(fakeClient, mockNamespaceName, mockConfigMapName)
		if err != nil {
			t.Fatalf("could not create manager %v", err)
		}
		config, err := mgr.ReadConfigForProduct(mockProductName)
		if err != nil {
			t.Fatalf("could not read config %v", err)
		}
		for key, value := range test.expected {
			if strings.Compare(config[key], value) != 0 {
				t.Fatalf("expected %s but got %s for key %s", value, config[key], key)
			}
		}
	}

}

func newMockConfig(productName string, vals map[string]string) *mockConfig {
	return &mockConfig{
		productName: productName,
		config:      vals,
	}
}
