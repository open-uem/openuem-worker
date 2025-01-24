//go:build linux

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-co-op/gocron/v2"
	"github.com/open-uem/openuem-worker/internal/common"
)

func main() {
	var err error
	w := common.NewWorker("openuem-cert-manager-worker")

	// Start Task Scheduler
	w.TaskScheduler, err = gocron.NewScheduler()
	if err != nil {
		log.Fatalf("[FATAL]: could not create task scheduler, reason: %v", err)
		return
	}
	w.TaskScheduler.Start()
	log.Println("[INFO]: task scheduler has been started")

	// Get config for service
	if err := w.GenerateCertManagerWorkerConfig(); err != nil {
		log.Printf("[ERROR]: could not generate config for cert-manager worker: %v", err)
		if err := w.StartGenerateWorkerConfigJob("cert-manager-worker", true); err != nil {
			log.Fatalf("[FATAL]: could not start generate config for worker: %v", err)
			return
		}

	}

	w.StartWorker(w.SubscribeToCertManagerWorkerQueues)

	// Keep the connection alive
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("[INFO]: the cert-manager worker is ready and waiting for requests")
	<-done

	w.StopWorker()
}
