package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"github.com/integr8ly/integreatly-operator/pkg/resources/rhmi"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/products/threescale"
	"github.com/integr8ly/integreatly-operator/pkg/resources/logger"
	testcommon "github.com/integr8ly/integreatly-operator/test/common"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	baseName        = "h22-test"
	endpointTimeout = 10 * time.Minute
	counterTimeout  = 2 * time.Minute
)

var (
	testRunStart time.Time
	allSkipped   = true
)

func main() {
	var namespacePrefix string
	var iterations int
	var cleanupAPI bool

	flag.StringVar(&namespacePrefix, "namespace-prefix", "redhat-rhoam-", "Namespace prefix of RHOAM. Defaults to redhat-rhoam-")
	flag.IntVar(&iterations, "iterations", 10, "Number of times to perform requests and test the counter")
	flag.BoolVar(&cleanupAPI, "cleanup-api", true, "Whether to clean up the test API or not")

	flag.Parse()

	namespace := fmt.Sprintf("%soperator", namespacePrefix)

	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(crov1alpha1.SchemeBuilder.AddToScheme(scheme))
	utilruntime.Must(integreatlyv1alpha1.AddToScheme(scheme))

	config := ctrl.GetConfigOrDie()
	client, err := k8sclient.New(config, k8sclient.Options{
		Scheme: scheme,
	})
	kubeClient := kubernetes.NewForConfigOrDie(config)

	if err != nil {
		os.Exit(1)
	}

	ctx := context.Background()

	cleanupRedis(ctx, client, namespace)
	// Start creating the Redis Pod
	redisCreated, err := createRedis(ctx, client, namespace)
	if err != nil {
		osExit1(err, ctx, client, namespace, namespacePrefix, cleanupAPI, nil)
	}
	defer cleanupRedis(ctx, client, namespace)

	// Get the host of the redis rate limiting instance
	redisHost, err := getRedisHost(ctx, client, namespace)
	if err != nil {
		osExit1(err, ctx, client, namespace, namespacePrefix, cleanupAPI, nil)
	}
	fmt.Printf("ℹ️  Redis host: %s\n", redisHost)

	// Create the threescale API
	api, err := createAPI(ctx, client, namespacePrefix, baseName)
	if err != nil {
		osExit1(err, ctx, client, namespace, namespacePrefix, cleanupAPI, api)
	}
	fmt.Printf("️🔌  Created API. Endpoint %s\n", api.Endpoint)
	defer deleteAPI(cleanupAPI, api, ctx, client, namespacePrefix)

	// Wait for the Redis Pod to finish creating
	redisPod := <-redisCreated
	if redisPod.Error != nil {
		osExit1(err, ctx, client, namespace, namespacePrefix, cleanupAPI, api)
	}

	fmt.Println("️ℹ️  Throw away Redis Pod ready")
	fmt.Printf("  Redis pod name: %s\n", redisPod.PodName)

	redisCounter := &redisCounter{
		client:    kubeClient,
		config:    config,
		PodName:   redisPod.PodName,
		Namespace: namespace,
		RedisHost: redisHost,
	}

	// Perform repeated tests where a random number of requests is made
	// and assert that the counter is *at least* greater or equal than the
	// previous count + the number of requests made
	overallSuccess := true
	testRunStart = time.Now()
	fmt.Printf("[%s] Starting test run\n", testRunStart.Format("15:04:05"))
	for i := 0; i < iterations; i++ {
		numRequests := rand.IntnRange(5, 15)
		success, err := testCountIncreases(redisCounter, api.Endpoint, numRequests)
		if err != nil {
			osExit1(err, ctx, client, namespace, namespacePrefix, cleanupAPI, api)
		}

		overallSuccess = overallSuccess && success

		time.Sleep(time.Second)
	}

	if !overallSuccess || allSkipped {
		fmt.Println("Test failed, not all iterations succeeded")
		fmt.Println("Note: Counter reset can cause false failures due to current count being higher than previous count")
		osExit1(errors.New("cleaning up after a failure"), ctx, client, namespace, namespacePrefix, cleanupAPI, api)
	}
}

