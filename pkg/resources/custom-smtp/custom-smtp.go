package custom_smtp

import (
	"context"
	"fmt"
	"github.com/integr8ly/integreatly-operator/api/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/addon"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	Valid        ValidationResponse = "valid"
	Partial      ValidationResponse = "partial"
	Blank        ValidationResponse = "blank"
	CustomSecret string             = "custom-smtp" // #nosec G101 -- This is a false positive
)

type ValidationResponse string

type CustomSmtp struct {
	FromAddress string
	Address     string
	Password    string
	Port        string
	Username    string
}

func GetCustomAddonValues(serverClient k8sclient.Client, namespace string) (*CustomSmtp, error) {

	secret, err := addon.GetAddonParametersSecret(context.TODO(), serverClient, namespace)
	if err != nil {
		return nil, err
	}

	customSmtp := &CustomSmtp{}

	value, ok := secret.Data["custom-smtp-from_address"]
	if ok {
		customSmtp.FromAddress = string(value)
	}

	value, ok = secret.Data["custom-smtp-address"]
	if ok {
		customSmtp.Address = string(value)
	}

	value, ok = secret.Data["custom-smtp-password"]
	if ok {
		customSmtp.Password = string(value)
	}

	value, ok = secret.Data["custom-smtp-port"]
	if ok {
		customSmtp.Port = string(value)
	}

	value, ok = secret.Data["custom-smtp-username"]
	if ok {
		customSmtp.Username = string(value)
	}

	return customSmtp, nil
}

// ParameterValidation If any field is populated then we consider this an attempt to use custom smtp and mark it as valid.
// In which case, if the mandatory fields are not all populated we mark it as partial in order to report back to the
// customer which fields need rectification.
func ParameterValidation(smtp *CustomSmtp) ValidationResponse {

	valid, partial := false, false

	if smtp.Port != "" {
		valid = true
	} else {
		partial = true
	}

	if smtp.Address != "" {
		valid = true
	} else {
		partial = true
	}

	if smtp.Username != "" {
		valid = true
	} else {
		partial = true
	}

	if smtp.FromAddress != "" {
		valid = true
	} else {
		partial = true
	}

	if smtp.Password != "" {
		valid = true
	} else {
		partial = true
	}

	if valid && !partial {
		return Valid
	} else if valid && partial {
		return Partial
	} else {
		return Blank
	}

}

func ParameterErrors(smtp *CustomSmtp) string {

	message := ""

	if smtp.Port == "" {
		message = message + "Port, "
	}

	if smtp.Address == "" {
		message = message + "Address, "
	}

	if smtp.Username == "" {
		message = message + "username, "
	}

	if smtp.FromAddress == "" {
		message = message + "From_Address, "
	}

	if smtp.Password == "" {
		message = message + "Password, "
	}

	return message
}

func CreateOrUpdateCustomSMTPSecret(ctx context.Context, serverClient k8sclient.Client, smtp *CustomSmtp, namespace string) (v1alpha1.StatusPhase, error) {
	if smtp == nil {
		return v1alpha1.PhaseFailed, fmt.Errorf("nill pointer passed for smtp details")
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CustomSecret,
			Namespace: namespace,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, serverClient, secret, func() error {
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		secret.Data["from_address"] = []byte(smtp.FromAddress)
		secret.Data["host"] = []byte(smtp.Address)
		secret.Data["password"] = []byte(smtp.Password)
		secret.Data["port"] = []byte(smtp.Port)
		secret.Data["username"] = []byte(smtp.Username)

		return nil
	}); err != nil {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseCompleted, nil

}

func DeleteCustomSMTP(ctx context.Context, serverClient k8sclient.Client, namespace string) (v1alpha1.StatusPhase, error) {

	secret := &corev1.Secret{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{
		Name:      CustomSecret,
		Namespace: namespace,
	}, secret)

	if err != nil {
		if k8serr.IsNotFound(err) {
			return v1alpha1.PhaseCompleted, nil
		}
		return v1alpha1.PhaseFailed, err
	}

	err = serverClient.Delete(ctx, secret)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	return v1alpha1.PhaseCompleted, nil
}

func GetFromAddress(ctx context.Context, serverClient k8sclient.Client, namespace string) (string, error) {
	secret := &corev1.Secret{}
	err := serverClient.Get(ctx, k8sclient.ObjectKey{
		Name:      CustomSecret,
		Namespace: namespace,
	}, secret)

	if err != nil {
		return "", err
	}

	return string(secret.Data["from_address"]), nil

}
