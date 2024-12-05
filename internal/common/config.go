package common

import (
	"log"
	"strings"

	"github.com/doncicuto/openuem_ent/component"
	"github.com/doncicuto/openuem_utils"
	"gopkg.in/ini.v1"
)

func (w *Worker) GenerateCommonWorkerConfig(c string) error {
	var err error

	// Get conf file
	configFile := openuem_utils.GetConfigFile()

	// Open ini file
	cfg, err := ini.Load(configFile)
	if err != nil {
		return err
	}

	w.DBUrl, err = openuem_utils.CreatePostgresDatabaseURL()
	if err != nil {
		log.Printf("[ERROR]: %v", err)
		return err
	}

	key, err := cfg.Section("Certificates").GetKey("CACert")
	if err != nil {
		log.Printf("[ERROR]: could not get CA cert path, reason: %v\n", err)
		return err
	}
	w.CACertPath = key.String()

	certKey := ""
	privateKey := ""
	switch c {
	case component.ComponentAgentWorker.String():
		certKey = "AgentWorkerCert"
		privateKey = "AgentWorkerKey"
	case component.ComponentCertManagerWorker.String():
		certKey = "CertManagerWorkerCert"
		privateKey = "CertManagerWorkerKey"

	case component.ComponentNotificationWorker.String():
		certKey = "NotificationWorkerCert"
		privateKey = "NotificationWorkerKey"
	}

	key, err = cfg.Section("Certificates").GetKey(certKey)
	if err != nil {
		log.Printf("[ERROR]: could not get Worker cert path, reason: %v\n", err)
		return err
	}
	w.ClientCertPath = key.String()

	key, err = cfg.Section("Certificates").GetKey(privateKey)
	if err != nil {
		log.Printf("[ERROR]: could not get Worker key path, reason: %v\n", err)
		return err
	}
	w.ClientKeyPath = key.String()

	key, err = cfg.Section("NATS").GetKey("NATSServer")
	if err != nil {
		log.Println("[ERROR]: could not get NATS servers urls")
		return err
	}
	w.NATSServers = key.String()

	return nil
}

func (w *Worker) GenerateCertManagerWorkerConfig() error {
	var err error

	// Get conf file
	configFile := openuem_utils.GetConfigFile()

	// Open ini file
	cfg, err := ini.Load(configFile)
	if err != nil {
		return err
	}

	if err := w.GenerateCommonWorkerConfig(component.ComponentCertManagerWorker.String()); err != nil {
		return err
	}

	key, err := cfg.Section("Server").GetKey("ca_key_path")
	if err != nil {
		log.Println("[ERROR]: could not get CA key path")
		return err
	}
	w.CAKeyPath = key.String()

	key, err = cfg.Section("Server").GetKey("ocsp_url")
	if err != nil {
		log.Println("[ERROR]: could not get OCSP Responder url")
		return err
	}
	ocspServers := []string{}
	servers := key.String()
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
