//go:build linux

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/doncicuto/openuem-worker/internal/common"
	"github.com/doncicuto/openuem_ent/component"
)

func main() {
	w := common.NewWorker("openuem-notification-worker", component.ComponentNotificationWorker)

	// Get config for service
	if err := w.GenerateCommonWorkerConfig(component.ComponentNotificationWorker.String()); err != nil {
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
