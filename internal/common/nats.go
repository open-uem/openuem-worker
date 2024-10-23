package common

import (
	"log"
	"time"

	"github.com/doncicuto/openuem_nats"
	"github.com/go-co-op/gocron/v2"
)

func (w *Worker) StartNATSConnectJob(queueSubscribe func() error) error {
	var err error

	w.NATSConnection, err = openuem_nats.ConnectWithNATS(w.NATSServers, w.ClientCertPath, w.ClientKeyPath, w.CACertPath)
	if err == nil {
		if err := queueSubscribe(); err == nil {
			return err
		}
		return nil
	}
	log.Printf("[ERROR]: could not connect to NATS %v", err)

	// Create task for running the agent
	w.NATSConnectJob, err = w.TaskScheduler.NewJob(
		gocron.DurationJob(
			time.Duration(time.Duration(2*time.Minute)),
		),
		gocron.NewTask(
			func() {
				w.NATSConnection, err = openuem_nats.ConnectWithNATS(w.NATSServers, w.ClientCertPath, w.ClientKeyPath, w.CACertPath)
				if err != nil {
					log.Printf("[ERROR]: could not connect to NATS %v", err)
					return
				}

				if err := w.TaskScheduler.RemoveJob(w.NATSConnectJob.ID()); err != nil {
					return
				}

				if err := queueSubscribe(); err != nil {
					return
				}
			},
		),
	)
	if err != nil {
		log.Fatalf("[FATAL]: could not start the NATS connect job: %v", err)
		return err
	}
	log.Printf("[INFO]: new NATS connect job has been scheduled every %d minutes", 2)
	return nil
}
