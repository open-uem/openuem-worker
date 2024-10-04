package commands

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/doncicuto/openuem_nats"
	"github.com/nats-io/nats.go"
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
	command := WorkerCommand{}
	command.checkCommonRequisites(cCtx)

	if err := command.connectToNATS(cCtx); err != nil {
		return err
	}

	log.Println("📩  subscribing to agents messages")

	_, err = command.MessageServer.Connection.QueueSubscribe("report", "openuem-agents", command.reportReceivedHandler)
	if err != nil {
		log.Fatalf("❌ could not subscribe to NATS message, reason: %s", err.Error())
	}

	_, err = command.MessageServer.Connection.QueueSubscribe("deployresult", "openuem-agents", command.deployResultReceivedHandler)
	if err != nil {
		log.Fatalf("❌ could not subscribe to NATS message, reason: %s", err.Error())
	}

	// Keep the connection alive
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Printf("✅ Done! Your Agents worker is ready\n\n")
	<-done

	command.MessageServer.Close()
	log.Printf("👋 Done! Your Agents worker has been shutdown\n\n")
	return nil
}

func (command *WorkerCommand) reportReceivedHandler(msg *nats.Msg) {
	data := openuem_nats.AgentReport{}

	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("❌ could not unmarshal agent report, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveAgentInfo(&data); err != nil {
		log.Printf("❌ could not save agent info into database, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveComputerInfo(&data); err != nil {
		log.Printf("❌ could not save computer info into database, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveOSInfo(&data); err != nil {
		log.Printf("❌ could not save operating system info into database, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveAntivirusInfo(&data); err != nil {
		log.Printf("❌ could not save antivirus info into database, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveSystemUpdateInfo(&data); err != nil {
		log.Printf("❌ could not save system updates info into database, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveAppsInfo(&data); err != nil {
		log.Printf("❌ could not save apps info into database, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveMonitorsInfo(&data); err != nil {
		log.Printf("❌ could not save monitors info into database, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveLogicalDisksInfo(&data); err != nil {
		log.Printf("❌ could not save logical disks info into database, reason: %s\n", err.Error())
	}

	if err := command.Model.SavePrintersInfo(&data); err != nil {
		log.Printf("❌ could not save printers info into database, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveNetworkAdaptersInfo(&data); err != nil {
		log.Printf("❌ could not save network adapters info into database, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveSharesInfo(&data); err != nil {
		log.Printf("❌ could not save shares info into database, reason: %s\n", err.Error())
	}

	if err := msg.Respond([]byte("Report received!")); err != nil {
		log.Printf("❌ could not respond to report message, reason: %s\n", err.Error())
	}
}

func (command *WorkerCommand) deployResultReceivedHandler(msg *nats.Msg) {
	data := openuem_nats.DeployAction{}

	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("❌ could not unmarshal deploy message, reason: %s\n", err.Error())
	}

	if err := command.Model.SaveDeployInfo(&data); err != nil {
		log.Printf("❌ could not save deployment info into database, reason: %s\n", err.Error())

		if err := msg.Respond([]byte(err.Error())); err != nil {
			log.Printf("❌ could not respond to deploy message, reason: %s\n", err.Error())
		}
		return
	}

	if err := msg.Respond([]byte("")); err != nil {
		log.Printf("❌ could not respond to deploy message, reason: %s\n", err.Error())
	}
}
