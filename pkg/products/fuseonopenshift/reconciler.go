package fuseonopenshift

import (
	"context"
	"encoding/json"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/integr8ly/integreatly-operator/pkg/resources/events"
	"github.com/integr8ly/integreatly-operator/pkg/resources/marketplace"
	"github.com/integr8ly/integreatly-operator/version"

	samplesv1 "github.com/openshift/cluster-samples-operator/pkg/apis/samples/v1"

	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/record"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	fuseOnOpenshiftNs      = "openshift"
	TemplatesBaseURL       = "https://raw.githubusercontent.com/jboss-fuse/application-templates/"
	templatesConfigMapName = "fuse-on-openshift-templates"
	imageStreamFileName    = "fis-image-streams.json"
)

var (
	// See installation reference below for current install version 7.6
	// https://access.redhat.com/documentation/en-us/red_hat_fuse/7.6/html-single/fuse_on_openshift_guide/index#install-fuse-on-openshift4

	corePrefix         = integreatlyv1alpha1.TagFuseOnOpenShiftCore + "/"
	springBoot2Prefix  = integreatlyv1alpha1.TagFuseOnOpenShiftSpringBoot2 + "/"
	quickStartLocation = "quickstarts/"

	quickstartCoreTemplates = []string{
		"eap-camel-amq-template.json",
		"eap-camel-cdi-template.json",
		"eap-camel-cxf-jaxrs-template.json",
		"eap-camel-cxf-jaxws-template.json",
		"eap-camel-jpa-template.json",
		"karaf-camel-amq-template.json",
		"karaf-camel-log-template.json",
		"karaf-camel-rest-sql-template.json",
		"karaf-cxf-rest-template.json",
		"spring-boot-camel-amq-template.json",
		"spring-boot-camel-config-template.json",
		"spring-boot-camel-drools-template.json",
		"spring-boot-camel-infinispan-template.json",
		"spring-boot-camel-rest-3scale-template.json",
		"spring-boot-camel-rest-sql-template.json",
		"spring-boot-camel-template.json",
		"spring-boot-camel-xa-template.json",
		"spring-boot-camel-xml-template.json",
		"spring-boot-cxf-jaxrs-template.json",
		"spring-boot-cxf-jaxws-template.json",
	}
	quickstartSpringBoot2Templates = []string{
		"spring-boot-2-camel-amq-template.json",
		"spring-boot-2-camel-config-template.json",
		"spring-boot-2-camel-drools-template.json",
		"spring-boot-2-camel-infinispan-template.json",
		"spring-boot-2-camel-rest-3scale-template.json",
		"spring-boot-2-camel-rest-sql-template.json",
		"spring-boot-2-camel-template.json",
		"spring-boot-2-camel-xa-template.json",
		"spring-boot-2-camel-xml-template.json",
		"spring-boot-2-cxf-jaxrs-template.json",
		"spring-boot-2-cxf-jaxrs-xml-template.json",
		"spring-boot-2-cxf-jaxws-template.json",
		"spring-boot-2-cxf-jaxws-xml-template.json",
	}
	consoleTemplates = []string{
		"fuse-console-namespace-os4.json",
		"fis-console-namespace-template.json",
		"fuse-console-cluster-os4.json",
		"fis-console-cluster-template.json",
		"fuse-apicurito.yml",
	}
)

type Reconciler struct {
	*resources.Reconciler
	Config        *config.FuseOnOpenshift
	ConfigManager config.ConfigReadWriter
	httpClient    *http.Client
	log           l.Logger
	recorder      record.EventRecorder
	installation  *integreatlyv1alpha1.RHMI
	baseURL       string
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return nil
}

func NewReconciler(configManager config.ConfigReadWriter, installation *integreatlyv1alpha1.RHMI, mpm marketplace.MarketplaceInterface, recorder record.EventRecorder, httpClient *http.Client, baseURL string, logger l.Logger) (*Reconciler, error) {
	config, err := configManager.ReadFuseOnOpenshift()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve %s config: %w", integreatlyv1alpha1.ProductFuseOnOpenshift, err)
	}

	if config.GetNamespace() == "" {
		config.SetNamespace(fuseOnOpenshiftNs)
	}

	if err = config.Validate(); err != nil {
		return nil, fmt.Errorf("%s config is not valid: %w", integreatlyv1alpha1.ProductFuseOnOpenshift, err)
	}

	httpClient.Timeout = time.Second * 20
	httpClient.Transport = &http.Transport{DisableKeepAlives: true, IdleConnTimeout: time.Second * 20}

	url := ""
	if baseURL == "" {
		url = TemplatesBaseURL
	} else {
		url = baseURL
	}
	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		log:           logger,
		httpClient:    httpClient,
		Reconciler:    resources.NewReconciler(mpm),
		recorder:      recorder,
		installation:  installation,
		baseURL:       url,
	}, nil
}

