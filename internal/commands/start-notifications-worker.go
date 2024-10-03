package commands

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/doncicuto/openuem-worker/internal/commands/notifications"
	"github.com/doncicuto/openuem_nats"
	"github.com/nats-io/nats.go"
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
	var err error
	command := WorkerCommand{}
	command.checkCommonRequisites(cCtx)

	// Specific requisite
	log.Println("⚙️   getting settings from database")
	command.Settings, err = command.Model.GetSettings()
	if err != nil {
		log.Fatalf("❌ could not get settings from DB, reason: %s", err.Error())
	}

	if err := command.connectToNATS(cCtx); err != nil {
		return err
	}

	log.Println("📩  subscribing to notification messages")
	// TODO do something with subscription?
	_, err = command.MessageServer.Connection.Subscribe("notification.confirm_email", command.sendConfirmEmailHandler)
	if err != nil {
		log.Fatalf("❌ could not subscribe to NATS message, reason: %s", err.Error())
	}

	_, err = command.MessageServer.Connection.Subscribe("notification.send_certificate", command.sendUserCertificateHandler)
	if err != nil {
		log.Fatalf("❌ could not subscribe to NATS message, reason: %s", err.Error())
	}

	if err := os.WriteFile("PIDFILE", []byte(strconv.Itoa(os.Getpid())), 0666); err != nil {
		return err
	}

	// Keep the connection alive
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Printf("✅  Done! Your Notification Worker is ready and listening for requests\n\n")
	<-done

	log.Printf("👋  Done! Your Notification Worker has been shutdown\n\n")
	return nil
}

func (command *WorkerCommand) sendConfirmEmailHandler(msg *nats.Msg) {
	notification := openuem_nats.Notification{}

	err := json.Unmarshal(msg.Data, &notification)
	if err != nil {
		log.Fatalf("❌ could not unmarshal notification request, reason: %s", err.Error())
		return
	}

	mailMessage, err := notifications.PrepareMessage(&notification, command.Settings)
	if err != nil {
		log.Fatalf("❌ could not prepare notification message, reason: %s", err.Error())
		return
	}

	client, err := notifications.PrepareSMTPClient(command.Settings)
	if err != nil {
		log.Fatalf("❌ could not prepare SMTP client, reason: %s", err.Error())
		return
	}
	if err := client.DialAndSend(mailMessage); err != nil {
		log.Fatalf("❌ could not connect and send message, reason: %s", err.Error())
		return
	}

	if err := msg.Respond([]byte("Confirmation email has been sent!")); err != nil {
		log.Println("[ERR]: could not sent response", err.Error())
		return
	}
}

func (command *WorkerCommand) sendUserCertificateHandler(msg *nats.Msg) {
	notification := openuem_nats.Notification{}

	if err := json.Unmarshal(msg.Data, &notification); err != nil {
		log.Fatalf("❌ could not unmarshal notification request, reason: %s", err.Error())
		return
	}

	mailMessage, err := notifications.PrepareMessage(&notification, command.Settings)
	if err != nil {
		log.Fatalf("❌ could not prepare notification message, reason: %s", err.Error())
		return
	}

	client, err := notifications.PrepareSMTPClient(command.Settings)
	if err != nil {
		log.Fatalf("❌ could not prepare SMTP client, reason: %s", err.Error())
		return
	}

	err = client.DialAndSend(mailMessage)
	if err != nil {
		log.Fatalf("❌ could not connect and send message, reason: %s", err.Error())
		return
	}

	if err := msg.Respond([]byte("User certificate has been sent!")); err != nil {
		log.Println("[ERR]: could not sent response", err.Error())
		return
	}
}