// createRedis creates a Redis throwaway instance, and returns a channel that
// will send the name of the Redis Pod once it's ready
func createRedis(ctx context.Context, client k8sclient.Client, namespace string) (<-chan struct {
	PodName string
	Error   error
}, error) {
	redis := &crov1alpha1.Redis{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "throw-away-redis-pod",
			Namespace: namespace,
		},
		Spec: types.ResourceTypeSpec{
			SecretRef: &types.SecretRef{
				Name: "throw-away-redis-ref",
			},
			Tier: "development",
			Type: "workshop",
		},
	}

	if err := client.Create(ctx, redis); err != nil {
		return nil, err
	}

	done := make(chan struct {
		PodName string
		Error   error
	})

	go func() {
		for {
			if err := client.Get(ctx, k8sclient.ObjectKey{
				Name:      redis.Name,
				Namespace: redis.Namespace,
			}, redis); err != nil {
				done <- struct {
					PodName string
					Error   error
				}{Error: err}
				return
			}

			if redis.Status.Phase != "complete" {
				fmt.Printf("  ☁️  Waiting for throwaway Redis container to complete. Current phase: %s...\n", redis.Status.Phase)
				time.Sleep(5 * time.Second)
				continue
			}

			podsList := &corev1.PodList{}
			if err := client.List(ctx, podsList, &k8sclient.ListOptions{
				Namespace: redis.Namespace,
			}); err != nil {
				done <- struct {
					PodName string
					Error   error
				}{Error: err}
				return
			}

			for _, pod := range podsList.Items {
				if strings.HasPrefix(pod.Name, redis.Name) {
					done <- struct {
						PodName string
						Error   error
					}{PodName: pod.Name}
					return
				}
			}

			done <- struct {
				PodName string
				Error   error
			}{Error: errors.New("redis created but Pod not found")}
			break
		}
	}()

	return done, nil
}

// cleanupRedis deletes the throwaway Redis instance
func cleanupRedis(ctx context.Context, client k8sclient.Client, namespace string) {
	redis := &crov1alpha1.Redis{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "throw-away-redis-pod",
			Namespace: namespace,
		},
	}

	if err := client.Delete(ctx, redis); err != nil && !k8serr.IsNotFound(err) {
		fmt.Printf("Failed to clean up throw away redis: %v. Please delete manually:\n", err)
		fmt.Println("Run: oc delete redis throw-away-redis-pod -n redhat-rhoam-operator")
	}

	fmt.Println("🗑️☁️  Deleted throw away Redis")
}

// testCountIncreases performs numberOfRequests simultaneously and asserts that
// the counter in Redis is increased at least that amount of times. Returns
// true with no error if the test succeeds. false with no error if the test failed
// and an error if an unexpected error occurred when performing the test
func testCountIncreases(r *redisCounter, endpoint string, numberOfRequests int) (bool, error) {
	// call endpoint for rate limiting key to show in redis
	for {
		err, code := requestEndpoint(endpoint)
		if err != nil && code != http.StatusNotFound && code != http.StatusForbidden {
			fmt.Printf("Error on initial request of endpoint: %s\n", err)
			break
		}
		if code == http.StatusOK {
			break
		}
		if testRunStart.Add(endpointTimeout).Before(time.Now()) {
			fmt.Println("Timeout waiting for the endpoint to become functional")
			break
		}
		fmt.Println("  ☁️  Waiting for the endpoint to become functional")
		time.Sleep(10 * time.Second)
	}

	initialCount, err := getCount(r, time.Now())
	if err != nil {
		return false, err
	}

	var wg sync.WaitGroup

	for i := 0; i < numberOfRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err, _ := requestEndpoint(endpoint)
			if err != nil {
				fmt.Printf("Error with requested end point, %s", err)
			}
		}()
	}
	wg.Wait()

	newCount, err := getCount(r, time.Now())
	if err != nil {
		return false, err
	}

	status := "[FAIL] ❌"
	success := false
	if newCount == 0 || initialCount == 0 {
		status = "[SKIP] ❌"
		success = true
	} else if newCount <= initialCount-numberOfRequests {
		status = "[PASS] 🎉"
		success = true
		allSkipped = false
	}

	fmt.Printf("[%s] Previous count: %d | Number of requests: %d | Current count: %d | %s\n",
		time.Now().Format("15:04:05"), initialCount, numberOfRequests, newCount, status)
	return success, nil
}

