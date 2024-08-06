package common

import (
	goctx "context"
	"crypto/rand"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/test/resources"
	"golang.org/x/net/context"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"math/big"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "smtp-server"
	none          = "None"
	partial       = "Partial"
	full          = "Full"
)

var (
	r1, _                        = rand.Int(rand.Reader, big.NewInt(200))
	smtpReplicas           int32 = 1
	emailAddress                 = fmt.Sprintf("test%v@test.com", r1.Int64())
	serviceIP                    = ""
	emailUsername                = "dummy"
	emailPassword                = "dummy"
	emailPort                    = "1587"
	originalHost                 = ""
	originalPassword             = ""
	originalPort                 = ""
	originalUsername             = ""
	original3scalePassword       = ""
	original3scalePort           = ""
	original3scaleHost           = ""
	original3scaleUsername       = ""
)

// Test3ScaleSMTPConfig to confirm 3scale can send an email
func Test3ScaleSMTPConfig(t TestingTB, ctx *TestingContext) {
	inst, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get RHMI instance %v", err)
	}

	okToTest, err := customSmtpParameters(inst, none, ctx.Client)
	if err != nil {
		t.Error("test failure getting the custom SMTP state", err)
	}

	if !okToTest {
		t.Skip("Addon custom smtp values are configured. This test is not ok to run.")
	}

	defer restartThreeScalePods(t, ctx, inst)

	t.Log("Create Namespace, Deployment and Service for SMTP-Server")
	err = createNamespace(ctx, t)
	if err != nil {
		t.Logf("%v", err)
	}

	defer removeNamespaceLogErrors(t, ctx)

	_, isCreated, err := patchSecret(ctx, t)
	if err != nil {
		t.Log(err)
	}
	if rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(inst.Spec.Type)) {
		_, err = patch3ScaleSecret(ctx, t)
		if err != nil {
			t.Log(err)
		}
	}

	defer func(t TestingTB, ctx *TestingContext, isCreated bool) {
		t.Log("Reset SMTP details")
		_, err = resetSecret(ctx, t, isCreated)
		if err != nil {
			t.Log(err)
		}
		if rhmiv1alpha1.IsRHOAMMultitenant(rhmiv1alpha1.InstallationType(inst.Spec.Type)) {
			err = reset3ScaleSecret(ctx, t)
			if err != nil {
				t.Log(err)
			}
		}
	}(t, ctx, isCreated)

	t.Log("Wait for reconciliation of SMTP details")
	err = checkSMTPReconciliation(ctx, t)
	if err != nil {
		t.Fatalf("Unable to reconcile smtp details : %v ", err)
	}

	restartThreeScalePods(t, ctx, inst)

	t.Log("Send Test email")
	sendTestEmail(ctx, t)

	t.Log("confirm email received")
	err = checkEmail(ctx, t, emailAddress)
	if err != nil {
		t.Fatal("No email found")
	}
}

