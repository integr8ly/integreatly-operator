package addon

import (
	"context"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"testing"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type SecretType string

var (
	bytesSecret  SecretType = "bytes"
	stringSecret SecretType = "string"
	intSecret    SecretType = "int"
	boolSecret   SecretType = "bool"
	noneSecret   SecretType = "none"

	addonSubscription        = "addon-managed-api-service"
	olmSubscriptionType      = "managed-api-service"
	internalSubscriptionType = "addon-managed-api-service-internal"
	noneSubscription         = "none"

	testRHOAMnamespace  = "redhat-rhoam-operator"
	testRHOAMInamespace = "redhat-rhoami-operator"

	stringSecretValue   = "the boop"
	intSecretValue      = 420
	boolSecretValue     = "true"
	bytesSecretValue    = "boop"
	defaultParameterKey = "parameter"
)

func TestGetParameter(t *testing.T) {
	type args struct {
		ctx       context.Context
		client    client.Client
		namespace string
		parameter string
	}
	scenarios := []struct {
		Name      string
		args      args
		wantFound bool
		wantValue []byte
		wantErr   bool
	}{
		{
			Name:      "Parameter found",
			wantFound: true,
			wantValue: []byte(bytesSecretValue),
			args: args{
				parameter: defaultParameterKey,
				namespace: testRHOAMnamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMnamespace, bytesSecret, addonSubscription),
			},
		},
		{
			Name:      "Parameter not found: not in secret",
			wantFound: false,
			args: args{
				parameter: "boop",
				namespace: testRHOAMnamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMnamespace, stringSecret, addonSubscription),
			},
		},
		{
			Name:      "Parameter not found: secret not defined",
			wantFound: false,
			wantErr:   true,
			args: args{
				parameter: defaultParameterKey,
				namespace: testRHOAMnamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMnamespace, noneSecret, addonSubscription),
			},
		},
		{
			Name:      "Parameter from const found",
			wantFound: true,
			wantValue: []byte(bytesSecretValue),
			args: args{
				parameter: defaultParameterKey,
				namespace: testRHOAMnamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMnamespace, bytesSecret, noneSubscription),
			},
		},
		{
			Name:      "Secret from RHOAMI namespace retrieved",
			wantFound: true,
			wantValue: []byte(bytesSecretValue),
			args: args{
				parameter: defaultParameterKey,
				namespace: testRHOAMInamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMInamespace, bytesSecret, internalSubscriptionType),
			},
		},
		{
			Name:      "OLM installations treated well",
			wantFound: true,
			wantValue: []byte(bytesSecretValue),
			args: args{
				parameter: defaultParameterKey,
				namespace: testRHOAMnamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMnamespace, bytesSecret, olmSubscriptionType),
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			result, ok, err := GetParameter(context.TODO(), scenario.args.client, scenario.args.namespace, scenario.args.parameter)
			if (err != nil) != scenario.wantErr {
				t.Fatalf("GetParameter() error = %v, wantErr %v", err, scenario.wantErr)
			}
			if ok != scenario.wantFound {
				t.Fatalf("GetParameter() ok = %v, wantFound %v", ok, scenario.wantFound)
			}
			if string(result) != string(scenario.wantValue) {
				t.Fatalf("GetParameter() result = %v, wantValue %v", result, scenario.wantValue)
			}
		})
	}
}

func TestGetStringParameter(t *testing.T) {
	scheme := runtime.NewScheme()
	err := integreatlyv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ctx       context.Context
		client    client.Client
		namespace string
		parameter string
	}
	tests := []struct {
		name      string
		args      args
		wantValue string
		wantOk    bool
		wantErr   bool
	}{
		{
			name: "retrieve the string value for an addon parameter",
			args: args{
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMInamespace, stringSecret, addonSubscription),
				namespace: testRHOAMInamespace,
				parameter: defaultParameterKey,
			},
			wantValue: stringSecretValue,
			wantOk:    true,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok, err := GetStringParameter(tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStringParameter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if value != tt.wantValue {
				t.Errorf("GetStringParameter() value = %v, wantValue %v", value, tt.wantValue)
			}
			if ok != tt.wantOk {
				t.Errorf("GetStringParameter() ok = %v, wantOk %v", ok, tt.wantOk)
			}
		})
	}
}

