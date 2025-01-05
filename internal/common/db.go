package common

import (
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/open-uem/openuem-worker/internal/models"
)

func (w *Worker) StartDBConnectJob(subscription func() error) error {
	var err error

	w.Model, err = models.New(w.DBUrl)
	if err == nil {
		log.Println("[INFO]: connection established with database")

		// Start a job to try to connect with NATS
		if err := w.StartNATSConnectJob(subscription); err != nil {
			log.Fatalf("[FATAL]: could not start NATS connect job, reason: %v", err)
		}
		return nil
	}
	log.Printf("[ERROR]: could not connect with database %v", err)

	// Create task for running the agent
	w.DBConnectJob, err = w.TaskScheduler.NewJob(
		gocron.DurationJob(
			time.Duration(time.Duration(2*time.Minute)),
		),
		gocron.NewTask(
			func() {
				w.Model, err = models.New(w.DBUrl)
				if err != nil {
					log.Printf("[ERROR]: could not connect with database %v", err)
					return
				}
				log.Println("[INFO]: connection established with database")

				if err := w.TaskScheduler.RemoveJob(w.DBConnectJob.ID()); err != nil {
					return
				}

				if err := w.StartNATSConnectJob(subscription); err != nil {
					log.Fatalf("[FATAL]: could not start NATS connect job, reason: %v", err)
				}
				return
			},
		),
	)
	if err != nil {
		log.Fatalf("[FATAL]: could not start the DB connect job: %v", err)
		return err
	}
	log.Printf("[INFO]: new DB connect job has been scheduled every %d minutes", 2)
	return nil
}
