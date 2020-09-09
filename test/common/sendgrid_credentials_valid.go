package common

import (
	"crypto/tls"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/smtp"
	"testing"
)

func TestSendgridCredentialsAreValid(t *testing.T, ctx *TestingContext) {
	// Get SMTP secret from rhmi-operator namespace
	kc := ctx.KubeClient
	smtpSecret, err := kc.CoreV1().Secrets(RHMIOperatorNamespace).Get("redhat-rhmi-smtp", metav1.GetOptions{})
	if err != nil {
		t.Fatal("Failed to get an SMTP secret", err)
	}

	username, password, host := string(smtpSecret.Data["username"]), string(smtpSecret.Data["password"]), string(smtpSecret.Data["host"])

	// Test if SMTP credentials are valid using smtp.Auth method
	auth := smtp.PlainAuth("", username, password, host)
	client, err := smtp.Dial(host + ":587")
	if err != nil {
		t.Fatal("Failed to create an SMTP client", err)
	}
	err = client.StartTLS(&tls.Config{ServerName: host})
	if err != nil {
		t.Fatal("Failed to encrypt the communication between an SMTP client and a server", err)
	}
	err = client.Auth(auth)
	if err != nil {
		t.Fatal("Failed to authenticate an SMTP client with provided credentials", err)
	}
}
