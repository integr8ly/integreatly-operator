package monitoringcommon

import (
	"context"
	"fmt"
	v1alpha12 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources/owner"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultBlackboxModule = "http_2xx"
)

func CreateBlackboxTarget(ctx context.Context, name string, target v1alpha12.BlackboxtargetData, cfg *config.Monitoring, installation *v1alpha12.RHMI, serverClient client.Client) error {
	if cfg.GetOperatorNamespace() == "" {
		// Retry later
		return nil
	}

	if target.Url == "" {
		// Retry later if the URL is not yet known
		return nil
	}

	// default policy is to require a 2xx http return code
	module := target.Module
	if module == "" {
		module = defaultBlackboxModule
	}

	// prepare the template
	extraParams := map[string]string{
		"Namespace":     cfg.GetOperatorNamespace(),
		"MonitoringKey": cfg.GetLabelSelector(),
		"name":          name,
		"url":           target.Url,
		"service":       target.Service,
		"module":        module,
	}

	templateHelper := NewTemplateHelper(extraParams)
	obj, err := templateHelper.CreateResource("blackbox/target.yaml")
	if err != nil {
		return fmt.Errorf("error creating resource from template: %w", err)
	}

	metaObj, err := meta.Accessor(obj)
	if err == nil {
		owner.AddIntegreatlyOwnerAnnotations(metaObj, installation)
	}
	// try to create the blackbox target. If if fails with already exist do nothing
	err = serverClient.Create(ctx, obj)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// The target already exists. Nothing else to do
			return nil
		}
		return fmt.Errorf("error creating blackbox target: %w", err)
	}

	return nil
}