type threescaleAPI struct {
	threescaleClient threescale.ThreeScaleInterface
	accessToken      string

	AccountID string
	ServiceID string
	BackendID int

	Endpoint string
}

// createAPI creates a testing API with baseName
func createAPI(ctx context.Context, client k8sclient.Client, namespacePrefix, baseName string) (*threescaleAPI, error) {
	namespace := fmt.Sprintf("%soperator", namespacePrefix)
	tsNamespace := fmt.Sprintf("%s3scale", namespacePrefix)

	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system-seed",
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: s.Name, Namespace: tsNamespace}, s)
	if err != nil {
		return nil, err
	}
	accessToken := string(s.Data["ADMIN_ACCESS_TOKEN"])
	fmt.Printf("  🔑  Found access token: %s\n", accessToken)

	installation, err := rhmi.GetRhmiCr(client, ctx, namespace, logger.NewLogger())
	if err != nil {
		return nil, err
	}

	threescaleClient := newThreescaleClient(installation)

	accountID, err := threescaleClient.CreateAccount(accessToken,
		baseName,
		fmt.Sprintf("%s-user", baseName),
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("  ✔️  Created Account ID: %s\n", accountID)

	backendID, err := threescaleClient.CreateBackend(accessToken,
		fmt.Sprintf("%s-backend", baseName),
		"https://echo-api.3scale.net:443",
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("  ✔️  Created Backend ID: %d\n", backendID)

	metricID, err := threescaleClient.CreateMetric(accessToken,
		backendID,
		fmt.Sprintf("%s-metric", baseName),
		"hit",
	)
	if err != nil {
		return nil, err
	}

	fmt.Printf("  ✔️  Created Metric ID: %d\n", metricID)

	if err := threescaleClient.CreateBackendMappingRule(accessToken,
		backendID,
		metricID,
		"GET",
		"/",
		1,
	); err != nil {
		return nil, err
	}
	fmt.Println("  ✔️  Mapping rule created")

	serviceID, err := threescaleClient.CreateService(accessToken,
		fmt.Sprintf("%s-api", baseName),
		fmt.Sprintf("%s-api", baseName),
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("  ✔️  Created Service ID: %s\n", serviceID)

	if err = threescaleClient.CreateBackendUsage(accessToken,
		serviceID,
		backendID,
		"/",
	); err != nil {
		return nil, err
	}
	fmt.Println("  ✔️  Backend usage created")

	applicationPlanID, err := threescaleClient.CreateApplicationPlan(accessToken,
		serviceID,
		fmt.Sprintf("%s-api-plan", baseName),
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("  ✔️  Created Application Plan ID: %s\n", applicationPlanID)

	userKey, err := threescaleClient.CreateApplication(accessToken,
		accountID,
		applicationPlanID,
		fmt.Sprintf("%s-api-app", baseName),
		fmt.Sprintf("%s-api-app", baseName),
	)
	if err != nil {
		return nil, err
	}
	fmt.Printf("  ✔️  User key: %s\n", userKey)

	time.Sleep(5 * time.Second)

	if err := threescaleClient.DeployProxy(accessToken, serviceID); err != nil {
		return nil, err
	}
	fmt.Println("  ✔️  Proxy deployed")

	time.Sleep(5 * time.Second)

	endpoint, err := threescaleClient.PromoteProxy(accessToken, serviceID, "sandbox", "production")
	if err != nil {
		return nil, err
	}
	fmt.Printf("  ✔️  Promoted proxy. Endpoint: %s\n", endpoint)

	fullEndpoint := fmt.Sprintf("%s?user_key=%s", endpoint, userKey)
	return &threescaleAPI{
		threescaleClient: threescaleClient,
		accessToken:      accessToken,
		AccountID:        accountID,
		ServiceID:        serviceID,
		BackendID:        backendID,
		Endpoint:         fullEndpoint,
	}, nil
}

func deleteAPI(cleanup bool, api *threescaleAPI, ctx context.Context, client k8sclient.Client, namespacePrefix string) {
	if !cleanup {
		fmt.Println("Skipping API clean-up")
		return
	}

	fmt.Println("🗑️🔌  Cleaning up API")
	if api == nil {
		api = mock3scaleAPI(ctx, client, namespacePrefix)
	}

	err := api.threescaleClient.DeleteService(api.accessToken, api.ServiceID)
	if err != nil && !strings.Contains(err.Error(), "Not Found") {
		fmt.Printf("Failed to clean up API: failed to delete service: %v", err)
	} else {
		fmt.Println("  ✔️️  Deleted API service")
	}
	err = api.threescaleClient.DeleteBackend(api.accessToken, api.BackendID)
	if err != nil && !strings.Contains(err.Error(), "Not Found") {
		fmt.Printf("Failed to clean up API: failed to delete backend: %v", err)
	} else {
		fmt.Println("  ✔️️  Deleted API backend")
	}
	err = api.threescaleClient.DeleteAccount(api.accessToken, api.AccountID)
	if err != nil && !strings.Contains(err.Error(), "Not Found") {
		fmt.Printf("Failed to clean up API: failed to delete account: %v", err)
	} else {
		fmt.Println("  ✔️  Deleted API account")
	}
	if err == nil {
		fmt.Println("  ✔️  Deleted API. If re-testing, allow a few minutes (~10) for the changes to propagate in 3scale before launching the test again.")
	}
}

// getRedisHosts finds the host of the rate limiting Redis instance
func getRedisHost(ctx context.Context, client k8sclient.Client, namespace string) (string, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, k8sclient.ObjectKey{
		Name:      "ratelimit-service-redis-rhoam",
		Namespace: namespace,
	}, secret); err != nil {
		return "", err
	}

	host, ok := secret.Data["uri"]
	if !ok {
		return "", errors.New("uri not found in ratelimit redis secret")
	}

	return string(host), nil
}

// requestEndpoint sends a simple GET request to the endpoint and returns an
// error if the response is not OK and the response code
func requestEndpoint(endpoint string) (error, int) {
	c := &http.Client{}
	res, err := c.Get(endpoint)
	if err != nil {
		return err, res.StatusCode
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code. Expected %d got %d ", http.StatusOK, res.StatusCode), res.StatusCode
	}

	return nil, res.StatusCode
}

type redisCounter struct {
	client kubernetes.Interface
	config *rest.Config

	PodName   string
	Namespace string
	RedisHost string
}

// GetCount uses the throwaway Redis pod to obtain the value of the counter
// in the rate limit Redis instance
func (r *redisCounter) GetCount() (int, error) {
	keys, err := testcommon.ExecToPodArgs(r.client, r.config,
		[]string{
			"/opt/rh/rh-redis6/root/usr/bin/redis-cli",
			"-c",
			"-h",
			r.RedisHost,
			"-p",
			"6379",
			"KEYS",
			"*",
		},
		r.PodName,
		r.Namespace,
		"redis",
	)
	if err != nil {
		return 0, err
	}

	keys = strings.TrimSpace(strings.ReplaceAll(keys, "liveness-probe", ""))
	if keys == "" {
		return 0, nil
	}

	keyList := strings.Split(keys, "\n")
	total := 0
	for _, key := range keyList {

		isString, err := r.keyIsString(key)
		if !isString || err != nil {
			continue
		}

		count, err := testcommon.ExecToPodArgs(r.client, r.config,
			[]string{
				"/opt/rh/rh-redis6/root/usr/bin/redis-cli",
				"-c",
				"-h",
				r.RedisHost,
				"-p",
				"6379",
				"GET",
				key,
			},
			r.PodName,
			r.Namespace,
			"redis",
		)
		if err != nil {
			return 0, err
		}

		count = strings.TrimSpace(count)
		countInt := 0
		if count != "" {
			countInt, err = strconv.Atoi(count)
			if err != nil {
				return 0, err
			}
		}

		total += countInt
	}

	return total, nil
}

func (r *redisCounter) keyIsString(key string) (bool, error) {
	keyType, err := testcommon.ExecToPodArgs(r.client, r.config,
		[]string{
			"/opt/rh/rh-redis6/root/usr/bin/redis-cli",
			"-c",
			"-h",
			r.RedisHost,
			"-p",
			"6379",
			"TYPE",
			key,
		},
		r.PodName,
		r.Namespace,
		"redis",
	)
	if err != nil {
		return false, err
	}

	if strings.Contains(keyType, "string") {
		return true, nil
	}

	return false, nil
}

func newThreescaleClient(installation *integreatlyv1alpha1.RHMI) threescale.ThreeScaleInterface {
	/* #nosec */
	httpc := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   time.Second * 10,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: installation.Spec.SelfSignedCerts}, // gosec G402, value is read from CR config
		},
	}

	if installation.Spec.SelfSignedCerts {
		fmt.Println("TLS insecure skip verify is enabled")
	}
	return threescale.NewThreeScaleClient(httpc, installation.Spec.RoutingSubdomain)
}

