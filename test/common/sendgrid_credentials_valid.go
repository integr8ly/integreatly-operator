package common

import (
	"context"
	"crypto/tls"
	"net/smtp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSendgridCredentialsAreValid(t TestingTB, ctx *TestingContext) {
	// Get SMTP secret from rhmi-operator namespace
	kc := ctx.KubeClient
	smtpSecret, err := kc.CoreV1().Secrets(RHMIOperatorNamespace).Get(context.TODO(), NamespacePrefix+"smtp", metav1.GetOptions{})
	if err != nil {
		t.Fatal("Failed to get an SMTP secret", err)
	}

	username, password, host, port :=
		string(smtpSecret.Data["username"]),
		string(smtpSecret.Data["password"]),
		string(smtpSecret.Data["host"]),
		string(smtpSecret.Data["port"])

	if host != "smtp.sendgrid.net" {
		t.Skip("Skipping the test, because the credentials were not set by a Sendgrid service")
	}

	// Test if SMTP credentials are valid using smtp.Auth method
	auth := smtp.PlainAuth("", username, password, host)
	client, err := smtp.Dial(host + ":" + port)
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