func Test3ScaleCustomSMTPFullConfig(t TestingTB, ctx *TestingContext) {
	inst, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get RHMI instance %v", err)
	}

	managed, err := addon.OperatorIsHiveManaged(context.TODO(), ctx.Client, inst)
	if err != nil {
		t.Errorf("error getting hive managed labels: %v", err)
	}

	if !managed {
		t.Log("Create Namespace, Deployment and Service for SMTP-Server")
		err = createNamespace(ctx, t)
		if err != nil {
			t.Logf("%v", err)
		}

		defer removeNamespaceLogErrors(t, ctx)

		serviceIP, err = getServiceIP(ctx)
		if err != nil {
			t.Errorf("setup error getting service address: %v", err)
		}
		secret := &v1.Secret{}
		data, err := copyAddonSecretData(secret, ctx, inst)
		if err != nil {
			t.Errorf(err.Error())
		}

		// add custom values
		_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, secret, func() error {

			secret.Data["custom-smtp-username"] = []byte(emailUsername)
			secret.Data["custom-smtp-port"] = []byte(emailPort)
			secret.Data["custom-smtp-password"] = []byte(emailPassword)
			secret.Data["custom-smtp-address"] = []byte(serviceIP)
			secret.Data["custom-smtp-from_address"] = []byte(emailAddress)

			return nil
		})
		if err != nil {
			return
		}

		defer resetSecretData(t, ctx, data, secret)

	}
	// check if addon secret has all the fields required
	okToTest, err := customSmtpParameters(inst, full, ctx.Client)
	if err != nil {
		t.Error("test failure getting the custom SMTP state", err)
	}

	if !okToTest {
		t.Skip("Addon custom smtp values are not fully configured. This test is not ok to run.")
	}

	err = wait.PollUntilContextTimeout(goctx.TODO(), retryInterval, timeout, false, func(ctx2 goctx.Context) (done bool, err error) {
		inst, err = GetRHMI(ctx.Client, true)
		if err != nil {
			t.Logf("failed to get RHOAM instance %v", err)
			return false, nil
		}

		if inst.Status.CustomSmtp != nil && inst.Status.CustomSmtp.Enabled {
			t.Log("CR conditions met")
			return true, nil
		}
		t.Log("CR conditions not met.")
		return false, nil

	})
	if err != nil {
		t.Errorf("CR conditions not meet", err)
	}

	if inst.Status.CustomSmtp.Error != "" {
		t.Fatal("Unexpected error in the custom smtp status block")
	}

	// test that the 3scale smtp secret has the same values as the custom-smtp secret
	err = compareSMTPSecrets(inst, ctx.Client, "custom-smtp")
	if err != nil {
		t.Errorf("unable to compare SMTP secrets, ", err)
	}

	if !managed {
		err = wait.PollUntilContextTimeout(goctx.TODO(), retryInterval, timeout, false, func(ctx2 goctx.Context) (done bool, err error) {
			pods, err := ctx.KubeClient.CoreV1().Pods("smtp-server").List(goctx.TODO(), metav1.ListOptions{})
			if err != nil || len(pods.Items) == 0 {
				t.Errorf("couldn't find pods: %v", err)
			}
			for _, pod := range pods.Items {
				if pod.Status.Phase == "Running" {
					t.Log("Found running pods")
					return true, nil
				}
			}
			return false, nil
		})
		if err != nil {
			t.Errorf("failed to find running pods", err)
		}
	}
	restartThreeScalePods(t, ctx, inst)

	t.Log("Send Test email")
	sendTestEmail(ctx, t)

	if managed {
		t.Log("Manual checking of received emails required")
	} else {
		t.Log("confirm email received")
		err = checkEmail(ctx, t, emailAddress)
		if err != nil {
			t.Fatalf("No email found: %v", err)
		}
	}

}

func Test3ScaleCustomSMTPPartialConfig(t TestingTB, ctx *TestingContext) {
	inst, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get RHMI instance %v", err)
	}

	managed, err := addon.OperatorIsHiveManaged(context.TODO(), ctx.Client, inst)
	if err != nil {
		t.Errorf("error getting hive managed labels: %v", err)
	}

	if !managed {
		// copy the current custom smtp values
		secret := &v1.Secret{}
		data, err := copyAddonSecretData(secret, ctx, inst)
		if err != nil {
			t.Errorf(err.Error())
		}

		// add my own partial values
		_, err = controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, secret, func() error {

			secret.Data["custom-smtp-username"] = []byte("Partial")
			secret.Data["custom-smtp-port"] = []byte("")
			secret.Data["custom-smtp-password"] = []byte("Values")
			secret.Data["custom-smtp-address"] = []byte("")
			secret.Data["custom-smtp-from_address"] = []byte("")

			return nil
		})
		if err != nil {
			return
		}

		defer resetSecretData(t, ctx, data, secret)
	}

	okToTest, err := customSmtpParameters(inst, partial, ctx.Client)
	if err != nil {
		t.Error("test failure getting the custom SMTP state", err)
	}

	if !okToTest {
		t.Skip("Addon custom smtp values are not partial configured. This test is not ok to run.")
	}

	err = wait.PollUntilContextTimeout(goctx.TODO(), retryInterval, timeout, false, func(ctx2 goctx.Context) (done bool, err error) {
		inst, err = GetRHMI(ctx.Client, true)
		if err != nil {
			t.Logf("failed to get RHOAM instance %v", err)
			return false, nil
		}

		if inst.Status.CustomSmtp != nil && !inst.Status.CustomSmtp.Enabled {
			t.Log("CR conditions met")
			return true, nil
		}
		t.Log("CR conditions not met.")
		return false, nil

	})

	if err != nil {
		t.Fatalf("RHOAM status block not configure correctly, %v", err)
	}

	if inst.Status.CustomSmtp.Error == "" {
		t.Fatal("No error found in the custom " +
			"smtp status block")
	}

	err = compareSMTPSecrets(inst, ctx.Client, "redhat-rhoam-smtp")
	if err != nil {
		t.Errorf("unable to compare SMTP secrets, ", err)
	}

	customSmtp := &v1.Secret{}
	err = ctx.Client.Get(context.TODO(), k8sclient.ObjectKey{
		Name:      "custom-smtp",
		Namespace: inst.Namespace,
	}, customSmtp)
	// On partial configured instances there should be no custom smtp secret
	if err != nil && !k8errors.IsNotFound(err) {
		t.Fatalf("Failed trying getting secret, %v", err)
	}
	if err == nil {
		t.Fatal("No error received when getting missing secret")
	}
}

