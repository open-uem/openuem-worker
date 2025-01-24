//go:build windows

package main

import (
	"log"

	"github.com/go-co-op/gocron/v2"
	"github.com/open-uem/openuem-worker/internal/common"
	"github.com/open-uem/utils"
	"golang.org/x/sys/windows/svc"
)

func main() {
	var err error
	w := common.NewWorker("openuem-cert-manager-worker.txt")
	s := utils.NewOpenUEMWindowsService()

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

	s.ServiceStart = func() { w.StartWorker(w.SubscribeToCertManagerWorkerQueues) }
	s.ServiceStop = w.StopWorker

	// Run service
	if err := svc.Run("openuem-cert-manager-worker", s); err != nil {
		log.Printf("[ERROR]: could not run service: %v", err)
	}
}
