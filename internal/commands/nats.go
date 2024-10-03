package commands

import (
	"log"

	"github.com/doncicuto/openuem_nats"
	"github.com/urfave/cli/v2"
)

func (command *WorkerCommand) connectToNATS(cCtx *cli.Context) error {
	log.Println("🔌  connecting to NATS cluster")
	command.MessageServer = openuem_nats.New(cCtx.String("nats-host"), cCtx.String("nats-port"), cCtx.String("cert"), cCtx.String("key"), command.CACert)
	return command.MessageServer.Connect()
}
