package common

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
	openuem_nats "github.com/open-uem/nats"
)

func (w *Worker) SubscribeToAgentWorkerQueues() error {
	_, err := w.NATSConnection.QueueSubscribe("report", "openuem-agents", w.ReportReceivedHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to queue report")

	_, err = w.NATSConnection.QueueSubscribe("deployresult", "openuem-agents", w.DeployResultReceivedHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message deployresult")

	_, err = w.NATSConnection.QueueSubscribe("ping.agentworker", "openuem-agents", w.PingHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message ping")

	_, err = w.NATSConnection.QueueSubscribe("agentconfig", "openuem-agents", w.AgentConfigHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message agent")
	return nil
}

func (w *Worker) ReportReceivedHandler(msg *nats.Msg) {
	data := openuem_nats.AgentReport{}

	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("[ERROR]: could not unmarshal agent report, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveAgentInfo(&data, w.NATSServers); err != nil {
		log.Printf("[ERROR]: could not save agent info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveComputerInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save computer info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveOSInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save operating system info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveAntivirusInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save antivirus info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveSystemUpdateInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save system updates info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveAppsInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save apps info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveMonitorsInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save monitors info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveLogicalDisksInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save logical disks info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SavePrintersInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save printers info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveNetworkAdaptersInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save network adapters info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveSharesInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save shares info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveUpdatesInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save updates info into database, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveReleaseInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save release info into database, reason: %s\n", err.Error())
	}

	if err := msg.Respond([]byte("Report received!")); err != nil {
		log.Printf("[ERROR]: could not respond to report message, reason: %s\n", err.Error())
	}
}

func (w *Worker) DeployResultReceivedHandler(msg *nats.Msg) {
	data := openuem_nats.DeployAction{}

	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("[ERROR]: could not unmarshal deploy message, reason: %s\n", err.Error())
	}

	if err := w.Model.SaveDeployInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save deployment info into database, reason: %s\n", err.Error())

		if err := msg.Respond([]byte(err.Error())); err != nil {
			log.Printf("[ERROR]: could not respond to deploy message, reason: %s\n", err.Error())
		}
		return
	}

	if err := msg.Respond([]byte("")); err != nil {
		log.Printf("[ERROR]: could not respond to deploy message, reason: %s\n", err.Error())
	}
}
