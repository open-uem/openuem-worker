package commands

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/doncicuto/openuem_ent/certificate"
	"github.com/doncicuto/openuem_nats"
	"github.com/doncicuto/openuem_utils"
	"github.com/nats-io/nats.go"
	"github.com/urfave/cli/v2"
	"software.sslmate.com/src/go-pkcs12"
)

func CertManagerWorker() *cli.Command {
	return &cli.Command{
		Name:  "cert-manager",
		Usage: "Manage OpenUEM's Cert-Manager worker",
		Subcommands: []*cli.Command{
			{
				Name:   "start",
				Usage:  "Start an OpenUEM's Cert-Manager worker",
				Action: startCertManagerWorker,
				Flags:  StartCertManagerWorkerFlags(),
			},
			{
				Name:   "stop",
				Usage:  "Stop an OpenUEM's Cert-Manager worker",
				Action: stopWorker,
			},
		},
	}
}

func StartCertManagerWorkerFlags() []cli.Flag {
	flags := CommonFlags()

	flags = append(flags, &cli.StringFlag{
		Name:     "ocsp",
		Usage:    "the url of the OCSP responder, e.g https://ocsp.example.com",
		EnvVars:  []string{"OCSP"},
		Required: true,
	})

	return append(flags, &cli.StringFlag{
		Name:    "cakey",
		Value:   "certificates/ca.key",
		Usage:   "the path to your CA private key file in PEM format",
		EnvVars: []string{"CA_KEY_FILENAME"},
	})
}

func startCertManagerWorker(cCtx *cli.Context) error {
	command := WorkerCommand{}

	// Specific requisites
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	log.Printf("📜  reading your CA private key PEM file")
	caKeyPath := filepath.Join(cwd, cCtx.String("cakey"))
	command.CAPrivateKey, err = openuem_utils.ReadPEMPrivateKey(caKeyPath)
	if err != nil {
		return err
	}

	// get ocsp servers
	ocspServers := []string{}
	for _, ocsp := range strings.Split(cCtx.String("ocsp"), ",") {
		ocspServers = append(ocspServers, strings.TrimSpace(ocsp))
	}
	command.OCSPResponders = ocspServers

	if err := command.checkCommonRequisites(cCtx); err != nil {
		return err
	}

	if err := command.connectToNATS(cCtx); err != nil {
		return err
	}

	log.Println("📩  subscribing to cert-manager messages")
	// TODO do something with subscription?
	if _, err := command.MessageServer.Connection.Subscribe("certificates.new", command.newCertificateHandler); err != nil {
		return err
	}

	if _, err := command.MessageServer.Connection.Subscribe("certificates.revoke", command.revokeCertificateHandler); err != nil {
		return err
	}

	// Keep the connection alive
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Printf("✅ Done! Your Cert Manager worker is ready\n\n")
	<-done

	command.MessageServer.Close()
	log.Printf("👋 Done! Your Cert Manager has been shutdown\n\n")
	return nil
}

func (command *WorkerCommand) newCertificateHandler(msg *nats.Msg) {
	// Read message
	cr := openuem_nats.CertificateRequest{}
	if err := json.Unmarshal(msg.Data, &cr); err != nil {
		log.Printf("[ERR]: could not unmarshall new certificate request, reason: %s", err.Error())
		return
	}

	command.CertRequest = &cr

	if err := command.generateUserCertificate(); err != nil {
		log.Printf("[ERR]: could not generate the user certificate, reason: %s", err.Error())
		return
	}

	if err := command.sendNotification(); err != nil {
		log.Printf("[ERR]: could not send the user certificate, reason: %s", err.Error())
		return
	}

	certDescription := command.CertRequest.Username + " client certificate"
	if err := command.Model.SaveCertificate(command.UserCert.SerialNumber.Int64(), certificate.Type("user"), command.CertRequest.Username, certDescription, command.UserCert.NotAfter); err != nil {
		log.Println("[ERR]: error saving certificate status", err.Error())
		return
	}

	if err := command.Model.SetCertificateSent(command.CertRequest.Username); err != nil {
		log.Println("[ERR]: error saving certificate status", err.Error())
		return
	}

	if err := msg.Respond([]byte("New certificate has been processed")); err != nil {
		log.Println("[ERR]: could not sent response", err.Error())
		return
	}
}

func (command *WorkerCommand) generateUserCertificate() error {
	var err error
	template, err := command.newX509UserCertificateTemplate()
	if err != nil {
		return err
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, command.CACert, &certPrivKey.PublicKey, command.CAPrivateKey)
	if err != nil {
		return err
	}

	command.UserCert, err = x509.ParseCertificate(certBytes)
	if err != nil {
		return err
	}

	password := command.CertRequest.Password
	if password == "" {
		password = pkcs12.DefaultPassword
	}
	pfxBytes, err := pkcs12.Modern.Encode(certPrivKey, command.UserCert, []*x509.Certificate{command.CACert}, password)
	if err != nil {
		return err
	}

	command.PKCS12 = pfxBytes
	return nil
}

func (command *WorkerCommand) newX509UserCertificateTemplate() (*x509.Certificate, error) {
	serialNumber, err := openuem_utils.GenerateSerialNumber()
	if err != nil {
		return nil, err
	}

	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:    command.CertRequest.Username,
			Organization:  []string{command.CertRequest.Organization},
			Country:       []string{command.CertRequest.Country},
			Province:      []string{command.CertRequest.Province},
			Locality:      []string{command.CertRequest.Locality},
			StreetAddress: []string{command.CertRequest.Address},
			PostalCode:    []string{command.CertRequest.PostalCode},
		},
		Issuer:      command.CACert.Subject,
		NotBefore:   time.Now().Add(-5 * time.Minute).UTC(),
		NotAfter:    time.Now().AddDate(command.CertRequest.YearsValid, command.CertRequest.MonthsValid, command.CertRequest.DaysValid),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		OCSPServer:  command.OCSPResponders,
	}, nil
}

func (command *WorkerCommand) revokeCertificateHandler(msg *nats.Msg) {
	if err := msg.Respond([]byte("Certificate has been revoked!")); err != nil {
		log.Println("[ERR]: could not send response", err.Error())
		return
	}
}

func (command *WorkerCommand) sendNotification() error {
	notification := openuem_nats.Notification{
		To:                    command.CertRequest.Email,
		Subject:               "Your certificate to log in to OpenUEM console",
		MessageTitle:          "OpenUEM | Your certificate",
		MessageText:           "You can find attached the digital certificate that you must import to your browser so you can use it to log in to the OpenUEM console",
		MessageGreeting:       fmt.Sprintf("Hi %s", command.CertRequest.FullName),
		MessageAttachFileName: command.CertRequest.Username + ".pfx",
		MessageAttachFile:     base64.StdEncoding.EncodeToString(command.PKCS12),
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	if err := command.MessageServer.Connection.Publish("notification.send_certificate", data); err != nil {
		return err
	}

	return nil
}
