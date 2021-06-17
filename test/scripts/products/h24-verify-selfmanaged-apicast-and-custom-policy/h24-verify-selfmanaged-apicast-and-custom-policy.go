// This script is automation of SOP H24 - Verify self-managed APIcast and custom policy
// This test case should prove that it is possible for customers to deploy self-managed APIcast
// and use custom policies on it.
// The 3scale QE team will perform this test case in RHOAM each time there is an upgrade of 3scale.
// RHOAM QE should perform this if there are modifications on RHOAM end that might break the functionality
// - typically changes in permissions in RHOAM and/or OSD.
// Additional context can be found in MGDAPI-370 - https://issues.redhat.com/browse/MGDAPI-370

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	l "github.com/integr8ly/integreatly-operator/pkg/resources/logger"
)

var (
	log                 = l.NewLogger()
	basepath            = "../../../../"
	apicast_namespace   = "selfmanaged-apicast"
	openshift_stage_url = "https://api.stage.openshift.com/"
)

// Test main flow
func main() {
	var apicastImageTag string
	var adminPassword string
	adminUser := "customer-admin01"
	var ocmToken string
	flag.StringVar(&apicastImageTag, "apicast-image-tag", "", "apicast-image-tag")
	flag.StringVar(&adminPassword, "admin-password", "", "Admin Password")
	flag.StringVar(&ocmToken, "ocm-token", "123", "OCM Token")
	flag.Parse()

	log.Info("Image tag: " + apicastImageTag + "Passwd: " + adminPassword + "  OCM Token: " + ocmToken)

	//OCM Login
	err := ocm_login(ocmToken)
	if err != nil {
		log.Error("Error ocm_login(), Exiting...", err)
		return
	}

	// Create a new namespace (selfmanaged-apicast) for self-managed APIcast (SOP item 2)
	err = create_new_namespace()
	if err != nil {
		log.Error("Error in create_new_namespace(), Exiting...", err)
		return
	}
	time.Sleep(10 * time.Second)

	// Get 3scale admin token (SOP item 10)
	// using kubeadmin, as customer user have no permission for get secret system-seed
	token3scale, err := get_3scale_admin_token()
	if err != nil {
		log.Error("Error get_3scale_admin_token, Exiting...", err)
		return
	}
	log.Info("3scale admin token: " + token3scale)

	// Create customer users in dedicated-admins group
	err = create_customer_users(adminPassword, ocmToken)
	if err != nil {
		log.Error("Error in create_customer_users() - create customer users, Exiting...", err)
		return
	}

	// Customer login
	err = customer_login(adminUser, adminPassword)
	if err != nil {
		log.Error("Error in customer_login()", err)
		return //TODO!! remove commend
	}

	// Import APIcast Image (SOP item 3)
	err = import_apicast_image(apicastImageTag)
	if err != nil {
		log.Error("Error import_apicast_image(), Exiting...", err)
		return
	}

	// Create an adminportal-credentials secret (SOP item 11)
	err = create_adminportal_cred_secret(token3scale)
	if err != nil {
		log.Error("Error create_adminportal_cred_secret(), Exiting...", err)
		return
	}

	// Use self-managed APIcast instead of the builded one for API (SOP item 12 - Manual)
	setSelfManagedAPIcast()

	// Install "Red Hat Integration - 3scale APIcast gateway" operator (SOP item 13)
	err = install_3scale_apicast_gateway_operator()
	if err != nil {
		log.Error("Error in Step 13 - install_3scale_apicast_gateway_operator(), Exiting...", err)
		return
	}

	// Create a self-managed APIcast (SOP item 14)
	err = create_self_managed_apicast()
	if err != nil {
		log.Error("Error in Step 14 - create_self_managed_apicast(), Exiting...", err)
		return
	}

	// Create a route for the self-managed APIcast (SOP item 15)
	err = create_apicast_route()
	if err != nil {
		log.Error("Error in Step 14 - create_apicast_route(), Exiting...", err)
		return
	}

	log.Info("Test Completed for SOP H24-Verify self-managed APIcast and custom policy")
}

// Implementation of test flow steps

// Create customer users in dedicated-admins group
func create_customer_users(adminPassword, ocmToken string) error {
	log.Info("create_customer_users()")
	command := "PASSWORD=" + adminPassword + " DEDICATED_ADMIN_PASSWORD=" + adminPassword + " " + basepath + "scripts/setup-sso-idp.sh"
	err := run_shell_command(command, "create_customer_users()")
	return err
}

