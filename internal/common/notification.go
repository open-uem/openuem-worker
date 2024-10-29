package common

import (
	"encoding/json"
	"log"
	"time"

	"github.com/doncicuto/openuem-worker/internal/common/notifications"
	"github.com/doncicuto/openuem_nats"
	"github.com/nats-io/nats.go"
)

func (w *Worker) SendConfirmEmailHandler(msg *nats.Msg) {
	notification := openuem_nats.Notification{}

	err := json.Unmarshal(msg.Data, &notification)
	if err != nil {
		log.Printf("[ERR]: could not unmarshal notification request, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	mailMessage, err := notifications.PrepareMessage(&notification, w.Settings)
	if err != nil {
		log.Printf("[ERR]: could not prepare notification message, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	client, err := notifications.PrepareSMTPClient(w.Settings)
	if err != nil {
		log.Printf("[ERR]: could not prepare SMTP client, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}
	if err := client.DialAndSend(mailMessage); err != nil {
		log.Printf("[ERR]: could not connect and send message, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
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
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	mailMessage, err := notifications.PrepareMessage(&notification, w.Settings)
	if err != nil {
		log.Printf("[ERR]: could not prepare notification message, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	client, err := notifications.PrepareSMTPClient(w.Settings)
	if err != nil {
		log.Printf("[ERR]: could not prepare SMTP client, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	err = client.DialAndSend(mailMessage)
	if err != nil {
		log.Printf("[ERR]: could not connect and send message, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	if err := msg.Respond([]byte("User certificate has been sent!")); err != nil {
		log.Printf("[ERR]: could not sent response, reason: %v", err.Error())
		return
	}
}