func (r *Reconciler) VerifyVersion(installation *integreatlyv1alpha1.RHMI) bool {
	return version.VerifyProductAndOperatorVersion(
		installation.Status.Stages[integreatlyv1alpha1.ProductsStage].Products[integreatlyv1alpha1.ProductFuseOnOpenshift],
		string(integreatlyv1alpha1.VersionFuseOnOpenshift),
		string(integreatlyv1alpha1.OperatorVersionFuse),
	)
}

func (r *Reconciler) Reconcile(ctx context.Context, installation *integreatlyv1alpha1.RHMI, product *integreatlyv1alpha1.RHMIProductStatus, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	phase, err := r.reconcileConfigMap(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile configmap", err)
		return phase, err
	}

	phase, err = r.reconcileImageStreams(ctx, serverClient)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile image streams", err)
		return phase, err
	}

	phase, err = r.reconcileTemplates(ctx, serverClient, installation)
	if err != nil || phase != integreatlyv1alpha1.PhaseCompleted {
		events.HandleError(r.recorder, installation, phase, "Failed to reconcile templates", err)
		return phase, err
	}

	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	events.HandleProductComplete(r.recorder, installation, integreatlyv1alpha1.ProductsStage, r.Config.GetProductName())
	r.log.Infof("Successfully reconciled", l.Fields{"product": integreatlyv1alpha1.ProductFuseOnOpenshift})
	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling Fuse on OpenShift templates config map")
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      templatesConfigMapName,
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, cfgMap, func() error {

		cfgMap.Name = templatesConfigMapName
		cfgMap.Namespace = r.ConfigManager.GetOperatorNamespace()

		configMapData := make(map[string]string)
		fileNames := []string{
			corePrefix + imageStreamFileName,
		}

		for _, qn := range consoleTemplates {
			fileNames = append(fileNames, corePrefix+qn)
		}

		for _, qn := range quickstartCoreTemplates {
			fileNames = append(fileNames, corePrefix+quickStartLocation+qn)
		}

		for _, qn := range quickstartSpringBoot2Templates {
			fileNames = append(fileNames, springBoot2Prefix+quickStartLocation+qn)
		}

		for _, fn := range fileNames {

			fileURL := r.baseURL + fn

			content, err := r.getFileContentFromURL(fileURL)
			if err != nil {
				return fmt.Errorf("failed to get file contents of %s: %w", fn, err)
			}
			defer content.Close()

			data, err := ioutil.ReadAll(content)
			if err != nil {
				return fmt.Errorf("failed to read contents of %s: %w", fn, err)
			}

			// Remove the possible prefixes from the key as this is not a valid configmap data key
			key := strings.TrimPrefix(fn, corePrefix)
			key = strings.TrimPrefix(key, springBoot2Prefix)
			key = strings.TrimPrefix(key, quickStartLocation)

			// Write content of file to configmap
			configMapData[key] = string(data)
		}

		cfgMap.Data = configMapData

		return nil
	})

	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, err
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileImageStreams(ctx context.Context, serverClient k8sclient.Client) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling Fuse on OpenShift imagestreams")
	cfgMap, err := r.getTemplatesConfigMap(ctx, serverClient)
	if err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get configmap %s from %s namespace: %w", cfgMap.Name, cfgMap.Data, err)
	}

	content := []byte(cfgMap.Data[imageStreamFileName])

	var fileContent map[string]interface{}
	if err := json.Unmarshal(content, &fileContent); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to unmarshal contents of %s: %w", imageStreamFileName, err)
	}

	// The content of the imagestream file is an object of kind List
	// Create the imagestreams seperately
	isList := r.getResourcesFromList(fileContent)
	imageStreams := make(map[string]runtime.Object)
	for _, is := range isList {
		jsonData, err := json.Marshal(is)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to marshal data %s: %w", imageStreamFileName, err)
		}

		imageStreamRuntimeObj, err := resources.LoadKubernetesResource(jsonData, r.Config.GetNamespace())
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to load kubernetes imagestream resource: %w", err)
		}

		// Get unstructured of image stream so we can retrieve the image stream name
		imageStreamUnstructured, err := resources.UnstructuredFromRuntimeObject(imageStreamRuntimeObj)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to parse runtime object to unstructured for imagestream: %w", err)
		}

		imageStreamName := imageStreamUnstructured.GetName()
		imageStreams[imageStreamName] = imageStreamRuntimeObj
	}

	imageStreamNames := r.getKeysFromMap(imageStreams)

	// Update the sample cluster sample operator CR to skip the Fuse on OpenShift image streams
	if err := r.updateClusterSampleCR(ctx, serverClient, "SkippedImagestreams", imageStreamNames); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to update SkippedImagestreams in cluster sample custom resource: %w", err)
	}

	for isName, isObj := range imageStreams {
		if err := r.createResourceIfNotExist(ctx, serverClient, isObj); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create image stream %s: %w", isName, err)
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileTemplates(ctx context.Context, serverClient k8sclient.Client, installation *integreatlyv1alpha1.RHMI) (integreatlyv1alpha1.StatusPhase, error) {
	r.log.Info("Reconciling Fuse on OpenShift templates")
	var templateFiles []string
	templates := make(map[string]runtime.Object)

	templateFiles = append(templateFiles, consoleTemplates...)
	templateFiles = append(templateFiles, quickstartCoreTemplates...)
	templateFiles = append(templateFiles, quickstartSpringBoot2Templates...)

	for _, fileName := range templateFiles {
		cfgMap, err := r.getTemplatesConfigMap(ctx, serverClient)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to get configmap %s from %s namespace: %w", cfgMap.Name, cfgMap.Data, err)
		}

		content := []byte(cfgMap.Data[fileName])

		if filepath.Ext(fileName) == ".yml" || filepath.Ext(fileName) == ".yaml" {
			content, err = yaml.ToJSON(content)
			if err != nil {
				return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to convert yaml to json %s: %w", fileName, err)
			}
		}

		templateRuntimeObj, err := resources.LoadKubernetesResource(content, r.Config.GetNamespace())
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to load resource %s: %w", fileName, err)
		}

		templateUnstructured, err := resources.UnstructuredFromRuntimeObject(templateRuntimeObj)
		if err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to parse object: %w", err)
		}

		templateName := templateUnstructured.GetName()
		templates[templateName] = templateRuntimeObj
	}

	templateNames := r.getKeysFromMap(templates)

	// Update sample cluster operator CR to skip Fuse on OpenShift quickstart templates
	if err := r.updateClusterSampleCR(ctx, serverClient, "SkippedTemplates", templateNames); err != nil {
		return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to update SkippedTemplates in cluster sample custom resource: %w", err)
	}

	for name, obj := range templates {
		if err := r.createResourceIfNotExist(ctx, serverClient, obj); err != nil {
			return integreatlyv1alpha1.PhaseFailed, fmt.Errorf("failed to create image stream %s: %w", name, err)
		}
	}

	return integreatlyv1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) getTemplatesConfigMap(ctx context.Context, serverClient k8sclient.Client) (*corev1.ConfigMap, error) {
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      templatesConfigMapName,
			Namespace: r.ConfigManager.GetOperatorNamespace(),
		},
	}

	err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: cfgMap.Name, Namespace: cfgMap.Namespace}, cfgMap)
	return cfgMap, err
}

