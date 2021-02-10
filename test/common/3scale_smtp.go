package common

import (
	goctx "context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	rhmiv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/config"
	"github.com/integr8ly/integreatly-operator/test/resources"

	k8sv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "smtp-server"
)

var (
	s1                            = rand.NewSource(time.Now().UnixNano())
	r1                            = rand.New(s1)
	smtpReplicas            int32 = 1
	threescaleLoginUserSMTP       = fmt.Sprintf("%v%02d", defaultDedicatedAdminName, 1)
	emailAddress                  = fmt.Sprintf("test%v@test.com", r1.Intn(200))
	serviceIP                     = ""
	emailUsername                 = "dummy"
	emailPassword                 = "dummy"
	emailPort                     = "1587"
	originalHost                  = ""
	originalPassword              = ""
	originalPort                  = ""
	originalUsername              = ""
)

//Test3ScaleSMTPConfig to confirm 3scale can send an email
func Test3ScaleSMTPConfig(t TestingTB, ctx *TestingContext) {
	inst, err := GetRHMI(ctx.Client, true)
	if err != nil {
		t.Fatalf("failed to get RHMI instance %v", err)
	}

	t.Log("Create Namespace, Deployment and Service for SMTP-Server")
	err = createNamespace(ctx, t)
	if err != nil {
		t.Logf("%v", err)
	}

	_, err = patchSecret(ctx, t)
	if err != nil {
		t.Log(err)
	}

	t.Log("Wait for reconciliation of SMTP details")
	err = checkSMTPReconciliation(ctx, t)
	if err != nil {
		t.Fatalf("Unable to reconcile smtp details : %v ", err)
	}

	// Scale down system-app and system-sidekiq in order to load new smtp config
	for _, dc := range []string{"system-app", "system-sidekiq"} {
		t.Logf("Scalind down dc '%s' to 0 replicas in '%s' namespace", dc, threescaleNamespace)
		err = scaleDeploymentConfig(dc, threescaleNamespace, 0, ctx.Client)
		if err != nil {
			t.Errorf("Failed to scale down %s: %v ", dc, err)
		}
	}

	t.Log("Checking pods are ready")
	threescaleConfig := config.NewThreeScale(map[string]string{})
	replicas := threescaleConfig.GetReplicasConfig(inst)
	if err := check3ScaleReplicasAreReady(ctx, t, replicas, retryInterval, timeout); err != nil {
		t.Logf("Replicas not Ready within timeout: %v", err)
	}

	// Add sleep to give threescale time to reconcile the pods restarts otherwise host address will update during next steps
	time.Sleep(30 * time.Second)
	t.Log("Checking host address is ready")
	err = checkHostAddressIsReady(ctx, t, retryInterval, timeout)
	if err != nil {
		t.Log(err)
	}

	t.Log("Send Test email")
	sendTestEmail(ctx, t)

	t.Log("confirm email received")
	err = checkEmail(ctx, t, emailAddress)
	if err != nil {
		t.Fatal("No email found")
	}

	t.Log("Reset email details")
	_, err = resetSecret(ctx, t)
	if err != nil {
		t.Log(err)
	}

	t.Log("Removing smtp-server namespace")
	err = removeNamespace(ctx)
	if err != nil {
		t.Log(err)
	}

}

func checkHostAddressIsReady(ctx *TestingContext, t TestingTB, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {

		// get console master url
		rhmi, err := GetRHMI(ctx.Client, true)
		if err != nil {
			t.Fatalf("error getting RHMI CR: %v", err)
		}

		host := rhmi.Status.Stages[rhmiv1alpha1.ProductsStage].Products[rhmiv1alpha1.Product3Scale].Host
		status := rhmi.Status.Stages[rhmiv1alpha1.ProductsStage].Products[rhmiv1alpha1.Product3Scale].Status
		if host == "" || status == "in progress" {
			t.Log("3scale host URL not ready yet.")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		//return fmt.Error("Number of replicas for threescale replicas is not correct : Replicas - %w, Expected")
		return fmt.Errorf("Error, Host url not ready before timeout - %v", err)
	}
	return nil

}

func removeNamespace(ctx *TestingContext) error {
	//Remove the smtp-server namespace to clean up after test
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
	return wait.Poll(retryInterval, timeout, func() (done bool, err error) {
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
	if err != nil {
		t.Logf("Couldn't find pods: %v", err)
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
	if receivedEmail == false {
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
	host := rhmi.Status.Stages[rhmiv1alpha1.ProductsStage].Products[rhmiv1alpha1.Product3Scale].Host
	if host == "" {
		host = fmt.Sprintf("https://3scale-admin.%v", rhmi.Spec.RoutingSubdomain)
	}
	keycloakHost := rhmi.Status.Stages[rhmiv1alpha1.AuthenticationStage].Products[rhmiv1alpha1.ProductRHSSO].Host
	redirectURL := fmt.Sprintf("%v/p/admin/dashboard", host)

	tsClient := resources.NewThreeScaleAPIClient(host, keycloakHost, redirectURL, ctx.HttpClient, ctx.Client, t)

	// Login to 3Scale
	err = loginToThreeScale(t, host, threescaleLoginUser, DefaultPassword, "testing-idp", ctx.HttpClient)
	if err != nil {
		t.Skip("Skipping due to known flaky behavior, to be fixed ASAP.\nJIRA:  https://issues.redhat.com/browse/MGDAPI-558")
		// t.Fatalf("[%s] error ocurred: %v", getTimeStampPrefix(), err)
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

func resetSecret(ctx *TestingContext, t TestingTB) (string, error) {
	//Reset the smtp details back to the pre test version
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

func patchSecret(ctx *TestingContext, t TestingTB) (string, error) {
	// Update secret with our test smtp details
	serviceIP, err := getServiceIP(ctx)

	secret, err := getSecret(ctx)

	if err != nil {
		return "", err
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
		return secret.APIVersion, err
	}

	return "", nil
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
		return fmt.Errorf("Unable to create namespace : %v", err)
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
		return fmt.Errorf("Unable to create deployment : %v", err)
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
		return fmt.Errorf("Unable to create service : %v", err)
	}
	return nil
}
