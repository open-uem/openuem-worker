package common

import (
	"log"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/open-uem/utils"
	"gopkg.in/ini.v1"
)

func (w *Worker) GenerateCommonWorkerConfig(c string) error {
	var err error

	// Get conf file
	configFile := utils.GetConfigFile()

	// Open ini file
	cfg, err := ini.Load(configFile)
	if err != nil {
		return err
	}

	w.DBUrl, err = utils.CreatePostgresDatabaseURL()
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
	case "agent-worker":
		certKey = "AgentWorkerCert"
		privateKey = "AgentWorkerKey"
	case "cert-manager-worker":
		certKey = "CertManagerWorkerCert"
		privateKey = "CertManagerWorkerKey"
	case "notification-worker":
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

	key, err = cfg.Section("NATS").GetKey("NATSServers")
	if err != nil {
		log.Println("[ERROR]: could not get NATS servers urls")
		return err
	}
	w.NATSServers = key.String()

	w.Replicas = len(strings.Split(w.NATSServers, ","))

	return nil
}

func (w *Worker) GenerateCertManagerWorkerConfig() error {
	var err error

	// Get conf file
	configFile := utils.GetConfigFile()

	// Open ini file
	cfg, err := ini.Load(configFile)
	if err != nil {
		return err
	}

	if err := w.GenerateCommonWorkerConfig("cert-manager-worker"); err != nil {
		return err
	}

	key, err := cfg.Section("Certificates").GetKey("CAKey")
	if err != nil {
		log.Println("[ERROR]: could not get CA key path")
		return err
	}
	w.CAKeyPath = key.String()

	key, err = cfg.Section("Certificates").GetKey("OCSPUrls")
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
	w.CACert, err = utils.ReadPEMCertificate(w.CACertPath)
	if err != nil {
		log.Println("[ERROR]: could not read CA cert file")
		return err
	}

	w.CAPrivateKey, err = utils.ReadPEMPrivateKey(w.CAKeyPath)
	if err != nil {
		log.Println("[ERROR]: could not read CA private key file")
		return err
	}

	return nil
}

func (w *Worker) StartGenerateWorkerConfigJob(workerName string, common bool) error {
	var err error

	// Create task for getting the worker config
	w.ConfigJob, err = w.TaskScheduler.NewJob(
		gocron.DurationJob(
			time.Duration(time.Duration(1*time.Minute)),
		),
		gocron.NewTask(
			func() {
				if common {
					err = w.GenerateCommonWorkerConfig(workerName)
				} else {
					err = w.GenerateCertManagerWorkerConfig()
				}
				if err != nil {
					log.Printf("[ERROR]: could not generate config for worker, reason: %v", err)
					return
				}

				log.Println("[INFO]: worker's config has been successfully generated")
				if err := w.TaskScheduler.RemoveJob(w.ConfigJob.ID()); err != nil {
					return
				}
			},
		),
	)
	if err != nil {
		log.Fatalf("[FATAL]: could not start the generate worker config job: %v", err)
		return err
	}
	log.Printf("[INFO]: new generate worker config job has been scheduled every %d minute", 1)
	return nil
}