func (r *Reconciler) createResourceIfNotExist(ctx context.Context, serverClient k8sclient.Client, resource runtime.Object) error {
	u, err := resources.UnstructuredFromRuntimeObject(resource)
	if err != nil {
		return fmt.Errorf("failed to get unstructured object of type %T from resource %s", resource, resource)
	}

	if err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, u); err != nil {
		if !k8errors.IsNotFound(err) {
			return fmt.Errorf("failed to get resource: %w", err)
		}
		if err := serverClient.Create(ctx, resource); err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}
		return nil
	}

	if !r.resourceHasLabel(u.GetLabels(), "integreatly", "true") {
		if err := serverClient.Delete(ctx, resource); err != nil {
			return fmt.Errorf("failed to delete resource: %w", err)
		}
		if err := serverClient.Create(ctx, resource); err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}
	}

	return nil
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

func (r *Reconciler) getResourcesFromList(listObj map[string]interface{}) []interface{} {
	items := reflect.ValueOf(listObj["items"])

	var resources []interface{}

	for i := 0; i < items.Len(); i++ {
		resources = append(resources, items.Index(i).Interface())
	}

	return resources
}

func (r *Reconciler) updateClusterSampleCR(ctx context.Context, serverClient k8sclient.Client, key string, value []string) error {
	clusterSampleCR := &samplesv1.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, serverClient, clusterSampleCR, func() error {
		clusterSampleCR.Name = "cluster"

		if key == "SkippedImagestreams" {
			for _, v := range value {
				if !r.contains(clusterSampleCR.Spec.SkippedImagestreams, v) {
					clusterSampleCR.Spec.SkippedImagestreams = append(clusterSampleCR.Spec.SkippedImagestreams, v)
				}
			}
		}

		if key == "SkippedTemplates" {
			for _, v := range value {
				if !r.contains(clusterSampleCR.Spec.SkippedTemplates, v) {
					clusterSampleCR.Spec.SkippedTemplates = append(clusterSampleCR.Spec.SkippedTemplates, v)
				}
			}
		}

		return nil
	})

	if err != nil {
		r.log.Error("Error updating cluster Sample CR", err)
	}

	return nil
}

func (r *Reconciler) getKeysFromMap(mapObj map[string]runtime.Object) []string {
	var keys []string

	for k := range mapObj {
		keys = append(keys, k)
	}
	return keys
}

func (r *Reconciler) resourceHasLabel(labels map[string]string, key, value string) bool {
	if val, ok := labels[key]; ok && val == value {
		return true
	}
	return false
}

func (r *Reconciler) contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}

	return false
}
