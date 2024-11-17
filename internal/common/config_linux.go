//go:build linux

package common

import (
	"log"
	"strings"

	"github.com/doncicuto/openuem_utils"
	"gopkg.in/ini.v1"
)

func (w *Worker) GenerateCommonWorkerConfig(configPath string) error {
	var err error

	// Open ini file
	cfg, err := ini.Load("/etc/openuem-server/openuem.ini")
	if err != nil {
		return err
	}

	w.DBUrl, err = openuem_utils.CreatePostgresDatabaseURL()
	if err != nil {
		log.Printf("[ERROR]: %v", err)
		return err
	}

	key, err := cfg.Section("Server").GetKey("ca_cert_path")
	if err != nil {
		log.Printf("[ERROR]: could not get CA cert path %s\n", configPath+"ca_cert_path")
		return err
	}
	w.CACertPath = key.String()

	key, err = cfg.Section("Server").GetKey(configPath + "_worker_cert_path")
	if err != nil {
		log.Printf("[ERROR]: could not get Worker cert path %s\n", configPath+"_worker_cert_path")
		return err
	}
	w.ClientCertPath = key.String()

	key, err = cfg.Section("Server").GetKey(configPath + "_worker_key_path")
	if err != nil {
		log.Printf("[ERROR]: could not get Worker key path %s\n", configPath+"_worker_key_path")
		return err
	}
	w.ClientKeyPath = key.String()

	key, err = cfg.Section("Server").GetKey("nats_url")
	if err != nil {
		log.Println("[ERROR]: could not get NATS url")
		return err
	}
	w.NATSServers = key.String()

	return nil
}

func (w *Worker) GenerateCertManagerWorkerConfig() error {
	var err error

	// Open ini file
	cfg, err := ini.Load("/etc/openuem-server/openuem.ini")
	if err != nil {
		return err
	}

	if err := w.GenerateCommonWorkerConfig("cert_manager"); err != nil {
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
