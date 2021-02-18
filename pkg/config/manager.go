package config

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	"gopkg.in/yaml.v2"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ProductConfig map[string]string

func NewManager(ctx context.Context, client k8sclient.Client, namespace string, configMapName string, installation *integreatlyv1alpha1.RHMI) (*Manager, error) {
	cfgmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      configMapName,
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: configMapName, Namespace: namespace}, cfgmap)
	if !errors.IsNotFound(err) && err != nil {
		return nil, err
	}
	return &Manager{Client: client, Namespace: namespace, cfgmap: cfgmap, context: ctx, installation: installation}, nil
}

//go:generate moq -out ConfigReadWriter_moq.go . ConfigReadWriter
type ConfigReadWriter interface {
	readConfigForProduct(product integreatlyv1alpha1.ProductName) (ProductConfig, error)
	GetOauthClientsSecretName() string
	GetGHOauthClientsSecretName() string
	GetBackupsSecretName() string
	WriteConfig(config ConfigReadable) error
	ReadAMQStreams() (*AMQStreams, error)
	ReadRHSSO() (*RHSSO, error)
	ReadRHSSOUser() (*RHSSOUser, error)
	ReadCodeReady() (*CodeReady, error)
	ReadThreeScale() (*ThreeScale, error)
	ReadMarin3r() (*Marin3r, error)
	ReadFuse() (*Fuse, error)
	ReadFuseOnOpenshift() (*FuseOnOpenshift, error)
	ReadAMQOnline() (*AMQOnline, error)
	GetOperatorNamespace() string
	ReadSolutionExplorer() (*SolutionExplorer, error)
	ReadMonitoring() (*Monitoring, error)
	ReadProduct(product integreatlyv1alpha1.ProductName) (ConfigReadable, error)
	ReadUps() (*Ups, error)
	ReadApicurioRegistry() (*ApicurioRegistry, error)
	ReadApicurito() (*Apicurito, error)
	ReadCloudResources() (*CloudResources, error)
	ReadDataSync() (*DataSync, error)
	ReadMonitoringSpec() (*MonitoringSpec, error)
	ReadGrafana() (*Grafana, error)
}

//go:generate moq -out ConfigReadable_moq.go . ConfigReadable
type ConfigReadable interface {
	//Read is used by the configManager to convert your config to yaml and store it in the configmap.
	Read() ProductConfig

	//GetProductName returns the value of the globally defined ProductName
	GetProductName() integreatlyv1alpha1.ProductName

	//GetProductVersion returns the value of the globally defined ProductVersion
	GetProductVersion() integreatlyv1alpha1.ProductVersion

	//GetOperatorVersion returns the value of the globally defined OperatorVersion
	GetOperatorVersion() integreatlyv1alpha1.OperatorVersion

	//GetHost returns a URL that can be used to access the product, either an API, or console, or blank if not applicable.
	GetHost() string

	//GetWatchableCRDs should return an array of CRDs that should be watched by the integreatly-operator, if a change of one of these CRDs
	//in any namespace is detected, it will trigger a full reconcile of the integreatly-operator. This usually returns all of
	//the CRDs the new products operator watches.
	GetWatchableCRDs() []runtime.Object

	//GetNamespace should return the namespace that the product will be installed into.
	GetNamespace() string
}

type Manager struct {
	Client       k8sclient.Client
	Namespace    string
	cfgmap       *corev1.ConfigMap
	context      context.Context
	installation *integreatlyv1alpha1.RHMI
}

func (m *Manager) ReadProduct(product integreatlyv1alpha1.ProductName) (ConfigReadable, error) {
	switch product {
	case integreatlyv1alpha1.Product3Scale:
		return m.ReadThreeScale()
	case integreatlyv1alpha1.ProductAMQOnline:
		return m.ReadAMQOnline()
	case integreatlyv1alpha1.ProductRHSSO:
		return m.ReadRHSSO()
	case integreatlyv1alpha1.ProductRHSSOUser:
		return m.ReadRHSSOUser()
	case integreatlyv1alpha1.ProductAMQStreams:
		return m.ReadAMQStreams()
	case integreatlyv1alpha1.ProductCodeReadyWorkspaces:
		return m.ReadCodeReady()
	case integreatlyv1alpha1.ProductFuse:
		return m.ReadFuse()
	case integreatlyv1alpha1.ProductFuseOnOpenshift:
		return m.ReadFuseOnOpenshift()
	case integreatlyv1alpha1.ProductSolutionExplorer:
		return m.ReadSolutionExplorer()
	case integreatlyv1alpha1.ProductUps:
		return m.ReadUps()
	case integreatlyv1alpha1.ProductApicurioRegistry:
		return m.ReadApicurioRegistry()
	case integreatlyv1alpha1.ProductApicurito:
		return m.ReadApicurito()
	case integreatlyv1alpha1.ProductCloudResources:
		return m.ReadCloudResources()
	case integreatlyv1alpha1.ProductMonitoring:
		return m.ReadMonitoring()
	case integreatlyv1alpha1.ProductDataSync:
		return m.ReadDataSync()
	case integreatlyv1alpha1.ProductMonitoringSpec:
		return m.ReadMonitoringSpec()
	case integreatlyv1alpha1.ProductMarin3r:
		return m.ReadMarin3r()
	case integreatlyv1alpha1.ProductGrafana:
		return m.ReadGrafana()
	}

	return nil, fmt.Errorf("no config found for product %v", product)
}