func restartThreeScalePods(t TestingTB, ctx *TestingContext, inst *rhmiv1alpha1.RHMI) {
	// Scale down system-app and system-sidekiq in order to load new smtp config
	t.Log("Redeploy 3Scale pods")
	for _, deployment := range []string{"system-app", "system-sidekiq"} {
		t.Logf("Scaling down deployment '%s' to 0 replicas in '%s' namespace", deployment, threescaleNamespace)
		scale3scaleDeployment(t, deployment, threescaleNamespace, 0, ctx.Client)
	}

	t.Log("Checking pods are ready")
	threeScaleConfig := config.NewThreeScale(map[string]string{})
	replicas := threeScaleConfig.GetReplicasConfig(inst)
	err := check3ScaleReplicasAreReady(ctx, t, replicas, retryInterval, timeout)
	if err != nil {
		t.Logf("Replicas not Ready within timeout: %v", err)
	}

	// Add sleep to give threescale time to reconcile the pods restarts otherwise host address will update during next steps
	time.Sleep(30 * time.Second)
	t.Log("Checking host address is ready")
	err = checkHostAddressIsReady(ctx, t, retryInterval, timeout)
	if err != nil {
		t.Log(err)
	}
}

func checkHostAddressIsReady(ctx *TestingContext, t TestingTB, retryInterval, timeout time.Duration) error {
	err := wait.PollUntilContextTimeout(goctx.TODO(), retryInterval, timeout, false, func(ctx2 goctx.Context) (done bool, err error) {

		// get console master url
		rhmi, err := GetRHMI(ctx.Client, true)
		if err != nil {
			t.Fatalf("error getting RHMI CR: %v", err)
		}

		host := rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.Product3Scale].Host
		status := rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.Product3Scale].Phase
		if host == "" || status == "in progress" {
			t.Log("3scale host URL not ready yet.")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		//return fmt.Error("Number of replicas for threescale replicas is not correct : Replicas - %w, Expected")
		return fmt.Errorf("error, Host url not ready before timeout - %v", err)
	}
	return nil

}

func removeNamespace(t TestingTB, ctx *TestingContext) error {
	//Remove the smtp-server namespace to clean up after test

	t.Log("Removing smtp-server namespace")
	nameSpaceForDeletion := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "smtp-server",
			Namespace: "smtp-server",
		},
	}

	err := ctx.Client.Delete(goctx.TODO(), nameSpaceForDeletion)
	if err != nil {
		return err
	}
	return nil
}

func checkSMTPReconciliation(ctx *TestingContext, t TestingTB) error {
	//Check that the smtp details have reconciled to the 3scale secret
	return wait.PollUntilContextTimeout(goctx.TODO(), retryInterval, timeout, false, func(ctx2 goctx.Context) (done bool, err error) {
		threescaleSecret, err := get3scaleSecret(ctx)
		if err != nil {
			t.Fatalf("Failed to get threescale secret %v", err)
		}

		rhmiSecret, err := getSecret(ctx)
		if err != nil {
			t.Fatalf("Failed to get rhmi secret %v", err)
		}
		if string(threescaleSecret.Data["address"]) != string(rhmiSecret.Data["host"]) {
			return false, nil
		}
		if string(threescaleSecret.Data["password"]) != string(rhmiSecret.Data["password"]) {
			return false, nil
		}
		if string(threescaleSecret.Data["port"]) != string(rhmiSecret.Data["port"]) {
			return false, nil
		}
		return true, nil
	})
}

