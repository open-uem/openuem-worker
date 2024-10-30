package common

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"log"

	"github.com/doncicuto/openuem-worker/internal/models"
	"github.com/doncicuto/openuem_ent"
	"github.com/doncicuto/openuem_nats"
	"github.com/doncicuto/openuem_utils"
	"github.com/go-co-op/gocron/v2"
	"github.com/nats-io/nats.go"
)

type Worker struct {
	NATSConnection         *nats.Conn
	NATSConnectJob         gocron.Job
	NATSServers            string
	DatabaseType           string
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
	UserCert               *x509.Certificate
	CertRequest            *openuem_nats.CertificateRequest
	Settings               *openuem_ent.Settings
	Logger                 *openuem_utils.OpenUEMLogger
	ConsoleURL             string
	OCSPResponders         []string
	JetstreamContextCancel context.CancelFunc
}

func NewWorker(logName string) *Worker {
	worker := Worker{}
	if logName != "" {
		worker.Logger = openuem_utils.NewLogger(logName)
	}
	return &worker
}

func (w *Worker) StartWorker(subscription func() error) {
	var err error

	// Start Task Scheduler
	w.TaskScheduler, err = gocron.NewScheduler()
	if err != nil {
		log.Printf("[ERROR]: could not create task scheduler, reason: %s", err.Error())
		return
	}
	w.TaskScheduler.Start()
	log.Println("[INFO]: task scheduler has been started")

	// Start a job to try to connect with the database
	if err := w.StartDBConnectJob(); err != nil {
		log.Printf("[ERROR]: could not start DB connect job, reason: %s", err.Error())
		return
	}

	// Start a job to try to connect with NATS
	if err := w.StartNATSConnectJob(subscription); err != nil {
		log.Printf("[ERROR]: could not start NATS connect job, reason: %s", err.Error())
		return
	}
}

func (w *Worker) StopWorker() {
	if err := w.NATSConnection.Drain(); err != nil {
		log.Printf("[ERROR]: could not drain NATS connection, reason: %s", err.Error())
	}
	w.Model.Close()
	w.Logger.Close()
	if err := w.TaskScheduler.Shutdown(); err != nil {
		log.Printf("[ERROR]: could not stop the task scheduler, reason: %s", err.Error())
	}
	if w.JetstreamContextCancel != nil {
		w.JetstreamContextCancel()
	}
}