// Create a new namespace (selfmanaged-apicast) for self-managed APIcast (SOP item 2)
func create_new_namespace() error {
	log.Info("create_new namespace()")
	exists, err := check_if_namespace_exists(apicast_namespace)
	if err != nil {
		return err
	}
	if !exists {
		log.Info("Namespace creating")
		command := "oc new-project " + apicast_namespace
		err = run_shell_command(command, "namespace()")
		return err
	}
	return nil
}

func check_if_namespace_exists(namespace string) (bool, error) {
	log.Info("check_if_namespace_exists()")
	command := "oc projects |grep " + namespace
	log.Info("Command: " + command)
	out, _ := exec.Command("sh", "-c", command).Output()
	log.Info("Out from grep namespace: " + string(out))
	if strings.Contains(string(out), namespace) {
		log.Info("Namespace exists")
		return true, nil
	}
	log.Info("Namespace dose not exists")
	return false, nil
}

// Import APIcast Image (SOP item 3)
func import_apicast_image(tag string) error {
	log.Info("import_apicast_image")
	//command: oc import-image 3scale-amp2/apicast-gateway-rhel8:<tag> --from=registry.redhat.io/3scale-amp2/apicast-gateway-rhel8:<tag> --confirm
	command := "oc import-image 3scale-amp2/apicast-gateway-rhel8:" + tag + " --from=registry.redhat.io/3scale-amp2/apicast-gateway-rhel8:" + tag + " --confirm"
	err := run_shell_command(command, "import_apicast_image()")
	return err
}

// step 4. Create a secret with credentials to `registry.redhat.io`.
// -- NOT used in current script version --
// SKIP if step 3 is ok
// This function is not in use meanwhile (not verified), as
// OSD clusters are allowed to access that registry by default
func create_redhatio_secret(user, passwd, email string) error {
	log.Info("create_redhatio_secret()")
	command := "oc creat secret docker-registry redhatio --docker-server=registry.redhat.io --docker-username=" + user + " --docker-password=" + passwd + " --docker-email=" + email
	err := run_shell_command(command, "create_redhatio_secret()")
	return err
}

// step 5. Configure the pull secret.
// -- NOT used in current script version --
// SKIP if step 3 is ok
// This function is not in use meanwhile (not verified), as
// OSD clusters are allowed to access that registry by default
func configure_pull_secret() error {
	log.Info("configure_pull_secret()")
	command := "oc secrets link default redhatio --for=pull"
	err := run_shell_command(command, "configure_pull_secret()")
	return err
}

// Get 3scale admin token (SOP item 10)
func get_3scale_admin_token() (string, error) {
	log.Info("get_3scale_admin_token()")
	command := "oc get secret system-seed -n redhat-rhoam-3scale --template '{{index .data \"ADMIN_ACCESS_TOKEN\"}}' | base64 -d"
	log.Info("command: " + command)
	out, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		log.Error("err", err)
		return "", err
	}
	token := string(out)
	log.Info("Admin Portal token: " + token)
	return token, nil
}

// Create an adminportal-credentials secret (SOP item 11)
func create_adminportal_cred_secret(token3scale string) error {
	log.Info("create_adminportal_cred_secret()")
	//get route
	route, err := get_3scale_admin_route()
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	route = strings.TrimSuffix(route, "\n")
	log.Info("Route: " + route)
	//get admin_portal
	command := "oc get route " + route + " -n redhat-rhoam-3scale -ojson |jq '.metadata.annotations.\"zync.3scale.net/host\"'"
	log.Info("command: " + command)
	out, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	admin_portal := string(out)
	log.Info("Admin portal: " + admin_portal)
	// create secret
	command = "oc create secret generic adminportal-credentials --from-literal=AdminPortalURL=https://" + token3scale + "@" + admin_portal
	log.Info("command: " + command)
	err = run_shell_command(command, "create_adminportal_cred_secret()")
	return err
}

func get_3scale_admin_route() (string, error) {
	log.Info("get_3scale_admin_route()")
	command := "oc get route -n redhat-rhoam-3scale |grep admin |awk '{print $1}'"
	route, err := run_shell_command_get_output(command, "get_3scale_admin_route()")
	return route, err
}