func checkEmail(ctx *TestingContext, t TestingTB, email string) error {
	//Check that we have received the test email
	receivedEmail := false
	pods, err := ctx.KubeClient.CoreV1().Pods("smtp-server").List(goctx.TODO(), metav1.ListOptions{})
	if err != nil || len(pods.Items) == 0 {
		return fmt.Errorf("couldn't find pods: %v", err)
	}
	for _, pod := range pods.Items {
		fmt.Println(pod.Name, pod.Status.PodIP)
		// exec into the smtp-server pod
		output, err := execToPod("cat /newuser/mail.log",
			pod.Name,
			"smtp-server",
			"smtp-server", ctx)
		if err != nil {
			t.Fatal("failed to exec to pod:", err)
		}
		t.Log(output)
		needle := "To: " + email
		if strings.Contains(output, needle) {
			receivedEmail = true
		}

	}
	if !receivedEmail {
		return err
	}
	return nil
}

func sendTestEmail(ctx *TestingContext, t TestingTB) {
	// Send test email using the 3scale api
	if err := createTestingIDP(t, goctx.TODO(), ctx.Client, ctx.KubeConfig, ctx.SelfSignedCerts); err != nil {
		t.Fatalf("error while creating testing idp: %v", err)
	}
	// get console master url
	rhmi, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("error getting RHMI CR: %v", err)
	}

	// Get the fuse host url from the rhmi status
	host := rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.Product3Scale].Host
	if host == "" {
		host = fmt.Sprintf("https://3scale-admin.%v", rhmi.Spec.RoutingSubdomain)
	}
	keycloakHost := rhmi.Status.Stages[rhmiv1alpha1.InstallStage].Products[rhmiv1alpha1.ProductRHSSO].Host
	redirectURL := fmt.Sprintf("%v/p/admin/dashboard", host)

	tsClient := resources.NewThreeScaleAPIClient(host, keycloakHost, redirectURL, ctx.HttpClient, ctx.Client, t)

	// Login to 3Scale
	err = loginToThreeScale(t, host, threescaleLoginUser, TestingIdpPassword, "testing-idp", ctx.HttpClient)
	if err != nil {
		t.Fatalf("[%s] error ocurred: %v", getTimeStampPrefix(), err)
	}

	// Make sure 3Scale is available
	err = tsClient.Ping()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Sending email")
	_, err = tsClient.SendUserInvitation(emailAddress)
	if err != nil {
		t.Fatalf("[%s] error ocurred: %v", getTimeStampPrefix(), err)
	}
}

func reset3ScaleSecret(ctx *TestingContext, t TestingTB) error {
	t.Log("Resetting 3Scale secret")
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-smtp",
			Namespace: NamespacePrefix + "3scale",
		},
		Data: map[string][]byte{},
	}
	secret.Data["address"] = []byte(original3scaleHost)
	secret.Data["password"] = []byte(original3scalePassword)
	secret.Data["port"] = []byte(original3scalePort)
	secret.Data["username"] = []byte(original3scaleUsername)

	if err := ctx.Client.Update(goctx.TODO(), secret.DeepCopy(), &k8sclient.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func resetSecret(ctx *TestingContext, t TestingTB, isCreated bool) (string, error) {
	//Reset the smtp details back to the pre test version
	if isCreated {
		if err := ctx.Client.Delete(goctx.TODO(), &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: NamespacePrefix + "smtp", Namespace: NamespacePrefix + "operator"}}); err != nil {
			return "", err
		}
		t.Log("SMTP was deleted")
		return "", nil

	}

	secret, err := getSecret(ctx)

	if err != nil {
		return "", err
	}

	secret = v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamespacePrefix + "smtp",
			Namespace: NamespacePrefix + "operator",
		},
		Data: map[string][]byte{},
	}
	secret.Data["host"] = []byte(originalHost)
	secret.Data["password"] = []byte(originalPassword)
	secret.Data["port"] = []byte(originalPort)
	secret.Data["username"] = []byte(originalUsername)

	if err := ctx.Client.Update(goctx.TODO(), secret.DeepCopy(), &k8sclient.UpdateOptions{}); err != nil {
		return secret.APIVersion, err
	}

	return "", nil
}

