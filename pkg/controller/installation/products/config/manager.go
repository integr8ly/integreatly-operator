package config

import (
	"context"
	"strings"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	errors2 "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ProductConfig map[string]string

func NewManager(ctx context.Context, client pkgclient.Client, namespace string, configMapName string) (*Manager, error) {
	cfgmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      configMapName,
		},
	}
	err := client.Get(ctx, pkgclient.ObjectKey{Name: configMapName, Namespace: namespace}, cfgmap)
	if !errors.IsNotFound(err) && err != nil {
		return nil, err
	}
	return &Manager{Client: client, Namespace: namespace, cfgmap: cfgmap, context: ctx}, nil
}

//go:generate moq -out ConfigReadWriter_moq.go . ConfigReadWriter
type ConfigReadWriter interface {
	readConfigForProduct(product v1alpha1.ProductName) (ProductConfig, error)
	GetOauthClientsSecretName() string
	WriteConfig(config ConfigReadable) error
	ReadAMQStreams() (*AMQStreams, error)
	ReadRHSSO() (*RHSSO, error)
	ReadRHSSOUser() (*RHSSOUser, error)
	ReadCodeReady() (*CodeReady, error)
	ReadThreeScale() (*ThreeScale, error)
	ReadMobileSecurityService() (*MobileSecurityService, error)
	ReadFuse() (*Fuse, error)
	ReadFuseOnOpenshift() (*FuseOnOpenshift, error)
	ReadAMQOnline() (*AMQOnline, error)
	ReadNexus() (*Nexus, error)
	ReadLauncher() (*Launcher, error)
	GetOperatorNamespace() string
	ReadSolutionExplorer() (*SolutionExplorer, error)
	ReadMonitoring() (*Monitoring, error)
	ReadProduct(product v1alpha1.ProductName) (ConfigReadable, error)
	ReadUps() (*Ups, error)
}

//go:generate moq -out ConfigReadable_moq.go . ConfigReadable
type ConfigReadable interface {
	Read() ProductConfig
	GetProductName() v1alpha1.ProductName
	GetProductVersion() v1alpha1.ProductVersion
	GetHost() string
}

type Manager struct {
	Client    pkgclient.Client
	Namespace string
	cfgmap    *v1.ConfigMap
	context   context.Context
}

func (m *Manager) ReadProduct(product v1alpha1.ProductName) (ConfigReadable, error) {
	switch product {
	case v1alpha1.Product3Scale:
		return m.ReadThreeScale()
	case v1alpha1.ProductAMQOnline:
		return m.ReadAMQOnline()
	case v1alpha1.ProductRHSSO:
		return m.ReadRHSSO()
	case v1alpha1.ProductRHSSOUser:
		return m.ReadRHSSOUser()
	case v1alpha1.ProductAMQStreams:
		return m.ReadAMQStreams()
	case v1alpha1.ProductCodeReadyWorkspaces:
		return m.ReadCodeReady()
	case v1alpha1.ProductFuse:
		return m.ReadFuse()
	case v1alpha1.ProductFuseOnOpenshift:
		return m.ReadFuseOnOpenshift()
	case v1alpha1.ProductNexus:
		return m.ReadNexus()
	case v1alpha1.ProductSolutionExplorer:
		return m.ReadSolutionExplorer()
	case v1alpha1.ProductUps:
		return m.ReadUps()
	case v1alpha1.ProductMobileSecurityService:
		return m.ReadMobileSecurityService()
	}

	return nil, errors2.New("no config found for product " + string(product))
}

func (m *Manager) ReadSolutionExplorer() (*SolutionExplorer, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductSolutionExplorer)
	if err != nil {
		return nil, err
	}
	return NewSolutionExplorer(config), nil
}

func (m *Manager) GetOperatorNamespace() string {
	return m.Namespace
}

func (m *Manager) GetOauthClientsSecretName() string {
	return "oauth-client-secrets"
}

func (m *Manager) ReadAMQStreams() (*AMQStreams, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductAMQStreams)
	if err != nil {
		return nil, err
	}
	return NewAMQStreams(config), nil
}

func (m *Manager) ReadThreeScale() (*ThreeScale, error) {
	config, err := m.readConfigForProduct(v1alpha1.Product3Scale)
	if err != nil {
		return nil, err
	}
	return NewThreeScale(config), nil
}

func (m *Manager) ReadCodeReady() (*CodeReady, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductCodeReadyWorkspaces)
	if err != nil {
		return nil, err
	}
	return NewCodeReady(config), nil
}

func (m *Manager) ReadFuse() (*Fuse, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductFuse)
	if err != nil {
		return nil, err
	}
	return NewFuse(config), nil
}

func (m *Manager) ReadFuseOnOpenshift() (*FuseOnOpenshift, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductFuseOnOpenshift)
	if err != nil {
		return nil, err
	}
	return NewFuseOnOpenshift(config), nil
}

func (m *Manager) ReadRHSSO() (*RHSSO, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductRHSSO)
	if err != nil {
		return nil, err
	}
	return NewRHSSO(config), nil
}

func (m *Manager) ReadRHSSOUser() (*RHSSOUser, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductRHSSOUser)
	if err != nil {
		return nil, err
	}
	return NewRHSSOUser(config), nil
}

func (m *Manager) ReadAMQOnline() (*AMQOnline, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductAMQOnline)
	if err != nil {
		return nil, err
	}
	return NewAMQOnline(config), nil
}

func (m *Manager) ReadNexus() (*Nexus, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductNexus)
	if err != nil {
		return nil, err
	}
	return NewNexus(config), nil
}

func (m *Manager) ReadLauncher() (*Launcher, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductLauncher)
	if err != nil {
		return nil, err
	}
	return NewLauncher(config), nil
}

func (m *Manager) ReadMonitoring() (*Monitoring, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductMonitoring)
	if err != nil {
		return nil, err
	}
	return NewMonitoring(config), nil
}

func (m *Manager) ReadUps() (*Ups, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductUps)
	if err != nil {
		return nil, err
	}

	return NewUps(config), nil
}

func (m *Manager) ReadMobileSecurityService() (*MobileSecurityService, error) {
	config, err := m.readConfigForProduct(v1alpha1.ProductMobileSecurityService)
	if err != nil {
		return nil, err
	}
	return NewMobileSecurityService(config), nil
}

func (m *Manager) WriteConfig(config ConfigReadable) error {
	stringConfig, err := yaml.Marshal(config.Read())
	err = m.Client.Get(m.context, pkgclient.ObjectKey{Name: m.cfgmap.Name, Namespace: m.Namespace}, m.cfgmap)
	if errors.IsNotFound(err) {
		m.cfgmap.Data = map[string]string{string(config.GetProductName()): string(stringConfig)}
		return m.Client.Create(m.context, m.cfgmap)
	} else {
		if m.cfgmap.Data == nil {
			m.cfgmap.Data = map[string]string{}
		}
		m.cfgmap.Data[string(config.GetProductName())] = string(stringConfig)
		return m.Client.Update(m.context, m.cfgmap)
	}
}

func (m *Manager) readConfigForProduct(product v1alpha1.ProductName) (ProductConfig, error) {
	config := m.cfgmap.Data[string(product)]
	decoder := yaml.NewDecoder(strings.NewReader(config))
	retConfig := ProductConfig{}
	if config == "" {
		return retConfig, nil
	}
	if err := decoder.Decode(retConfig); err != nil {
		return nil, errors2.Wrap(err, "failed to decode product config for "+string(product))
	}
	return retConfig, nil
}
