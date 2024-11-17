//go:build linux

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/doncicuto/openuem-worker/internal/common"
)

func main() {
	w := common.NewWorker("openuem-notification-worker")

	// Get config for service
	if err := w.GenerateCommonWorkerConfig("notification"); err != nil {
		log.Printf("[ERROR]: could not generate config for notification worker: %v", err)
	}

	w.StartWorker(w.SubscribeToNotificationWorkerQueues)

	// Keep the connection alive
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("[INFO]: the notification worker is ready and waiting for requests")
	<-done

	w.StopWorker()
}