// mock3scaleAPI used to simulate an API while creation of one failed but we still want to clean up resources
func mock3scaleAPI(ctx context.Context, client k8sclient.Client, namespacePrefix string) *threescaleAPI {
	namespace := fmt.Sprintf("%soperator", namespacePrefix)
	tsNamespace := fmt.Sprintf("%s3scale", namespacePrefix)

	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "system-seed",
		},
	}
	err := client.Get(ctx, k8sclient.ObjectKey{Name: s.Name, Namespace: tsNamespace}, s)
	if err != nil {
		os.Exit(1)
	}
	accessToken := string(s.Data["ADMIN_ACCESS_TOKEN"])
	fmt.Printf("  🔑  Found access token: %s\n", accessToken)
	installation, err := rhmi.GetRhmiCr(client, ctx, namespace, logger.NewLogger())
	if err != nil {
		fmt.Printf("Error getting installaction: %s\n", err)
		os.Exit(1)
	}
	return &threescaleAPI{
		threescaleClient: newThreescaleClient(installation),
		accessToken:      accessToken,
	}
}

// getCount queries redisCounter until an error or non 0 counter
func getCount(r *redisCounter, startTime time.Time) (int, error) {
	fmt.Println("  ☁️  Waiting for the counter to return a valid value")
	for {
		count, err := r.GetCount()
		if err != nil {
			return 0, err
		}
		if count > 0 {
			return count, nil
		}

		if startTime.Add(counterTimeout).Before(time.Now()) {
			fmt.Println("Timeout waiting for the counter to return > 0 count")
			return 0, err
		}
		time.Sleep(1 * time.Second)
	}
}

func osExit1(err error, ctx context.Context, client k8sclient.Client, namespace, namespacePrefix string, cleanupAPI bool, api *threescaleAPI) {
	fmt.Println(err)
	cleanupRedis(ctx, client, namespace)
	deleteAPI(cleanupAPI, api, ctx, client, namespacePrefix)
	os.Exit(1)
}
