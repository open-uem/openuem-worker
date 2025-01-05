package main

import (
	"log"
	"os"

	"github.com/doncicuto/openuem-worker/internal/commands"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:      "openuem-worker",
		Commands:  getCommands(),
		Usage:     "Manage an OpenUEM worker",
		Authors:   []*cli.Author{{Name: "Miguel Angel Alvarez Cabrerizo", Email: "mcabrerizo@openuem.eu"}},
		Copyright: "2024 - Miguel Angel Alvarez Cabrerizo <https://github.com/doncicuto>",
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func getCommands() []*cli.Command {
	return []*cli.Command{
		commands.AgentWorker(),
		commands.CertManagerWorker(),
		commands.NotificationsWorker(),
	}
}
