package config

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewManager(client pkgclient.Client, instance *v1alpha1.Installation) (*Manager, error) {
	return &Manager{client: client, instance: instance}, nil
}

type ConfigReadWriter interface {
	GetProductConfig(product v1alpha1.ProductName) (v1alpha1.ProductConfig, error)
	WriteConfig(config ConfigReadable) error
	GetAMQStreams() (*AMQStreams, error)
}

type ConfigReadable interface {
	Read() v1alpha1.ProductConfig
	GetProductName() v1alpha1.ProductName
}

type Manager struct {
	client   pkgclient.Client
	instance *v1alpha1.Installation
}

func (m *Manager) GetAMQStreams() (*AMQStreams, error) {
	config, _ := m.GetProductConfig(v1alpha1.ProductAMQStreams)
	return newAMQStreams(config), nil
}
func (m *Manager) WriteConfig(config ConfigReadable) error {
	m.SetProductConfig(config.GetProductName(), config.Read())
	return m.client.Update(context.TODO(), m.instance)
}

func (m *Manager) GetProductConfig(product v1alpha1.ProductName) (v1alpha1.ProductConfig, error) {
	return m.instance.Status.ProductConfig[product], nil
}

func (m *Manager) SetProductConfig(name v1alpha1.ProductName, config v1alpha1.ProductConfig) error {
	if m.instance.Status.ProductConfig == nil {
		m.instance.Status.ProductConfig = make(map[v1alpha1.ProductName]v1alpha1.ProductConfig)
	}
	m.instance.Status.ProductConfig[name] = config
	return nil
}
