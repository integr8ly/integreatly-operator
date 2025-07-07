package addon

import (
	"context"
	"strconv"
	"testing"

	clientMock "github.com/integr8ly/integreatly-operator/pkg/client"
	"github.com/integr8ly/integreatly-operator/utils"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	integreatlyv1alpha1 "github.com/integr8ly/integreatly-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	multipleSubscriptions    = "multiple"

	testRHOAMnamespace  = "redhat-rhoam-operator"
	testRHOAMInamespace = "redhat-rhoami-operator"

	stringSecretValue   = "the boop"
	intSecretValue      = 420
	boolSecretValue     = "true"
	bytesSecretValue    = "boop"
	defaultParameterKey = "parameter"
)

func TestGetParameter(t *testing.T) {
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx       context.Context
		client    k8sclient.Client
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
				client:    getDefaultClient(scheme, testRHOAMnamespace, bytesSecret, addonSubscription),
			},
		},
		{
			Name:      "Parameter not found: not in secret",
			wantFound: false,
			args: args{
				parameter: "boop",
				namespace: testRHOAMnamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(scheme, testRHOAMnamespace, stringSecret, addonSubscription),
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
				client:    getDefaultClient(scheme, testRHOAMnamespace, noneSecret, addonSubscription),
			},
		},
		{
			Name:      "Parameter found: subscription is not present",
			wantFound: true,
			wantValue: []byte(bytesSecretValue),
			args: args{
				parameter: defaultParameterKey,
				namespace: testRHOAMnamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(scheme, testRHOAMnamespace, bytesSecret, noneSubscription),
			},
		},
		{
			Name:      "Parameter found in RHOAMI namespace",
			wantFound: true,
			wantValue: []byte(bytesSecretValue),
			args: args{
				parameter: defaultParameterKey,
				namespace: testRHOAMInamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(scheme, testRHOAMInamespace, bytesSecret, internalSubscriptionType),
			},
		},
		{
			Name:      "Parameter found for OLM installations",
			wantFound: true,
			wantValue: []byte(bytesSecretValue),
			args: args{
				parameter: defaultParameterKey,
				namespace: testRHOAMnamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(scheme, testRHOAMnamespace, bytesSecret, olmSubscriptionType),
			},
		},
		{
			Name:      "Multiple subscriptions fail secret retrieval",
			wantFound: false,
			wantErr:   true,
			args: args{
				parameter: defaultParameterKey,
				namespace: testRHOAMnamespace,
				ctx:       context.TODO(),
				client:    getDefaultClient(scheme, testRHOAMnamespace, bytesSecret, multipleSubscriptions),
			},
		},
		{
			Name:      "parameter not found: failed to list subscriptions",
			wantFound: false,
			wantErr:   true,
			args: args{
				parameter: defaultParameterKey,
				namespace: testRHOAMnamespace,
				ctx:       context.TODO(),
				client: &clientMock.SigsClientInterfaceMock{
					ListFunc: func(ctx context.Context, list k8sclient.ObjectList, opts ...k8sclient.ListOption) error {
						return genericError
					},
				},
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
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx       context.Context
		client    k8sclient.Client
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
				client:    getDefaultClient(scheme, testRHOAMInamespace, stringSecret, addonSubscription),
				namespace: testRHOAMInamespace,
				parameter: defaultParameterKey,
			},
			wantValue: stringSecretValue,
			wantOk:    true,
			wantErr:   false,
		},
		{
			name: "fail to retrieve string parameter: secret not present",
			args: args{
				ctx:       context.TODO(),
				client:    getDefaultClient(scheme, testRHOAMnamespace, noneSecret, addonSubscription),
				namespace: testRHOAMnamespace,
				parameter: defaultParameterKey,
			},
			wantErr: true,
			wantOk:  false,
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
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx       context.Context
		client    k8sclient.Client
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
				client:    getDefaultClient(scheme, testRHOAMnamespace, intSecret, olmSubscriptionType),
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
				client:    getDefaultClient(scheme, testRHOAMnamespace, stringSecret, olmSubscriptionType),
				namespace: testRHOAMnamespace,
				parameter: defaultParameterKey,
			},
			wantValue: 0,
			wantOk:    true,
			wantErr:   true,
		},
		{
			name: "failed to retrieve int parameter: not in a secret",
			args: args{
				ctx:       context.TODO(),
				client:    getDefaultClient(scheme, testRHOAMnamespace, intSecret, olmSubscriptionType),
				namespace: testRHOAMnamespace,
				parameter: "boop",
			},
			wantOk: false,
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
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx       context.Context
		client    k8sclient.Client
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
				client:    getDefaultClient(scheme, testRHOAMnamespace, boolSecret, addonSubscription),
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
				client:    getDefaultClient(scheme, testRHOAMnamespace, stringSecret, addonSubscription),
				namespace: testRHOAMnamespace,
				parameter: defaultParameterKey,
			},
			wantValue: false,
			wantOk:    true,
			wantErr:   true,
		},
		{
			name: "failed to retrieve bool parameter: not in a secret",
			args: args{
				ctx:       context.TODO(),
				client:    getDefaultClient(scheme, testRHOAMnamespace, boolSecret, addonSubscription),
				namespace: testRHOAMnamespace,
				parameter: "boop",
			},
			wantOk: false,
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
	scheme, err := utils.NewTestScheme()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx       context.Context
		client    k8sclient.Client
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
				client: getDefaultClient(scheme, testRHOAMnamespace, bytesSecret, addonSubscription),
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

func getDefaultClient(scheme *runtime.Scheme, namespace string, secretType SecretType, subscriptionType string) k8sclient.Client {
	return utils.NewTestClient(scheme, getValidInits(namespace, secretType, subscriptionType)...)
}

func getValidInits(namespace string, secretType SecretType, subscriptionType string) []runtime.Object {
	if subscriptionType == noneSubscription {
		return []runtime.Object{getSecretByType(namespace, secretType, subscriptionType)}
	} else {
		subs := []runtime.Object{
			&operatorsv1alpha1.Subscription{
				ObjectMeta: v1.ObjectMeta{
					Name:      subscriptionType,
					Namespace: namespace,
				},
			},
		}
		if subscriptionType == multipleSubscriptions {
			subs = append(subs, &operatorsv1alpha1.Subscription{
				ObjectMeta: v1.ObjectMeta{
					Name:      "boop",
					Namespace: namespace,
				},
			})
		}
		return append(subs, getSecretByType(namespace, secretType, subscriptionType))
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