func patch3ScaleSecret(ctx *TestingContext, t TestingTB) (string, error) {
	t.Log("Patching 3Scale secret")
	// Update secret with our test smtp details
	serviceIP, err := getServiceIP(ctx)
	if err != nil {
		return "", err
	}
	secret, err := get3scaleSecret(ctx)
	if err != nil {
		return "", err
	}

	original3scaleHost = string(secret.Data["host"])
	original3scalePassword = string(secret.Data["password"])
	original3scalePort = string(secret.Data["port"])
	original3scaleUsername = string(secret.Data["username"])

	secret = v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "system-smtp",
			Namespace: NamespacePrefix + "3scale",
		},
		Data: map[string][]byte{},
	}

	secret.Data["address"] = []byte(serviceIP)
	secret.Data["password"] = []byte(emailPassword)
	secret.Data["port"] = []byte(emailPort)
	secret.Data["username"] = []byte(emailUsername)

	if err = ctx.Client.Update(goctx.TODO(), secret.DeepCopy(), &k8sclient.UpdateOptions{}); err != nil {
		return secret.APIVersion, err
	}

	return "", nil
}
func patchSecret(ctx *TestingContext, t TestingTB) (string, bool, error) {
	// Update secret with our test smtp details
	serviceIP, err := getServiceIP(ctx)
	if err != nil {
		return "", false, err
	}

	secret, err := getSecret(ctx)
	if k8errors.IsNotFound(err) {
		t.Log("SMTP was not found , creating for test")
		secret := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      NamespacePrefix + "smtp",
				Namespace: NamespacePrefix + "operator",
			},
			Data: map[string][]byte{
				"username": []byte(emailUsername),
				"password": []byte(emailPassword),
				"host":     []byte(serviceIP),
				"port":     []byte(emailPort),
			},
		}

		if err := ctx.Client.Create(goctx.TODO(), secret.DeepCopy()); err != nil {
			return secret.APIVersion, false, err
		}
		t.Log("SMTP was created")
		return secret.APIVersion, true, err
	}

	if err != nil {
		return "", false, err
	}
	originalHost = string(secret.Data["host"])
	originalPassword = string(secret.Data["password"])
	originalPort = string(secret.Data["port"])
	originalUsername = string(secret.Data["username"])

	secret = v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamespacePrefix + "smtp",
			Namespace: NamespacePrefix + "operator",
		},
		Data: map[string][]byte{},
	}

	secret.Data["host"] = []byte(serviceIP)
	secret.Data["password"] = []byte(emailPassword)
	secret.Data["port"] = []byte(emailPort)
	secret.Data["username"] = []byte(emailUsername)

	if err := ctx.Client.Update(goctx.TODO(), secret.DeepCopy(), &k8sclient.UpdateOptions{}); err != nil {
		return secret.APIVersion, false, err
	}

	return "", false, nil
}

func getSecret(ctx *TestingContext) (v1.Secret, error) {

	secret := &v1.Secret{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: NamespacePrefix + "smtp", Namespace: NamespacePrefix + "operator"}, secret); err != nil {
		return *secret, err
	}
	return *secret, nil
}

func get3scaleSecret(ctx *TestingContext) (v1.Secret, error) {

	secret := &v1.Secret{}
	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: "system-smtp", Namespace: NamespacePrefix + "3scale"}, secret); err != nil {
		return *secret, err
	}
	return *secret, nil
}

func getServiceIP(ctx *TestingContext) (string, error) {
	service := &v1.Service{}

	if err := ctx.Client.Get(goctx.TODO(), types.NamespacedName{Name: "smtp-server", Namespace: "smtp-server"}, service); err != nil {
		return service.Spec.ClusterIP, err
	}

	return service.Spec.ClusterIP, nil
}

func createNamespace(ctx *TestingContext, t TestingTB) error {

	// Create namespace for our test smtp-server
	t.Log("Creating namespace, deployment and service")
	nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}

	_, err := ctx.KubeClient.CoreV1().Namespaces().Create(goctx.TODO(), nsSpec, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create namespace : %v", err)
	}

	// Create deployement for smtp-server
	deployment := &k8sv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "smtp-server",
		},
		Spec: k8sv1.DeploymentSpec{
			Replicas: &smtpReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "smtp-server",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "smtp-server",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "smtp-server",
							Image: "quay.io/cathaloconnor/smtp-server:latest",
							Ports: []v1.ContainerPort{
								{
									Name:          "tcp",
									Protocol:      v1.ProtocolTCP,
									ContainerPort: 1587,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = ctx.KubeClient.AppsV1().Deployments(testNamespace).Create(goctx.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create deployment : %v", err)
	}

	// Create service for smtp-server
	service := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "smtp-server",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Port:       int32(1587),
				TargetPort: intstr.FromInt(1587),
				Protocol:   v1.ProtocolTCP,
			}},
			Selector: map[string]string{
				"app": "smtp-server",
			},
		},
	}

	_, err = ctx.KubeClient.CoreV1().Services(testNamespace).Create(goctx.TODO(), &service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create service : %v", err)
	}
	return nil
}

