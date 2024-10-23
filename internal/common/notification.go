package common

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/doncicuto/openuem-worker/internal/common/notifications"
	"github.com/doncicuto/openuem_nats"
	"github.com/nats-io/nats.go"
)

func (w *Worker) SendNotification() error {
	notification := openuem_nats.Notification{
		To:                    w.CertRequest.Email,
		Subject:               "Your certificate to log in to OpenUEM console",
		MessageTitle:          "OpenUEM | Your certificate",
		MessageText:           "You can find attached the digital certificate that you must import to your browser so you can use it to log in to the OpenUEM console",
		MessageGreeting:       fmt.Sprintf("Hi %s", w.CertRequest.FullName),
		MessageAttachFileName: w.CertRequest.Username + ".pfx",
		MessageAttachFile:     base64.StdEncoding.EncodeToString(w.PKCS12),
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	if err := w.NATSConnection.Publish("notification.send_certificate", data); err != nil {
		return err
	}

	return nil
}

func (w *Worker) SendConfirmEmailHandler(msg *nats.Msg) {
	notification := openuem_nats.Notification{}

	err := json.Unmarshal(msg.Data, &notification)
	if err != nil {
		log.Printf("[ERR]: could not unmarshal notification request, reason: %v", err.Error())
		return
	}

	mailMessage, err := notifications.PrepareMessage(&notification, w.Settings)
	if err != nil {
		log.Printf("[ERR]: could not prepare notification message, reason: %v", err.Error())
		return
	}

	client, err := notifications.PrepareSMTPClient(w.Settings)
	if err != nil {
		log.Printf("[ERR]: could not prepare SMTP client, reason: %v", err.Error())
		return
	}
	if err := client.DialAndSend(mailMessage); err != nil {
		log.Printf("[ERR]: could not connect and send message, reason: %v", err.Error())
		return
	}

	if err := msg.Respond([]byte("Confirmation email has been sent!")); err != nil {
		log.Printf("[ERR]: could not sent response, reason: %v", err.Error())
		return
	}
}

func (w *Worker) SendUserCertificateHandler(msg *nats.Msg) {
	notification := openuem_nats.Notification{}

	if err := json.Unmarshal(msg.Data, &notification); err != nil {
		log.Printf("[ERR]: could not unmarshal notification request, reason: %v", err.Error())
		return
	}

	mailMessage, err := notifications.PrepareMessage(&notification, w.Settings)
	if err != nil {
		log.Printf("[ERR]: could not prepare notification message, reason: %v", err.Error())
		return
	}

	client, err := notifications.PrepareSMTPClient(w.Settings)
	if err != nil {
		log.Printf("[ERR]: could not prepare SMTP client, reason: %v", err.Error())
		return
	}

	err = client.DialAndSend(mailMessage)
	if err != nil {
		log.Printf("[ERR]: could not connect and send message, reason: %v", err.Error())
		return
	}

	if err := msg.Respond([]byte("User certificate has been sent!")); err != nil {
		log.Printf("[ERR]: could not sent response, reason: %v", err.Error())
		return
	}
}
