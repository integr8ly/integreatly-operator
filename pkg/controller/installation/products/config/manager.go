package config

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type ProductConfig map[string]string

func NewManager(client pkgclient.Client, namespace string, configMapName string) (*Manager, error) {
	cfgmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      configMapName,
		},
	}
	err := client.Get(context.TODO(), pkgclient.ObjectKey{Name: configMapName, Namespace: namespace}, cfgmap)
	if !errors.IsNotFound(err) && err != nil {
		return nil, err
	}
	return &Manager{Client: client, Namespace: namespace, cfgmap: cfgmap}, nil
}

type ConfigReadWriter interface {
	ReadConfigForProduct(product v1alpha1.ProductName) (ProductConfig, error)
	WriteConfig(config ConfigReadable) error
	ReadAMQStreams() (*AMQStreams, error)
}

type ConfigReadable interface {
	Read() ProductConfig
	GetProductName() v1alpha1.ProductName
}

type Manager struct {
	Client    pkgclient.Client
	Namespace string
	cfgmap    *v1.ConfigMap
}

func (m *Manager) ReadAMQStreams() (*AMQStreams, error) {
	config, err := m.ReadConfigForProduct(v1alpha1.ProductAMQStreams)
	if err != nil {
		return nil, err
	}
	return newAMQStreams(config), nil
}
func (m *Manager) WriteConfig(config ConfigReadable) error {
	stringConfig, err := yaml.Marshal(config.Read())
	err = m.Client.Get(context.TODO(), pkgclient.ObjectKey{Name: m.cfgmap.Name, Namespace: m.Namespace}, m.cfgmap)
	if errors.IsNotFound(err) {
		m.cfgmap.Data = map[string]string{string(config.GetProductName()): string(stringConfig)}
		return m.Client.Create(context.TODO(), m.cfgmap)
	} else {
		if m.cfgmap.Data == nil {
			m.cfgmap.Data = map[string]string{}
		}
		m.cfgmap.Data[string(config.GetProductName())] = string(stringConfig)
		return m.Client.Update(context.TODO(), m.cfgmap)
	}
}

func (m *Manager) ReadConfigForProduct(product v1alpha1.ProductName) (ProductConfig, error) {
	config := m.cfgmap.Data[string(product)]
	decoder := yaml.NewDecoder(strings.NewReader(config))
	retConfig := ProductConfig{}
	decoder.Decode(retConfig)

	return retConfig, nil
}
