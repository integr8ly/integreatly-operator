package marin3r

import (
	"context"
	"fmt"
	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type csvUpdater struct {
	ctx       context.Context
	client    k8sclient.Client
	namespace string
	log       l.Logger
	csv       *operatorsv1alpha1.ClusterServiceVersion
}

func newCsvUpdater(ctx context.Context, client k8sclient.Client, namespace string, log l.Logger) csvUpdater {
	return csvUpdater{ctx: ctx, client: client, namespace: namespace, log: log}
}

func (f *csvUpdater) findCsv() error {
	csvPrefix := "marin3r"
	csvList := &operatorsv1alpha1.ClusterServiceVersionList{}
	listOptions := &k8sclient.ListOptions{
		Namespace: f.namespace,
	}

	f.log.Infof("finding marin3r CSV", l.Fields{"namespace": f.namespace, "csv prefix": csvPrefix})
	err := f.client.List(f.ctx, csvList, listOptions)
	if err != nil {
		f.log.Errorf("failed to list CSVs", l.Fields{"namespace": f.namespace}, err)
		return err
	}

	for i, csv := range csvList.Items {
		if strings.HasPrefix(csv.Name, csvPrefix) {
			f.csv = &csvList.Items[i]
			return nil
		}
	}

	err = fmt.Errorf("failed to find marin3r CSV")
	f.log.Errorf("failed to find marin3r CSV", l.Fields{"namespace": f.namespace, "csv prefix": csvPrefix}, err)
	return err
}

func (f *csvUpdater) setManagerResources() error {
	if f.csv == nil {
		err := fmt.Errorf("csvUpdater.csv is not set, run csvUpdater.findCsv()")
		f.log.Errorf("failed setting manager resources", l.Fields{"namespace": f.namespace}, err)
		return err
	}

	f.log.Infof("setting manager resources", l.Fields{"csv": f.csv.Name, "namespace": f.namespace})
	memoryLimit := "800Mi"
	deploymentName := "marin3r-controller-manager"
	containerName := "manager"

	for i, spec := range f.csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		if spec.Name == deploymentName {
			for j, container := range spec.Spec.Template.Spec.Containers {
				if container.Name == containerName {
					cpu := container.Resources.Limits.Cpu().DeepCopy()
					limits := corev1.ResourceList{corev1.ResourceCPU: cpu, corev1.ResourceMemory: resource.MustParse(memoryLimit)}
					f.csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[i].Spec.Template.Spec.Containers[j].Resources.Limits = limits
					return nil
				}
			}
		}
	}
	err := fmt.Errorf("unable to find manager container to update")
	f.log.Errorf("failed setting manager resources", l.Fields{"csv": f.csv.Name, "namespace": f.namespace}, err)
	return err
}

func (f *csvUpdater) updateCSV() error {
	if f.csv == nil {
		err := fmt.Errorf("csvUpdater.csv is not set, run csvUpdater.findCsv()")
		f.log.Errorf("failed updating csv", l.Fields{"namespace": f.namespace}, err)
		return err
	}

	f.log.Infof("updating csv", l.Fields{"csv": f.csv.Name, "namespace": f.namespace})
	err := f.client.Update(f.ctx, f.csv)
	if err != nil {
		f.log.Errorf("failed updating csv", l.Fields{"csv": f.csv.Name, "namespace": f.namespace}, err)
		return err
	}
	return nil
}
