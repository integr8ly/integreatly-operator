package resources

import (
	"context"
	"errors"
	"testing"

	crov1 "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1"
	croTypes "github.com/integr8ly/cloud-resource-operator/api/integreatly/v1alpha1/types"
	moqclient "github.com/integr8ly/integreatly-operator/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRedisEngineForReconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := crov1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add cro scheme: %v", err)
	}

	ctx := context.Background()
	ns := "test-ns"
	name := "test-redis"

	t.Run("returns valkey for new redis", func(t *testing.T) {
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		engine, version, err := RedisEngineForReconcile(ctx, client, name, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if engine != croTypes.EngineValkey {
			t.Fatalf("expected engine %q, got %q", croTypes.EngineValkey, engine)
		}
		if version != "" {
			t.Fatalf("expected empty version, got %q", version)
		}
	})

	t.Run("preserves existing redis engine", func(t *testing.T) {
		existing := &crov1.Redis{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: croTypes.ResourceTypeSpec{
				Engine:        croTypes.EngineRedis,
				EngineVersion: "7.1",
			},
		}
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
		engine, version, err := RedisEngineForReconcile(ctx, client, name, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if engine != croTypes.EngineRedis {
			t.Fatalf("expected engine %q, got %q", croTypes.EngineRedis, engine)
		}
		if version != "7.1" {
			t.Fatalf("expected version %q, got %q", "7.1", version)
		}
	})

	t.Run("preserves unset engine for existing installation", func(t *testing.T) {
		existing := &crov1.Redis{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec:       croTypes.ResourceTypeSpec{},
		}
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
		engine, version, err := RedisEngineForReconcile(ctx, client, name, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if engine != "" {
			t.Fatalf("expected empty engine for existing CR, got %q", engine)
		}
		if version != "" {
			t.Fatalf("expected empty version, got %q", version)
		}
	})

	t.Run("preserves existing valkey engine", func(t *testing.T) {
		existing := &crov1.Redis{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: croTypes.ResourceTypeSpec{
				Engine: croTypes.EngineValkey,
			},
		}
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
		engine, version, err := RedisEngineForReconcile(ctx, client, name, ns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if engine != croTypes.EngineValkey {
			t.Fatalf("expected engine %q, got %q", croTypes.EngineValkey, engine)
		}
		if version != "" {
			t.Fatalf("expected empty version, got %q", version)
		}
	})

	t.Run("returns error when get fails", func(t *testing.T) {
		getErr := errors.New("api server unavailable")
		client := moqclient.NewSigsClientMoqWithScheme(scheme)
		client.GetFunc = func(ctx context.Context, key types.NamespacedName, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
			return getErr
		}
		_, _, err := RedisEngineForReconcile(ctx, client, name, ns)
		if !errors.Is(err, getErr) {
			t.Fatalf("expected error %v, got %v", getErr, err)
		}
	})
}
