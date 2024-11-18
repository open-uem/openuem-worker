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
	w := common.NewWorker("openuem-agent-worker")

	// Get config for service
	if err := w.GenerateCommonWorkerConfig("agents"); err != nil {
		log.Printf("[ERROR]: could not generate config for agent worker: %v", err)
	}

	w.StartWorker(w.SubscribeToAgentWorkerQueues)

	// Keep the connection alive
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Println("[INFO]: the Agent worker is ready and waiting for requests")
	<-done

	w.StopWorker()
}
