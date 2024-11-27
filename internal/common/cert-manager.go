package common

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/doncicuto/openuem_ent/certificate"
	"github.com/doncicuto/openuem_nats"
	"github.com/doncicuto/openuem_utils"
	"github.com/nats-io/nats.go"
	"software.sslmate.com/src/go-pkcs12"
)

func (w *Worker) SubscribeToCertManagerWorkerQueues() error {
	_, err := w.NATSConnection.QueueSubscribe("certificates.user", "openuem-cert-manager", w.NewUserCertificateHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %v", err)
		return err
	}
	log.Println("[INFO]: subscribed to queue certificates.user")

	_, err = w.NATSConnection.QueueSubscribe("certificates.revoke", "openuem-cert-manager", w.RevokeCertificateHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %v", err)
		return err
	}
	log.Println("[INFO]: subscribed to queue certificates.revoke")

	_, err = w.NATSConnection.QueueSubscribe("certificates.agent.*", "openuem-cert-manager", w.NewAgentCertificateHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to queue certificates.agent")

	_, err = w.NATSConnection.QueueSubscribe("ping.certmanagerworker", "openuem-cert-manager", w.PingHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to queue ping")
	return nil
}

func (w *Worker) GenerateUserCertificate() error {
	var err error
	template, err := w.NewX509UserCertificateTemplate()
	if err != nil {
		return err
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, w.CACert, &certPrivKey.PublicKey, w.CAPrivateKey)
	if err != nil {
		return err
	}

	w.UserCert, err = x509.ParseCertificate(certBytes)
	if err != nil {
		return err
	}

	password := w.CertRequest.Password
	if password == "" {
		password = pkcs12.DefaultPassword
	}

	w.PKCS12, err = pkcs12.Modern.Encode(certPrivKey, w.UserCert, []*x509.Certificate{w.CACert}, password)
	if err != nil {
		return err
	}

	return nil
}

func (w *Worker) NewX509UserCertificateTemplate() (*x509.Certificate, error) {
	serialNumber, err := openuem_utils.GenerateSerialNumber()
	if err != nil {
		return nil, err
	}

	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:    w.CertRequest.Username,
			Organization:  []string{w.CertRequest.Organization},
			Country:       []string{w.CertRequest.Country},
			Province:      []string{w.CertRequest.Province},
			Locality:      []string{w.CertRequest.Locality},
			StreetAddress: []string{w.CertRequest.Address},
			PostalCode:    []string{w.CertRequest.PostalCode},
		},
		Issuer:      w.CACert.Subject,
		NotBefore:   time.Now().Add(-5 * time.Minute).UTC(),
		NotAfter:    time.Now().AddDate(w.CertRequest.YearsValid, w.CertRequest.MonthsValid, w.CertRequest.DaysValid),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		OCSPServer:  w.OCSPResponders,
	}, nil
}

