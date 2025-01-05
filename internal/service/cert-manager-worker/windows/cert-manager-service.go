//go:build windows

package main

import (
	"log"

	"github.com/open-uem/openuem-worker/internal/common"
	"github.com/open-uem/utils"
	"golang.org/x/sys/windows/svc"
)

func main() {
	w := common.NewWorker("openuem-cert-manager-worker.txt")
	s := utils.NewOpenUEMWindowsService()

	// Get config for service
	if err := w.GenerateCertManagerWorkerConfig(); err != nil {
		log.Printf("[ERROR]: could not generate config for cert-manager worker: %v", err)
	}

	s.ServiceStart = func() { w.StartWorker(w.SubscribeToCertManagerWorkerQueues) }
	s.ServiceStop = w.StopWorker

	// Run service
	err := svc.Run("openuem-cert-manager-worker", s)
	if err != nil {
		log.Printf("[ERROR]: could not run service: %v", err)
	}
}