// Use self-managed APIcast instead of the builded one for API (SOP item 12 - Manual)
// - navigate to 3scale Admin Portal (web console) route can be got with `oc get routes --namespace redhat-rhoam-3scale | grep admin`
// - In `API's\Products` on the Dashboard screen go to
// - API -> Integration -> Settings -> tick APIcast Self Managed radio-box
//    - change "Staging Public Base URL" so that it is slightly different at the beginning, e.g. replace `api-3scale-apicast-` with `selfmanaged-`
//    - then use the `Update Product` button
// - API -> Configuration -> Use the `Promote to Staging` and `Promote to Production` buttons
func setSelfManagedAPIcast() {
	log.Info("setSelfManagedAPIcast()")
	message := "This is the manual step, SOP item 12, - Use self-managed APIcast instead of the builded one for API (echo service) \n"
	message += "- Navigate to 3scale Admin Portal (web console) route can be got with  \"oc get routes --namespace redhat-rhoam-3scale | grep admin\" \n"
	message += "- In \"APIs Products\" on the Dashboard screen go to"
	message += "- API -> Integration -> Settings -> tick APIcast Self Managed radio-box \n"
	message += "- and change \"Staging Public Base URL\" - replace prefix \"api-3scale-apicast-\" with \"selfmanaged-\"\n"
	message += "- Then use the \"Update Product\" button \n"
	message += "- API -> Configuration -> Use the \"Promote to Staging\" and \"Promote to Production\" buttons + admin_portal)\n"
	log.Info(message)
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Waiting for manual step completion. Press enter when done :")
	reader.ReadString('\n')
	log.Info("SOP item 12 (Manual) - completed, continue ...")
}

// Step 13. Install "Red Hat Integration - 3scale APIcast gateway" operator
func install_3scale_apicast_gateway_operator() error {
	log.Info("install_3scale_apicast_gateway_operator()")
	command := "oc apply -f selfmanaged-apicast-operator-group.yaml"
	err := run_shell_command(command, "install_3scale_apicast_gateway_operator()")
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	command = "oc apply -f apicast-operator-subscription.yaml"
	err = run_shell_command(command, "install_3scale_apicast_gateway_operator()")
	return err
}

// Create a self-managed APIcast (SOP item 14)
func create_self_managed_apicast() error {
	log.Info("create_self_managed_apicast()")
	command := "oc apply -f apicast-example-apicast.yaml"
	err := run_shell_command(command, "create_self_managed_apicast()")
	return err
}

// Create a route for the self-managed APIcast (SOP item 15)
// route name: apicast-route
func create_apicast_route() error {
	log.Info("create_apicast_route()")
	command := "oc apply -f apicast-route.yaml"
	err := run_shell_command(command, "create_self_managed_apicast()")
	return err
}

// 16. Verify your work.
// check Configuration
func check_apicast_configuration() (bool, error) {
	return true, nil
}

// 17. Make custom-policy available in 3scale Admin Portal
// APIcast Example Policy
func check_apicast_example_policy() (bool, error) {
	return true, nil
}

func ocm_login(ocmToken string) error {
	log.Info("ocm_login()")
	command := "ocm login --token=" + ocmToken + " --url=" + openshift_stage_url
	log.Info("command: " + command)
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Error("ocm login error", err)
		return err
	}
	return nil
}

func customer_login(customer_admin_name, adminPassword string) error {
	log.Info("customer_login()")
	command := "oc login -u " + customer_admin_name + " -p " + adminPassword
	err := retry(20, 10*time.Second, func() (err error) {
		err = run_shell_command(command, "customer_login()")
		return
	})

	if err != nil {
		log.Error("Error login customer: ", err)
		return err
	}
	return nil
}

// Utils

func run_shell_command(command string, function_name string) error {
	log.Info("run_shell_command()")
	log.Info("run command for function: " + function_name)
	log.Info("command: " + command)
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	log.Info("run_command done")
	return nil
}

func run_shell_command_get_output(command string, function_name string) (string, error) {
	log.Info("run_shell_command_get_output()")
	log.Info("run command for function: " + function_name)
	log.Info(command)
	out, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		log.Error("Error: ", err)
		return "", err
	}
	outstr := string(out)
	log.Info("run command done")
	return outstr, nil
}

func retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return
		}
		if i >= (attempts - 1) {
			break
		}
		time.Sleep(sleep)
		log.Info("retrying after error")
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
