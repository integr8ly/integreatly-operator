package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/mail"
	"net/smtp"
	"os"
)

func main() {
	fmt.Println("starting email sending hack script . . . beep beep boop boop")

	fmt.Println("validating input . . . boop")
	smtpHostPtr := flag.String("host", "email-smtp.eu-west-1.amazonaws.com", "smtp host without port")
	fromAddrPtr := flag.String("from", "", "email to send mail from")
	toAddrPtr := flag.String("to", "", "email to send mail to")
	authUserPtr := flag.String("user", "", "username for smtp auth")
	authPassPtr := flag.String("pass", "", "password for smtp auth")
	flag.Parse()
	if *fromAddrPtr == "" || *toAddrPtr == "" || *authUserPtr == "" || *authPassPtr == "" {
		flag.Usage()
		os.Exit(1)
	}

	hostWithPort := fmt.Sprintf("%s:%s", *smtpHostPtr, "465")
	mailAuth := smtp.PlainAuth("", *authUserPtr, *authPassPtr, *smtpHostPtr)

	from := mail.Address{"", *fromAddrPtr}
	to := mail.Address{"", *toAddrPtr}

	fmt.Println("building mail contents . . . beep beep")
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = to.String()
	headers["Subject"] = "Integreatly Cloud Resource Operator"
	mailContent := ""
	for k, v := range headers {
		mailContent += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	mailContent += "\r\n" + "test message from the integreatly cloud-resource-operator"

	// setup tls smtp client
	fmt.Println("setting up tls connection . . . beep boop")
	tlsCfg := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         *smtpHostPtr,
	}
	conn, err := tls.Dial("tcp", hostWithPort, tlsCfg)
	if err != nil {
		panic(err)
	}
	c, err := smtp.NewClient(conn, *smtpHostPtr)
	if err != nil {
		panic(err)
	}
	if err = c.Auth(mailAuth); err != nil {
		panic(err)
	}

	// setup to and from
	if err = c.Mail(from.Address); err != nil {
		panic(err)
	}
	if err = c.Rcpt(to.Address); err != nil {
		panic(err)
	}

	fmt.Println("sending mail . . . boop beep beep")
	// setup mail body and send
	w, err := c.Data()
	if err != nil {
		panic(err)
	}
	_, err = w.Write([]byte(mailContent))
	if err != nil {
		panic(err)
	}
	err = w.Close()
	if err != nil {
		panic(err)
	}
	if err = c.Quit(); err != nil {
		panic(err)
	}
	fmt.Println("email send successfully, remember to check your spam . . . boop boop . . .")
}