func (m *Manager) ReadSolutionExplorer() (*SolutionExplorer, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductSolutionExplorer)
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

func (m *Manager) GetBackupsSecretName() string {
	return "backups-s3-credentials"
}

func (m *Manager) GetGHOauthClientsSecretName() string {
	return "github-oauth-secret"
}

func (m *Manager) ReadAMQStreams() (*AMQStreams, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductAMQStreams)
	if err != nil {
		return nil, err
	}
	return NewAMQStreams(config), nil
}

func (m *Manager) ReadThreeScale() (*ThreeScale, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.Product3Scale)
	if err != nil {
		return nil, err
	}
	return NewThreeScale(config), nil
}

func (m *Manager) ReadCodeReady() (*CodeReady, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductCodeReadyWorkspaces)
	if err != nil {
		return nil, err
	}
	return NewCodeReady(config), nil
}

func (m *Manager) ReadFuse() (*Fuse, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductFuse)
	if err != nil {
		return nil, err
	}
	return NewFuse(config), nil
}

func (m *Manager) ReadFuseOnOpenshift() (*FuseOnOpenshift, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductFuseOnOpenshift)
	if err != nil {
		return nil, err
	}
	return NewFuseOnOpenshift(config), nil
}

func (m *Manager) ReadRHSSO() (*RHSSO, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductRHSSO)
	if err != nil {
		return nil, err
	}
	return NewRHSSO(config), nil
}

func (m *Manager) ReadRHSSOUser() (*RHSSOUser, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductRHSSOUser)
	if err != nil {
		return nil, err
	}
	return NewRHSSOUser(config), nil
}

func (m *Manager) ReadAMQOnline() (*AMQOnline, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductAMQOnline)
	if err != nil {
		return nil, err
	}
	return NewAMQOnline(config), nil
}

func (m *Manager) ReadMonitoring() (*Monitoring, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductMonitoring)
	if err != nil {
		return nil, err
	}
	return NewMonitoring(config), nil
}

func (m *Manager) ReadMonitoringSpec() (*MonitoringSpec, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductMonitoringSpec)
	if err != nil {
		return nil, err
	}
	return NewMonitoringSpec(config), nil
}

func (m *Manager) ReadApicurioRegistry() (*ApicurioRegistry, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductApicurioRegistry)
	if err != nil {
		return nil, err
	}
	return NewApicurioRegistry(config), nil
}

func (m *Manager) ReadApicurito() (*Apicurito, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductApicurito)
	if err != nil {
		return nil, err
	}

	return NewApicurito(config), nil
}

func (m *Manager) ReadGrafana() (*Grafana, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductGrafana)
	if err != nil {
		return nil, err
	}

	return NewGrafana(config), nil
}

func (m *Manager) ReadUps() (*Ups, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductUps)
	if err != nil {
		return nil, err
	}

	return NewUps(config), nil
}

func (m *Manager) ReadCloudResources() (*CloudResources, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductCloudResources)
	if err != nil {
		return nil, err
	}
	return NewCloudResources(config), nil
}

func (m *Manager) ReadDataSync() (*DataSync, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductDataSync)
	if err != nil {
		return nil, err
	}
	return NewDataSync(config), nil
}

func (m *Manager) ReadMarin3r() (*Marin3r, error) {
	config, err := m.readConfigForProduct(integreatlyv1alpha1.ProductMarin3r)
	if err != nil {
		return nil, err
	}
	return NewMarin3r(config), nil
}

func (m *Manager) WriteConfig(config ConfigReadable) error {
	stringConfig, err := yaml.Marshal(config.Read())
	err = m.Client.Get(m.context, k8sclient.ObjectKey{Name: m.cfgmap.Name, Namespace: m.Namespace}, m.cfgmap)
	if errors.IsNotFound(err) {
		m.cfgmap.Data = map[string]string{string(config.GetProductName()): string(stringConfig)}
		return m.Client.Create(m.context, m.cfgmap)
	}
	if m.cfgmap.Data == nil {
		m.cfgmap.Data = map[string]string{}
	}
	m.cfgmap.Data[string(config.GetProductName())] = string(stringConfig)
	return m.Client.Update(m.context, m.cfgmap)
}

func (m *Manager) readConfigForProduct(product integreatlyv1alpha1.ProductName) (ProductConfig, error) {
	config := m.cfgmap.Data[string(product)]
	decoder := yaml.NewDecoder(strings.NewReader(config))
	retConfig := ProductConfig{}
	if config == "" {
		return retConfig, nil
	}
	if err := decoder.Decode(retConfig); err != nil {
		return nil, fmt.Errorf("failed to decode product config for %v: %w", product, err)
	}
	return retConfig, nil
}
