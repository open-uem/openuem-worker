//go:build linux

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-uem/openuem-worker/internal/common"
)

func main() {
	w := common.NewWorker("openuem-cert-manager-worker")

	// Get config for service
	if err := w.GenerateCertManagerWorkerConfig(); err != nil {
		log.Printf("[ERROR]: could not generate config for cert-manager worker: %v", err)
	}

	w.StartWorker(w.SubscribeToCertManagerWorkerQueues)

	// Keep the connection alive
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("[INFO]: the cert-manager worker is ready and waiting for requests")
	<-done

	w.StopWorker()
}