func (w *Worker) NewUserCertificateHandler(msg *nats.Msg) {

	// Read message
	cr := openuem_nats.CertificateRequest{}
	if err := json.Unmarshal(msg.Data, &cr); err != nil {
		log.Printf("[ERROR]: could not unmarshall new certificate request, reason: %v", err)
		msg.NakWithDelay(5 * time.Minute)
		return
	}
	w.CertRequest = &cr

	if err := w.GenerateUserCertificate(); err != nil {
		log.Printf("[ERROR]: could not generate the user certificate, reason: %v", err)
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	if err := w.SendCertificate(); err != nil {
		log.Printf("[ERROR]: could not send the user certificate, reason: %v", err)
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	certDescription := w.CertRequest.Username + " client certificate"
	if err := w.Model.SaveCertificate(w.UserCert.SerialNumber.Int64(), certificate.Type("user"), w.CertRequest.Username, certDescription, w.UserCert.NotAfter); err != nil {
		log.Println("[ERROR]: error saving certificate status", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	if err := w.Model.SetCertificateSent(w.CertRequest.Username); err != nil {
		log.Println("[ERROR]: error saving certificate status", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	// If certificate has been sent we also set email as verified in case it wasn't (import users)
	if err := w.Model.SetEmailVerified(w.CertRequest.Username); err != nil {
		log.Println("[ERROR]: error saving certificate status", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	if err := msg.Respond([]byte("New certificate has been processed")); err != nil {
		log.Println("[ERROR]: could not sent response", err.Error())
		return
	}
}

func (w *Worker) NewAgentCertificateHandler(msg *nats.Msg) {

	// Read message
	subject := msg.Subject
	subjectTokens := strings.Split(subject, ".")

	if len(subjectTokens) != 3 {
		return
	}

	agentId := subjectTokens[2]

	log.Println("AgentID: ", agentId)

	/* cr := openuem_nats.CertificateRequest{}
	if err := json.Unmarshal(msg.Data, &cr); err != nil {
		log.Printf("[ERROR]: could not unmarshall new certificate request, reason: %v", err)
		msg.NakWithDelay(5 * time.Minute)
		return
	}
	w.CertRequest = &cr

	if err := w.GenerateUserCertificate(); err != nil {
		log.Printf("[ERROR]: could not generate the user certificate, reason: %v", err)
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	if err := w.SendCertificate(); err != nil {
		log.Printf("[ERROR]: could not send the user certificate, reason: %v", err)
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	certDescription := w.CertRequest.Username + " client certificate"
	if err := w.Model.SaveCertificate(w.UserCert.SerialNumber.Int64(), certificate.Type("user"), w.CertRequest.Username, certDescription, w.UserCert.NotAfter); err != nil {
		log.Println("[ERROR]: error saving certificate status", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	if err := w.Model.SetCertificateSent(w.CertRequest.Username); err != nil {
		log.Println("[ERROR]: error saving certificate status", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	// If certificate has been sent we also set email as verified in case it wasn't (import users)
	if err := w.Model.SetEmailVerified(w.CertRequest.Username); err != nil {
		log.Println("[ERROR]: error saving certificate status", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}
	*/
	if err := msg.Respond([]byte("")); err != nil {
		log.Println("[ERROR]: could not sent response", err.Error())
		return
	}
}

func (w *Worker) RevokeCertificateHandler(msg *nats.Msg) {
	if err := msg.Respond([]byte("Certificate has been revoked!")); err != nil {
		log.Println("[ERROR]: could not send response", err.Error())
		return
	}
}

func (w *Worker) SendCertificate() error {

	// Read the CA certificate file to attach it to the message
	caCert, err := os.ReadFile(w.CACertPath)
	if err != nil {
		return err
	}

	// ZIP the file as Outlook block the .cer, .crt extensions....
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	f, err := zw.Create("ca.cer")
	if err != nil {
		return err
	}
	_, err = f.Write(caCert)
	if err != nil {
		return err
	}

	if err := zw.Close(); err != nil {
		return err
	}

	notification := openuem_nats.Notification{
		To:           w.CertRequest.Email,
		Subject:      "Your certificate to log in to OpenUEM web console",
		MessageTitle: "OpenUEM | Your certificate",
		MessageText: `You can find attached the digital certificate in pfx format that you must import to your browser so you can use it to log in to the OpenUEM console. 
		
		<br/><br/>Also you may need to import the zipped ca.cer file as a trusted root certificate authority so your browser can trust in the certificates generated by OpenUEM CA`,
		MessageGreeting:        fmt.Sprintf("Hi %s", w.CertRequest.FullName),
		MessageAction:          "Go to console",
		MessageActionURL:       w.CertRequest.ConsoleURL,
		MessageAttachFileName:  w.CertRequest.Username + ".pfx",
		MessageAttachFile:      base64.StdEncoding.EncodeToString(w.PKCS12),
		MessageAttachFileName2: "ca_crt.zip",
		MessageAttachFile2:     base64.StdEncoding.EncodeToString(buf.Bytes()),
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	if err := w.NATSConnection.Publish("notification.send_certificate", data); err != nil {
		return err
	}

	return nil
}
