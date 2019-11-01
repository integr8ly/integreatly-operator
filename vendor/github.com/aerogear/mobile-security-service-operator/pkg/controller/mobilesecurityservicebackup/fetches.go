package mobilesecurityservicebackup

import (
	"context"
	"fmt"
	mobilesecurityservicev1alpha1 "github.com/aerogear/mobile-security-service-operator/pkg/apis/mobilesecurityservice/v1alpha1"
	"github.com/aerogear/mobile-security-service-operator/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Request object not found, could have been deleted after reconcile request.
// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
func (r *ReconcileMobileSecurityServiceBackup) fetchBkpInstance(reqLogger logr.Logger, request reconcile.Request) (*mobilesecurityservicev1alpha1.MobileSecurityServiceBackup, error) {
	reqLogger.Info("Checking if the ReconcileMobileSecurityServiceBackup already exists")
	bkp := &mobilesecurityservicev1alpha1.MobileSecurityServiceBackup{}
	//Fetch the MobileSecurityServiceDB db
	err := r.client.Get(context.TODO(), request.NamespacedName, bkp)
	return bkp, err
}

// fetchCronJob return the cronJob created pod created by eMobileSecurityServiceDBBackup
func (r *ReconcileMobileSecurityServiceBackup) fetchCronJob(reqLogger logr.Logger, bkp *mobilesecurityservicev1alpha1.MobileSecurityServiceBackup) (*v1beta1.CronJob, error) {
	reqLogger.Info("Checking if the cronJob already exists")
	cronJob := &v1beta1.CronJob{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: bkp.Name, Namespace: bkp.Namespace}, cronJob)
	return cronJob, err
}

// fetchCronJob return the cronJob created pod created by eMobileSecurityServiceDBBackup
func (r *ReconcileMobileSecurityServiceBackup) fetchSecret(reqLogger logr.Logger, secretNamespace, secretName string) (*corev1.Secret, error) {
	reqLogger.Info("Checking if the secret already exists", "secret.name", secretName, "secret.Namespace", secretNamespace)
	secret := &corev1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: secretNamespace}, secret)
	return secret, err
}

func (r *ReconcileMobileSecurityServiceBackup) fetchConfigMap(bkp *mobilesecurityservicev1alpha1.MobileSecurityServiceBackup, cfgName string) (*corev1.ConfigMap, error) {
	log.Info("Looking for ConfigMap to get database data", "configMapName", cfgName)
	cfg := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cfgName, Namespace: bkp.Namespace}, cfg)
	return cfg, err
}

func (r *ReconcileMobileSecurityServiceBackup) fetchBDPod(reqLogger logr.Logger, request reconcile.Request) (*corev1.Pod, error) {
	listOps, err := r.getListOpsToSearchDBObject(reqLogger)
	if err != nil {
		return nil, err
	}

	// Search DB pods
	reqLogger.Info("Searching for DB pod ...")
	dbPodList := &corev1.PodList{}
	err = r.client.List(context.TODO(), listOps, dbPodList)
	if err != nil {
		return nil, err
	}

	if len(dbPodList.Items) == 0 {
		err = fmt.Errorf("Unable to find database pod. Maybe, it was not create yet")
		return nil, err
	}

	// Getting the pod ( it has just one )
	pod := dbPodList.Items[0]
	reqLogger.Info("DB Pod was found", "pod.Name", pod.Name)
	return &pod, nil
}

func (r *ReconcileMobileSecurityServiceBackup) fetchServiceDB(reqLogger logr.Logger, request reconcile.Request) (*corev1.Service, error) {
	listOps, err := r.getListOpsToSearchDBObject(reqLogger)
	if err != nil {
		return nil, err
	}

	// Search DB pods
	reqLogger.Info("Searching for Service pod ...")
	dbServiceList := &corev1.ServiceList{}
	err = r.client.List(context.TODO(), listOps, dbServiceList)
	if err != nil {
		return nil, err
	}

	if len(dbServiceList.Items) == 0 {
		err = fmt.Errorf("Unable to find database service. Maybe, it was not create yet")
		return nil, err
	}

	// Getting the pod ( it has just one )
	srv := dbServiceList.Items[0]
	reqLogger.Info("DB Service was found", "srv.Name", srv.Name)
	return &srv, nil
}

func (r *ReconcileMobileSecurityServiceBackup) getListOpsToSearchDBObject(reqLogger logr.Logger) (*client.ListOptions, error) {
	reqLogger.Info("Checking if the Database Service exists ...")
	reqLogger.Info("Checking operator namespace ...")
	operatorNamespace, err := k8sutil.GetOperatorNamespace()
	// Check if it is a local env or an unit test
	if err == k8sutil.ErrNoNamespace {
		operatorNamespace = utils.OperatorNamespaceForLocalEnv
	}
	// Fetch Mobile Security Service Database
	reqLogger.Info("Checking MobileSecurityServiceDB exists ...")
	db := &mobilesecurityservicev1alpha1.MobileSecurityServiceDB{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: utils.MobileSecurityServiceDBCRName, Namespace: operatorNamespace}, db)
	if err != nil {
		return nil, err
	}
	// Create criteria
	reqLogger.Info("Creating criteria to looking for Service ...")
	ls := map[string]string{"app": "mobilesecurityservice", "mobilesecurityservicedb_cr": db.Name, "name": "mobilesecurityservicedb"}
	labelSelector := labels.SelectorFromSet(ls)
	listOps := &client.ListOptions{Namespace: operatorNamespace, LabelSelector: labelSelector}
	return listOps, nil
}