func scale3scaleDeployment(t TestingTB, name string, namespace string, replicas int32, client k8sclient.Client) {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		getErr := client.Get(goctx.TODO(), k8sclient.ObjectKey{Name: name, Namespace: namespace}, deployment)
		if getErr != nil {
			return fmt.Errorf("failed to get deployment %s in namespace %s with error: %s", name, namespace, getErr)
		}

		deployment.Spec.Replicas = &replicas
		updateErr := client.Update(goctx.TODO(), deployment)
		return updateErr
	})
	if retryErr != nil {
		t.Logf("update failed: %v", retryErr)
	}
}

func removeNamespaceLogErrors(t TestingTB, ctx *TestingContext) {
	err := removeNamespace(t, ctx)
	if err != nil {
		t.Logf("error cleaning up namespace, %v", err)
	}
}

func resetSecretData(t TestingTB, ctx *TestingContext, data map[string][]byte, secret *v1.Secret) {
	t.Log("Resetting data")
	_, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, secret, func() error {
		secret.Data = make(map[string][]byte)
		for key, value := range data {
			secret.Data[key] = value
		}
		return nil
	})
	if err != nil {
		t.Errorf(err.Error())
	}
}

func copyAddonSecretData(secret *v1.Secret, ctx *TestingContext, inst *rhmiv1alpha1.RHMI) (map[string][]byte, error) {
	// copy the current custom smtp values
	newSecret, err := addonSecret(inst, ctx.Client)
	if err != nil {
		return nil, fmt.Errorf("error getting addon: %v", err)
	}

	data := make(map[string][]byte)
	for key, value := range newSecret.Data {
		data[key] = value
	}
	*secret = *newSecret
	return data, nil
}

func compareSMTPSecrets(inst *rhmiv1alpha1.RHMI, client k8sclient.Client, rhoamSecret string) error {

	// get the rhoam custom smtp secret
	customSmtp := &v1.Secret{}
	err := client.Get(context.TODO(), k8sclient.ObjectKey{
		Name:      rhoamSecret,
		Namespace: inst.Namespace,
	}, customSmtp)
	if err != nil {
		return err
	}

	// get the 3scale smtp secret
	smtp3scale := &v1.Secret{}
	err = client.Get(context.TODO(), k8sclient.ObjectKey{
		Name:      "system-smtp",
		Namespace: fmt.Sprintf("%s3scale", inst.Spec.NamespacePrefix),
	}, smtp3scale)

	if err != nil {
		return err
	}
	// compare the values on the secrets

	message := ""
	pass := true
	if string(customSmtp.Data["host"]) != string(smtp3scale.Data["address"]) {
		pass = false
		message = message + "Mismatch in host name, "
	}

	if string(customSmtp.Data["password"]) != string(smtp3scale.Data["password"]) {
		pass = false
		message = message + "Mismatch in password, "
	}

	if string(customSmtp.Data["port"]) != string(smtp3scale.Data["port"]) {
		pass = false
		message = message + "Mismatch in port, "
	}

	if string(customSmtp.Data["username"]) != string(smtp3scale.Data["username"]) {
		pass = false
		message = message + "Mismatch in username, "
	}

	if !pass {
		return fmt.Errorf(message)
	}

	return nil
}

func customSmtpParameters(inst *rhmiv1alpha1.RHMI, require string, client k8sclient.Client) (bool, error) {

	fields := []string{"custom-smtp-username", "custom-smtp-port", "custom-smtp-password",
		"custom-smtp-address", "custom-smtp-from_address"}

	secret, err := addonSecret(inst, client)
	if err != nil {
		return false, err
	}

	switch require {
	case none:
		for index := range fields {
			value, ok := secret.Data[fields[index]]
			if ok {
				if string(value) != "" {
					return false, nil
				}
			}
		}
		return true, nil
	case partial:
		isPartial := 0
		for index := range fields {
			value, ok := secret.Data[fields[index]]
			if ok {
				if string(value) == "" {
					isPartial++
				}
			} else {
				isPartial++
			}
		}
		if isPartial < len(fields) && isPartial > 0 {
			return true, nil
		}
		return false, nil
	case full:
		for index := range fields {
			value, ok := secret.Data[fields[index]]

			if ok {
				if string(value) == "" {
					return false, nil
				}
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	}
	return false, nil
}

func addonSecret(inst *rhmiv1alpha1.RHMI, client k8sclient.Client) (*v1.Secret, error) {
	secret, err := addon.GetAddonParametersSecret(context.TODO(), client, inst.Namespace)
	if err != nil {
		return nil, err
	}
	return secret, nil
}
