package common

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"log"

	"github.com/go-co-op/gocron/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/open-uem/ent"
	"github.com/open-uem/ent/server"
	openuem_nats "github.com/open-uem/nats"
	"github.com/open-uem/openuem-worker/internal/models"
	"github.com/open-uem/utils"
)

type Worker struct {
	NATSConnection         *nats.Conn
	NATSConnectJob         gocron.Job
	NATSServers            string
	DBUrl                  string
	DBConnectJob           gocron.Job
	ConfigJob              gocron.Job
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
	Settings               *ent.Settings
	Logger                 *utils.OpenUEMLogger
	ConsoleURL             string
	OCSPResponders         []string
	JetstreamContextCancel context.CancelFunc
	Version                string
	Channel                server.Channel
	Replicas               int
	Jetstream              jetstream.JetStream
}

func NewWorker(logName string) *Worker {
	worker := Worker{}
	if logName != "" {
		worker.Logger = utils.NewLogger(logName)
	}

	return &worker
}

func (w *Worker) StartWorker(subscription func() error) {
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

	remoteConfigRequest := openuem_nats.RemoteConfigRequest{}
	err := json.Unmarshal(msg.Data, &remoteConfigRequest)
	if err != nil {
		remoteConfigRequest.AgentID = string(msg.Data)
	}

	frequency, err := w.Model.GetDefaultAgentFrequency(remoteConfigRequest)
	if err != nil {
		log.Printf("[ERROR]: could not get default frequency, reason: %v", err)
		config.Ok = false
	} else {
		config.AgentFrequency = frequency
		config.Ok = true
	}

	wingetFrequency, err := w.Model.GetWingetFrequency(remoteConfigRequest)
	if err != nil {
		log.Printf("[ERROR]: could not get winget frequency, reason: %v", err)
		config.Ok = false
	} else {
		config.WinGetFrequency = wingetFrequency
		config.Ok = true
	}

	sftpStatus, err := w.Model.GetSFTPAgentSetting(remoteConfigRequest)
	if err != nil {
		log.Printf("[ERROR]: could not get SFTP service for agent, reason: %v", err)
		config.Ok = false
	} else {
		config.SFTPDisabled = !sftpStatus
		config.Ok = true
		if err := w.Model.SaveSFTPAgentSetting(remoteConfigRequest, sftpStatus); err != nil {
			log.Printf("[ERROR]: could not save Agent SFTP status, reason: %v", err)
		}
	}

	remoteAssistance, err := w.Model.GetRemoteAssistanceAgentSetting(remoteConfigRequest)
	if err != nil {
		log.Printf("[ERROR]: could not get Remote Assistance for agent, reason: %v", err)
		config.Ok = false
	} else {
		config.RemoteAssistanceDisabled = !remoteAssistance
		config.Ok = true
		if err := w.Model.SaveRemoteAssistanceAgentSetting(remoteConfigRequest, remoteAssistance); err != nil {
			log.Printf("[ERROR]: could not save Agent Remote Assistance status, reason: %v", err)
		}
	}

	data, err := json.Marshal(config)
	if err != nil {
		log.Printf("[ERROR]: could not marshal config data, reason: %v", err)
		return
	}

	if err := msg.Respond(data); err != nil {
		log.Printf("[ERROR]: could not respond with agent config, reason: %v", err)
	}

}
