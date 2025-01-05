//go:build windows

package main

import (
	"log"

	"github.com/open-uem/openuem-worker/internal/common"
	"github.com/open-uem/openuem_utils"
	"golang.org/x/sys/windows/svc"
)

func main() {
	w := common.NewWorker("openuem-notification-worker.txt")
	s := openuem_utils.NewOpenUEMWindowsService()

	// Get config for service
	if err := w.GenerateCommonWorkerConfig("notification-worker"); err != nil {
		log.Printf("[ERROR]: could not generate config for notification worker: %v", err)
	}

	s.ServiceStart = func() { w.StartWorker(w.SubscribeToNotificationWorkerQueues) }
	s.ServiceStop = w.StopWorker

	// Run service
	err := svc.Run("openuem-notification-worker", s)
	if err != nil {
		log.Printf("[ERROR]: could not run service: %v", err)
	}
}
