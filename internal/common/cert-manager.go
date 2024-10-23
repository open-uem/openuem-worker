package common

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/doncicuto/openuem_ent/certificate"
	"github.com/doncicuto/openuem_nats"
	"github.com/doncicuto/openuem_utils"
	"github.com/nats-io/nats.go"
	"golang.org/x/sys/windows/registry"
	"software.sslmate.com/src/go-pkcs12"
)

func (w *Worker) SubscribeToCertManagerWorkerQueues() error {
	_, err := w.NATSConnection.Subscribe("certificates.new", w.NewCertificateHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %s", err.Error())
		return err
	}
	log.Println("[INFO]: subscribed to queue certificates.new")

	_, err = w.NATSConnection.Subscribe("certificates.revoke", w.RevokeCertificateHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %s", err.Error())
		return err
	}
	log.Println("[INFO]: subscribed to queue certificates.revoke")

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

func (w *Worker) NewCertificateHandler(msg *nats.Msg) {

	// Read message
	cr := openuem_nats.CertificateRequest{}
	if err := json.Unmarshal(msg.Data, &cr); err != nil {
		log.Printf("[ERR]: could not unmarshall new certificate request, reason: %s", err.Error())
		return
	}
	w.CertRequest = &cr

	if err := w.GenerateUserCertificate(); err != nil {
		log.Printf("[ERR]: could not generate the user certificate, reason: %s", err.Error())
		return
	}

	if err := w.SendNotification(); err != nil {
		log.Printf("[ERR]: could not send the user certificate, reason: %s", err.Error())
		return
	}

	certDescription := w.CertRequest.Username + " client certificate"
	if err := w.Model.SaveCertificate(w.UserCert.SerialNumber.Int64(), certificate.Type("user"), w.CertRequest.Username, certDescription, w.UserCert.NotAfter); err != nil {
		log.Println("[ERR]: error saving certificate status", err.Error())
		return
	}

	if err := w.Model.SetCertificateSent(w.CertRequest.Username); err != nil {
		log.Println("[ERR]: error saving certificate status", err.Error())
		return
	}

	if err := msg.Respond([]byte("New certificate has been processed")); err != nil {
		log.Println("[ERR]: could not sent response", err.Error())
		return
	}
}

func (w *Worker) RevokeCertificateHandler(msg *nats.Msg) {
	if err := msg.Respond([]byte("Certificate has been revoked!")); err != nil {
		log.Println("[ERR]: could not send response", err.Error())
		return
	}
}

func (w *Worker) GenerateCertManagerWorkerConfig() error {
	var err error

	w.DBUrl, err = openuem_utils.CreatePostgresDatabaseURL()
	if err != nil {
		log.Printf("[ERROR]: %v", err)
		return err
	}

	cwd, err := GetWd()
	if err != nil {
		log.Println("[ERROR]: could not get working directory")
		return err
	}

	k, err := openuem_utils.OpenRegistryForQuery(registry.LOCAL_MACHINE, `SOFTWARE\OpenUEM\Server`)
	if err != nil {
		log.Println("[ERROR]: could not open registry")
		return err
	}
	defer k.Close()

	w.ClientCertPath = filepath.Join(cwd, "certificates", "cert-manager-worker", "worker.cer")
	w.ClientKeyPath = filepath.Join(cwd, "certificates", "cert-manager-worker", "worker.key")
	w.CACertPath = filepath.Join(cwd, "certificates", "ca", "ca.cer")
	w.CAKeyPath = filepath.Join(cwd, "certificates", "ca", "ca.key")

	w.NATSServers, err = openuem_utils.GetValueFromRegistry(k, "NATSServers")
	if err != nil {
		log.Println("[ERROR]: could not read NATS servers from registry")
		return err
	}

	// get ocsp servers
	ocspServers := []string{}
	servers, err := openuem_utils.GetValueFromRegistry(k, "OCSPResponders")
	if err != nil {
		log.Println("[ERROR]: could not read OCSP responders from registry")
		return err
	}

	for _, ocsp := range strings.Split(servers, ",") {
		ocspServers = append(ocspServers, strings.TrimSpace(ocsp))
	}
	w.OCSPResponders = ocspServers

	// read required certificates and private keys
	w.CACert, err = openuem_utils.ReadPEMCertificate(w.CACertPath)
	if err != nil {
		log.Println("[ERROR]: could not read CA cert file")
		return err
	}

	w.CAPrivateKey, err = openuem_utils.ReadPEMPrivateKey(w.CAKeyPath)
	if err != nil {
		log.Println("[ERROR]: could not read CA private key file")
		return err
	}

	return nil
}