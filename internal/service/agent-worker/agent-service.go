package main

import (
	"log"
	"path/filepath"

	"github.com/doncicuto/openuem-worker/internal/common"
	"github.com/doncicuto/openuem_utils"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
)

func main() {
	w := common.NewWorker("openuem-agent-worker.txt")
	s := openuem_utils.NewOpenUEMWindowsService()

	// Get config for service
	if err := generateAgentWorkerConfig(w); err != nil {
		log.Printf("[ERROR]: could not generate config for cert-manager worker: %v", err)
	}

	s.ServiceStart = func() { w.StartWorker(w.SubscribeToAgentWorkerQueues) }
	s.ServiceStop = w.StopWorker

	// Run service
	err := svc.Run("openuem-cert-manager-worker", s)
	if err != nil {
		log.Printf("[ERROR]: could not run service: %v", err)
	}
}

func generateAgentWorkerConfig(w *common.Worker) error {
	var err error

	w.DBUrl, err = openuem_utils.CreatePostgresDatabaseURL()
	if err != nil {
		log.Printf("[ERROR]: %v", err)
		return err
	}

	cwd, err := common.GetWd()
	if err != nil {
		log.Println("[ERROR]: could not get working directory")
		return err
	}

	k, err := openuem_utils.OpenRegistryForQuery(registry.LOCAL_MACHINE, `SOFTWARE\OpenUEM\Server`)
	if err != nil {
		log.Println("[ERROR]: could not open registry")
		return err
	}
	defer k.Close()

	w.ClientCertPath = filepath.Join(cwd, "certificates", "agents-worker", "worker.cer")
	w.ClientKeyPath = filepath.Join(cwd, "certificates", "agents-worker", "worker.key")
	w.CACertPath = filepath.Join(cwd, "certificates", "ca", "ca.cer")

	w.NATSServers, err = openuem_utils.GetValueFromRegistry(k, "NATSServers")
	if err != nil {
		log.Println("[ERROR]: could not read NATS servers from registry")
		return err
	}

	return nil
}
