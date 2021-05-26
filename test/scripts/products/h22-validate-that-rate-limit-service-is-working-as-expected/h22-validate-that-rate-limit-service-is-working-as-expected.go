package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	crov1alpha1 "github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1"
	"github.com/integr8ly/cloud-resource-operator/apis/integreatly/v1alpha1/types"
	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/products/threescale"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
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

	// Start creating the Redis Pod
	redisCreated, err := createRedis(ctx, client, namespace)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	defer cleanupRedis(ctx, client, namespace)

	// Get the host of the redis rate limiting instance
	redisHost, err := getRedisHost(ctx, client, namespace)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("ℹ️  Redis host: %s\n", redisHost)

	// Create the threescale API
	api, err := createAPI(ctx, client, namespacePrefix, "h22-test")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("️🔌  Created API. Endpoint %s\n", api.Endpoint)
	defer deleteAPI(cleanupAPI, api)

	// Wait for the Redis Pod to finish creating
	redisPod := <-redisCreated
	if redisPod.Error != nil {
		fmt.Println(err)
		os.Exit(1)
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
	if err := redisCounter.Flush(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Perform repeated tests where a random number of requests is made
	// and assert that the counter is *at least* greater or equal than the
	// previous count + the number of requests made
	overallSuccess := true
	for i := 0; i < iterations; i++ {
		numRequests := rand.IntnRange(5, 15)
		success, err := testCountIncreases(redisCounter, api.Endpoint, numRequests)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		overallSuccess = overallSuccess && success

		time.Sleep(time.Second)
	}

	if !overallSuccess {
		fmt.Println("Test failed, not all iterations succeeded")
		os.Exit(1)
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
			}{Error: errors.New("Redis created but Pod not found")}
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
	currentCount, err := r.GetCount()
	if err != nil {
		return false, err
	}

	var wg sync.WaitGroup

	for i := 0; i < numberOfRequests; i++ {
		go func() {
			requestEndpoint(endpoint)
			wg.Done()
		}()
		wg.Add(1)
	}

	wg.Wait()

	newCount, err := r.GetCount()
	if err != nil {
		return false, err
	}

	status := "[FAIL] ❌"
	success := false
	if newCount >= currentCount+numberOfRequests {
		status = "[PASS] 🎉"
		success = true
	}

	fmt.Printf("Previous count: %d | Number of requests: %d | Current count: %d | %s\n",
		currentCount, numberOfRequests, newCount, status)
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

	installation, err := resources.GetRhmiCr(client, ctx, namespace, logger.NewLogger())
	if err != nil {
		return nil, err
	}

	threescaleClient := newThreescaleClient(
		installation,
		accessToken,
	)

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
	fmt.Printf("  ✔️  Created Service ID: %s\n", serviceID)

	if err := threescaleClient.CreateBackendUsage(accessToken,
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

func deleteAPI(cleanup bool, api *threescaleAPI) {
	if !cleanup {
		fmt.Println("Skipping API clean-up")
		return
	}

	fmt.Println("🗑️🔌  Cleaning up API")

	if err := api.threescaleClient.DeleteService(api.accessToken, api.ServiceID); err != nil {
		fmt.Printf("Failed to clean up API: failed to delete service: %v", err)
		return
	}
	fmt.Println("  ✔️️  Deleted API service")

	if err := api.threescaleClient.DeleteBackend(api.accessToken, api.BackendID); err != nil {
		fmt.Printf("Failed to clean up API: failed to delete backend: %v", err)
		return
	}
	fmt.Println("  ✔️️  Deleted API backend")

	if err := api.threescaleClient.DeleteAccount(api.accessToken, api.AccountID); err != nil {
		fmt.Printf("Failed to clean up API: failed to delete account: %v", err)
		return
	}
	fmt.Println("  ✔️  Deleted API account")

	fmt.Println("✔️  Deleted API. If re-testing, allow a few minutes (~5) for the changes to propagate in 3scale before launching the test again.")
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
// error if the response is not OK
func requestEndpoint(endpoint string) error {
	c := &http.Client{}
	res, err := c.Get(endpoint)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code. Expected %d got %d", http.StatusOK, res.StatusCode)
	}

	return nil
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
			"/opt/rh/rh-redis32/root/usr/bin/redis-cli",
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

	keyList := strings.Fields(keys)
	total := 0
	for _, key := range keyList {
		count, err := testcommon.ExecToPod(r.client, r.config,
			fmt.Sprintf("/opt/rh/rh-redis32/root/usr/bin/redis-cli -c -h %s -p 6379 GET %s", r.RedisHost, key),
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

	if len(keyList) > 2 {
		if err := r.Flush(); err != nil {
			return total, err
		}
	}

	return total, nil
}

func (r *redisCounter) Flush() error {
	_, err := testcommon.ExecToPodArgs(r.client, r.config,
		[]string{
			"/opt/rh/rh-redis32/root/usr/bin/redis-cli",
			"-c",
			"-h",
			r.RedisHost,
			"-p",
			"6379",
			"FLUSHALL",
		},
		r.PodName,
		r.Namespace,
		"redis",
	)
	return err
}

func newThreescaleClient(installation *integreatlyv1alpha1.RHMI, accessToken string) threescale.ThreeScaleInterface {
	httpc := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   time.Second * 10,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: installation.Spec.SelfSignedCerts},
		},
	}

	return threescale.NewThreeScaleClient(httpc, installation.Spec.RoutingSubdomain)
}
