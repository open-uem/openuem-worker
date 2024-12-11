package common

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"log"

	"github.com/doncicuto/openuem-worker/internal/models"
	"github.com/doncicuto/openuem_ent"
	"github.com/doncicuto/openuem_ent/server"
	"github.com/doncicuto/openuem_nats"
	"github.com/doncicuto/openuem_utils"
	"github.com/go-co-op/gocron/v2"
	"github.com/nats-io/nats.go"
)

type Worker struct {
	NATSConnection         *nats.Conn
	NATSConnectJob         gocron.Job
	NATSServers            string
	DBUrl                  string
	DBConnectJob           gocron.Job
	TaskScheduler          gocron.Scheduler
	Model                  *models.Model
	CACert                 *x509.Certificate
	CAPrivateKey           *rsa.PrivateKey
	ClientCertPath         string
	ClientKeyPath          string
	CACertPath             string
	CAKeyPath              string
	PKCS12                 []byte
	Cert                   *x509.Certificate
	CertBytes              []byte
	PrivateKey             *rsa.PrivateKey
	CertRequest            *openuem_nats.CertificateRequest
	Settings               *openuem_ent.Settings
	Logger                 *openuem_utils.OpenUEMLogger
	ConsoleURL             string
	OCSPResponders         []string
	JetstreamContextCancel context.CancelFunc
	Version                string
	Channel                server.Channel
}

func NewWorker(logName string) *Worker {
	worker := Worker{}
	if logName != "" {
		worker.Logger = openuem_utils.NewLogger(logName)
	}

	worker.Version = VERSION
	worker.Channel = CHANNEL
	return &worker
}

func (w *Worker) StartWorker(subscription func() error) {
	var err error

	// Start Task Scheduler
	w.TaskScheduler, err = gocron.NewScheduler()
	if err != nil {
		log.Fatalf("[FATAL]: could not create task scheduler, reason: %v", err)
		return
	}
	w.TaskScheduler.Start()
	log.Println("[INFO]: task scheduler has been started")

	// Start a job to try to connect with the database
	if err := w.StartDBConnectJob(subscription); err != nil {
		log.Fatalf("[FATAL]: could not start DB connect job, reason: %v", err)
		return
	}
}

func (w *Worker) StopWorker() {
	if w.NATSConnection != nil {
		if err := w.NATSConnection.Drain(); err != nil {
			log.Printf("[ERROR]: could not drain NATS connection, reason: %v", err)
		}
		if w.JetstreamContextCancel != nil {
			w.JetstreamContextCancel()
		}
	}

	if w.Model != nil {
		w.Model.Close()
	}

	if w.TaskScheduler != nil {
		if err := w.TaskScheduler.Shutdown(); err != nil {
			log.Printf("[ERROR]: could not stop the task scheduler, reason: %v", err)
		}
	}

	log.Println("[INFO]: the worker has stopped")

	if w.Logger != nil {
		w.Logger.Close()
	}
}

func (w *Worker) PingHandler(msg *nats.Msg) {
	if err := msg.Respond(nil); err != nil {
		log.Printf("[ERROR]: could not respond to ping message, reason: %v", err)
	}
}

func (w *Worker) AgentConfigHandler(msg *nats.Msg) {
	config := openuem_nats.Config{}

	frequency, err := w.Model.GetDefaultAgentFrequency()
	if err != nil {
		log.Printf("[ERROR]: could not get default frequency, reason: %v", err)
		config.Ok = false
	} else {
		config.AgentFrequency = frequency
		config.Ok = true
	}

	data, err := json.Marshal(config)
	if err != nil {
		log.Printf("[ERROR]: could not marshal config data, reason: %v", err)
		return
	}

	if err := msg.Respond(data); err != nil {
		log.Printf("[ERROR]: could not respond with agent config, reason: %v", err)
	}
	return
}
