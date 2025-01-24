package commands

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/go-co-op/gocron/v2"
	"github.com/open-uem/openuem-worker/internal/common"
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
	var err error

	worker := common.NewWorker("")

	if err := worker.CheckCLICommonRequisites(cCtx); err != nil {
		log.Printf("[ERROR]: could not generate config for Agents Worker: %v", err)
	}

	if err := os.WriteFile("PIDFILE", []byte(strconv.Itoa(os.Getpid())), 0666); err != nil {
		return err
	}

	// Start Task Scheduler
	worker.TaskScheduler, err = gocron.NewScheduler()
	if err != nil {
		log.Fatalf("[FATAL]: could not create task scheduler, reason: %v", err)
	}
	worker.TaskScheduler.Start()
	log.Println("[INFO]: task scheduler has been started")

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
