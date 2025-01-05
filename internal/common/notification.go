package common

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/open-uem/ent"
	openuem_nats "github.com/open-uem/nats"
	"github.com/open-uem/openuem-worker/internal/common/notifications"
)

func (w *Worker) JetStreamNotificationsHandler(msg jetstream.Msg) {
	if msg.Subject() == "notification.confirm_email" {
		w.JetStreamSendConfirmEmailHandler(msg)
	}

	if msg.Subject() == "notification.send_certificate" {
		w.JetStreamSendUserCertificateHandler(msg)
	}
}

func (w *Worker) JetStreamSendConfirmEmailHandler(msg jetstream.Msg) {
	notification := openuem_nats.Notification{}

	if w.Settings == nil {
		log.Println("[ERROR]: no SMTP settings found, retry in 5 minutes")
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	err := json.Unmarshal(msg.Data(), &notification)
	if err != nil {
		log.Printf("[ERROR]: could not unmarshal notification request, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	mailMessage, err := notifications.PrepareMessage(&notification, w.Settings)
	if err != nil {
		log.Printf("[ERROR]: could not prepare notification message, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	client, err := notifications.PrepareSMTPClient(w.Settings)
	if err != nil {
		log.Printf("[ERROR]: could not prepare SMTP client, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}
	if err := client.DialAndSend(mailMessage); err != nil {
		log.Printf("[ERROR]: could not connect and send message, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	if err := msg.Ack(); err != nil {
		log.Printf("[ERROR]: could not sent ACK, reason: %v", err.Error())
		return
	}
}

func (w *Worker) JetStreamSendUserCertificateHandler(msg jetstream.Msg) {
	notification := openuem_nats.Notification{}

	if w.Settings == nil {
		log.Println("[ERROR]: no SMTP settings found, retry in 5 minutes")
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	if err := json.Unmarshal(msg.Data(), &notification); err != nil {
		log.Printf("[ERROR]: could not unmarshal notification request, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	mailMessage, err := notifications.PrepareMessage(&notification, w.Settings)
	if err != nil {
		log.Printf("[ERROR]: could not prepare notification message, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	client, err := notifications.PrepareSMTPClient(w.Settings)
	if err != nil {
		log.Printf("[ERROR]: could not prepare SMTP client, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	err = client.DialAndSend(mailMessage)
	if err != nil {
		log.Printf("[ERROR]: could not connect and send message, reason: %v", err.Error())
		msg.NakWithDelay(5 * time.Minute)
		return
	}

	if err := msg.Ack(); err != nil {
		log.Printf("[ERROR]: could not sent response, reason: %v", err.Error())
		return
	}
}

func (w *Worker) ReloadSettingsHandler(msg *nats.Msg) {
	var err error
	// read again SMTP settings from database
	w.Settings, err = w.Model.GetSettings()
	if err != nil {
		if ent.IsNotFound(err) {
			log.Println("[INFO]: no SMTP settings found")
		} else {
			log.Printf("[ERROR]: could not get settings from DB, reason: %v", err)
			return
		}
	}

	log.Println("[INFO]: SMTP settings have been reloaded")
}