func TestGetIntParameter(t *testing.T) {
	scheme := runtime.NewScheme()
	err := integreatlyv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ctx       context.Context
		client    client.Client
		namespace string
		parameter string
	}
	tests := []struct {
		name      string
		args      args
		wantValue int
		wantOk    bool
		wantErr   bool
	}{
		{
			name: "retrieve the integer value for an addon parameter",
			args: args{
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMnamespace, intSecret, olmSubscriptionType),
				namespace: testRHOAMnamespace,
				parameter: defaultParameterKey,
			},
			wantValue: intSecretValue,
			wantOk:    true,
			wantErr:   false,
		},
		{
			name: "failed to parse string to integer",
			args: args{
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMnamespace, stringSecret, olmSubscriptionType),
				namespace: testRHOAMnamespace,
				parameter: defaultParameterKey,
			},
			wantValue: 0,
			wantOk:    true,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok, err := GetIntParameter(tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIntParameter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if value != tt.wantValue {
				t.Errorf("GetIntParameter() value = %v, wantValue %v", value, tt.wantValue)
			}
			if ok != tt.wantOk {
				t.Errorf("GetIntParameter() ok = %v, wantOk %v", ok, tt.wantOk)
			}
		})
	}
}

func TestGetBoolParameter(t *testing.T) {
	scheme := runtime.NewScheme()
	err := integreatlyv1alpha1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		ctx       context.Context
		client    client.Client
		namespace string
		parameter string
	}
	tests := []struct {
		name      string
		args      args
		wantValue bool
		wantOk    bool
		wantErr   bool
	}{
		{
			name: "retrieve the boolean value for an addon parameter",
			args: args{
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMnamespace, boolSecret, addonSubscription),
				namespace: testRHOAMnamespace,
				parameter: defaultParameterKey,
			},
			wantValue: true,
			wantOk:    true,
			wantErr:   false,
		},
		{
			name: "failed to parse string to boolean",
			args: args{
				ctx:       context.TODO(),
				client:    getDefaultClient(testRHOAMnamespace, stringSecret, addonSubscription),
				namespace: testRHOAMnamespace,
				parameter: defaultParameterKey,
			},
			wantValue: false,
			wantOk:    true,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok, err := GetBoolParameter(tt.args.ctx, tt.args.client, tt.args.namespace, tt.args.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBoolParameter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if value != tt.wantValue {
				t.Errorf("GetBoolParameter() value = %v, wantValue %v", value, tt.wantValue)
			}
			if ok != tt.wantOk {
				t.Errorf("GetBoolParameter() ok = %v, wantOk %v", ok, tt.wantOk)
			}
		})
	}
}

func TestExistsParameterByInstallation(t *testing.T) {
	type args struct {
		ctx       context.Context
		client    client.Client
		install   *integreatlyv1alpha1.RHMI
		parameter string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "parameter exists",
			args: args{
				ctx:    context.TODO(),
				client: getDefaultClient(testRHOAMnamespace, bytesSecret, addonSubscription),
				install: &integreatlyv1alpha1.RHMI{
					ObjectMeta: v1.ObjectMeta{
						Namespace: testRHOAMnamespace,
					},
				},
				parameter: defaultParameterKey,
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := ExistsParameterByInstallation(tt.args.ctx, tt.args.client, tt.args.install, tt.args.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExistsParameterByInstallation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if found != tt.want {
				t.Errorf("ExistsParameterByInstallation() found = %v, want %v", found, tt.want)
			}
		})
	}
}

func getDefaultClient(namespace string, secretType SecretType, subscriptionType string) client.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)

	return fake.NewFakeClientWithScheme(scheme, getValidInits(namespace, secretType, subscriptionType)...)
}

func getValidInits(namespace string, secretType SecretType, subscriptionType string) []runtime.Object {
	if subscriptionType == noneSubscription {
		return []runtime.Object{getSecretByType(namespace, secretType, subscriptionType)}
	} else {
		return append([]runtime.Object{
			&v1alpha1.Subscription{
				ObjectMeta: v1.ObjectMeta{
					Name:      subscriptionType,
					Namespace: namespace,
				},
			},
		}, getSecretByType(namespace, secretType, subscriptionType))
	}
}

func getSecretByType(namespace string, secretType SecretType, subscriptionType string) *corev1.Secret {
	if secretType == noneSecret {
		return &corev1.Secret{}
	}
	var value string

	switch secretType {
	case intSecret:
		value = strconv.Itoa(intSecretValue)
	case boolSecret:
		value = boolSecretValue
	case bytesSecret:
		value = bytesSecretValue
	case stringSecret:
		value = stringSecretValue
	}

	name := "addon-managed-api-service-parameters"
	switch subscriptionType {
	case addonSubscription:
		name = "addon-managed-api-service-parameters"
	case internalSubscriptionType:
		name = "addon-managed-api-service-internal-parameters"
	case olmSubscriptionType:
		name = "addon-managed-api-service-parameters"
	}

	return &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"parameter": []byte(value),
		},
	}
}
