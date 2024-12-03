//go:build windows

package main

import (
	"log"

	"github.com/doncicuto/openuem-worker/internal/common"
	"github.com/doncicuto/openuem_ent/component"
	"github.com/doncicuto/openuem_utils"
	"golang.org/x/sys/windows/svc"
)

func main() {
	w := common.NewWorker("openuem-agent-worker.txt", component.ComponentAgentWorker)
	s := openuem_utils.NewOpenUEMWindowsService()

	// Get config for service
	if err := w.GenerateAgentWorkerConfig(); err != nil {
		log.Printf("[ERROR]: could not generate config for agent worker: %v", err)
	}

	s.ServiceStart = func() { w.StartWorker(w.SubscribeToAgentWorkerQueues) }
	s.ServiceStop = w.StopWorker

	// Run service
	err := svc.Run("openuem-agent-worker", s)
	if err != nil {
		log.Printf("[ERROR]: could not run service: %v", err)
	}
}
