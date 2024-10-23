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

func NotificationsWorker() *cli.Command {
	return &cli.Command{
		Name:  "notifications",
		Usage: "Manage OpenUEM's Notifications worker",
		Subcommands: []*cli.Command{
			{
				Name:   "start",
				Usage:  "Start an OpenUEM's Notifications worker",
				Action: startNotificationsWorker,
				Flags:  CommonFlags(),
			},
			{
				Name:   "stop",
				Usage:  "Stop an OpenUEM's Notifications worker",
				Action: stopWorker,
			},
		},
	}
}

func startNotificationsWorker(cCtx *cli.Context) error {
	worker := common.NewWorker("")

	if err := worker.CheckCLICommonRequisites(cCtx); err != nil {
		return err
	}

	worker.StartWorker(worker.SubscribeToNotificationWorkerQueues)

	if err := os.WriteFile("PIDFILE", []byte(strconv.Itoa(os.Getpid())), 0666); err != nil {
		return err
	}

	// Keep the connection alive
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Printf("✅  Done! Your Notification Worker is ready and listening for requests\n\n")
	<-done

	worker.StopWorker()

	log.Printf("👋  Done! Your Notification Worker has been shutdown\n\n")
	return nil
}
