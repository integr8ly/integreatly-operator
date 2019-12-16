package detector

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"time"
)

// Detector represents a procedure that runs in the background, periodically auto-detecting features
type Detector struct {
	dc     discovery.DiscoveryInterface
	ticker *time.Ticker
	crds   map[runtime.Object]trigger
}

type trigger func(runtime.Object)

// New creates a new auto-detect runner
func NewAutoDetect(dc discovery.DiscoveryInterface) (*Detector, error) {
	return &Detector{dc: dc, crds: map[runtime.Object]trigger{}}, nil
}

//AddCRDTrigger to run the trigger function,
//the first time that the background scanner discovers that the CRD type specified exists
func (d *Detector) AddCRDTrigger(crd runtime.Object, trigger trigger) {
	d.crds[crd] = trigger
}

//AddCRDsTrigger to run the trigger function,
//the first time that the background scanner discovers that each of the CRD types specified exists
func (d *Detector) AddCRDsTrigger(crds []runtime.Object, trigger trigger) {
	for _, crd := range crds {
		d.AddCRDTrigger(crd, trigger)
	}
}

//AddCRDsWithTriggers to run the associated trigger function for the particular CRD,
//the first time that the background scanner discovers that the CRD type specified exists
func (d *Detector) AddCRDsWithTriggers(crdsTriggers map[runtime.Object]trigger) {
	for crd, trigger := range crdsTriggers {
		d.AddCRDTrigger(crd, trigger)
	}
}

// Start initializes the auto-detection process that runs in the background
func (d *Detector) Start(interval time.Duration) {
	go func() {
		d.autoDetectCapabilities()
		d.ticker = time.NewTicker(interval)
		for range d.ticker.C {
			d.autoDetectCapabilities()
		}
	}()
}

// Stop causes the background process to stop auto detecting capabilities
func (d *Detector) Stop() {
	d.ticker.Stop()
}

func (d *Detector) autoDetectCapabilities() {
	apiLists, err := d.dc.ServerResources()
	if err != nil {
		return
	}
	for crd, trigger := range d.crds {
		crdGVK := crd.GetObjectKind().GroupVersionKind()
		resourceExists, _ := d.resourceExists(apiLists, crdGVK.GroupVersion().String(), crdGVK.Kind)
		if resourceExists {
			stateManager := GetStateManager()
			if stateManager.GetState(crdGVK.Kind) != true {
				stateManager.SetState(crdGVK.Kind, true)
				trigger(crd)
			}
		}
	}
}

func (d *Detector) resourceExists(apiLists []*metav1.APIResourceList, apiGroupVersion, kind string) (bool, error) {
	for _, apiList := range apiLists {
		if apiList.GroupVersion == apiGroupVersion {
			for _, r := range apiList.APIResources {
				if r.Kind == kind {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
