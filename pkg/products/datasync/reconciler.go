package datasync

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	datasyncNs               = "openshift"
	templatesBaseURL         = "https://raw.githubusercontent.com/aerogear/datasync-deployment/"
	openshiftTemplatesFolder = "/openshift/"
)

var (
	datasyncTemplates = []string{
		"datasync-http.yml",
		"datasync-showcase.yml",
	}
)

type Reconciler struct {
	*resources.Reconciler
	coreClient    kubernetes.Interface
	Config        *config.DataSync
	ConfigManager config.ConfigReadWriter
	httpClient    http.Client
	logger        *logrus.Entry
	recorder      record.EventRecorder
	installation  *integreatlyv1alpha1.RHMI
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder) (*Reconciler, error) {
	config, err := configManager.ReadDataSync()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve %s config: %w", integreatlyv1alpha1.ProductDataSync, err)
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(datasyncNs)
	}

	if err = config.Validate(); err != nil {
		return nil, fmt.Errorf("%s config is not valid: %w", integreatlyv1alpha1.ProductDataSync, err)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())
	var httpClient http.Client

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		logger:        logger,
		httpClient:    httpClient,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
		installation:  installation,
	}, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.reconcileTemplates(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile configmap", err)
		return phase, err
	}

	product.Version = r.Config.GetProductVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileTemplates(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	for _, templateFn := range datasyncTemplates {
		fileUrl := templatesBaseURL + string(r.Config.GetProductVersion()) + openshiftTemplatesFolder + templateFn

		fileData, err := r.getFileContentFromURL(fileUrl)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get file contents of %s: %w", templateFn, err)
		}

		content, err := ioutil.ReadAll(fileData)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to read contents of %s: %w", templateFn, err)
		}

		if filepath.Ext(templateFn) == ".yml" || filepath.Ext(templateFn) == ".yaml" {
			content, err = yaml.ToJSON(content)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to convert yaml to json %s: %w", templateFn, err)
			}
		}

		templateRuntimeObj, err := resources.LoadKubernetesResource(content, r.Config.GetNamespace())
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to load resource %s: %w", templateFn, err)
		}

		templateUnstructured, err := resources.UnstructuredFromRuntimeObject(templateRuntimeObj)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to parse object: %w", err)
		}

		if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, templateUnstructured, func() error {
			ownerutil.EnsureOwner(templateUnstructured, r.installation)
			return nil
		}); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("Error reconciling datasync template %s: %w", templateUnstructured.GetName(), err)
		}
	}
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getFileContentFromURL(url string) (io.ReadCloser, error) {
	resp, err := r.httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return resp.Body, nil
	}
	return nil, fmt.Errorf("failed to get file content from %s. Status: %d", url, resp.StatusCode)
}
