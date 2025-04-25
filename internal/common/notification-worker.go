package common

import (
	"log"

	"github.com/open-uem/ent"
)

func (w *Worker) SubscribeToNotificationWorkerQueues() error {
	var err error

	// read SMTP settings from database
	w.Settings, err = w.Model.GetSettings()
	if err != nil {
		if ent.IsNotFound(err) {
			log.Println("[INFO]: no SMTP settings found")
		} else {
			log.Printf("[ERROR]: could not get settings from DB, reason: %v", err)
			return err
		}
	}

	_, err = w.NATSConnection.Subscribe("notification.reload_settings", w.ReloadSettingsHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to notification.reload_settings, reason: %v", err)
		return err
	}
	log.Println("[INFO]: subscribed to queue notification.reload_setting")

	_, err = w.NATSConnection.QueueSubscribe("notification.confirm_email", "openuem-notification", w.SendConfirmEmailHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to notification.confirm_email, reason: %v", err)
		return err
	}
	log.Println("[INFO]: subscribed to queue notification.confirm_email")

	_, err = w.NATSConnection.QueueSubscribe("notification.send_certificate", "openuem-notification", w.SendUserCertificateHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to notification.send_certificate, reason: %v", err)
		return err
	}
	log.Println("[INFO]: subscribed to queue notification.send_certificate")

	_, err = w.NATSConnection.QueueSubscribe("ping.notificationworker", "openuem-notification", w.PingHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to ping.notificationworker, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to queue ping.notificationworker")

	return nil
}
