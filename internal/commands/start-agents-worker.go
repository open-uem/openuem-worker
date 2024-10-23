package commands

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/doncicuto/openuem-worker/internal/common"
	"github.com/urfave/cli/v2"
)

func AgentWorker() *cli.Command {
	return &cli.Command{
		Name:  "agents",
		Usage: "Manage OpenUEM's Agents worker",
		Subcommands: []*cli.Command{
			{
				Name:   "start",
				Usage:  "Start an OpenUEM's Agents worker",
				Action: startAgentsWorker,
				Flags:  CommonFlags(),
			},
			{
				Name:   "stop",
				Usage:  "Stop an OpenUEM's Agents worker",
				Action: stopWorker,
			},
		},
	}
}

func startAgentsWorker(cCtx *cli.Context) error {
	worker := common.NewWorker("")

	if err := worker.CheckCLICommonRequisites(cCtx); err != nil {
		log.Printf("[ERROR]: could not generate config for Agents Worker: %v", err)
	}

	if err := os.WriteFile("PIDFILE", []byte(strconv.Itoa(os.Getpid())), 0666); err != nil {
		return err
	}

	worker.StartWorker(worker.SubscribeToAgentWorkerQueues)
	// Keep the connection alive
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Printf("[INFO]: agents worker is ready\n\n")
	<-done

	worker.StopWorker()
	log.Printf("[INFO]: agents worker has been shutdown\n\n")
	return nil
}
