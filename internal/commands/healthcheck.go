package commands

import (
	"database/sql"
	"log"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/open-uem/openuem-worker/internal/common"
	"github.com/urfave/cli/v2"
)

func HealthCheck() *cli.Command {
	return &cli.Command{
		Name:   "healthcheck",
		Usage:  "Check the health of the worker",
		Action: healtCheck,
		Flags:  CommonFlags(),
	}
}

func healtCheck(cCtx *cli.Context) error {
	worker := common.NewWorker("")

	// Get CLI flags values
	if err := worker.CheckCLICommonRequisites(cCtx); err != nil {
		log.Printf("[ERROR]: could not check requisites, reason: %v", err)
		os.Exit(1)
	}

	// Check if we can connect with the NATS service
	nc, err := nats.Connect(worker.NATSServers, nats.RootCAs(worker.CACertPath), nats.ClientCert(worker.ClientCertPath, worker.ClientKeyPath))
	if err != nil {
		log.Printf("[ERROR]: could not connect to NATS server, reason: %v", err)
		os.Exit(1)
	}
	nc.Close()

	// Check if we can connect with the database
	db, err := sql.Open("pgx", worker.DBUrl)
	if err != nil {
		log.Printf("[ERROR]: could not connect to database, reason: %v", err)
		os.Exit(1)
	}
	db.Close()

	return nil
}
