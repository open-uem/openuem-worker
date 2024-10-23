package common

import "log"

func (w *Worker) SubscribeToNotificationWorkerQueues() error {
	var err error

	// read SMTP settings from database
	w.Settings, err = w.Model.GetSettings()
	if err != nil {
		log.Printf("[ERROR]: could not get settings from DB, reason: %s", err.Error())
		return err
	}

	_, err = w.NATSConnection.Subscribe("notification.confirm_email", w.SendConfirmEmailHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %s", err.Error())
		return err
	}
	log.Println("[INFO]: subscribed to queue notification.confirm_email")

	_, err = w.NATSConnection.Subscribe("notification.send_certificate", w.SendUserCertificateHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %s", err.Error())
		return err
	}
	log.Println("[INFO]: subscribed to queue notification.send_certificate")

	return nil
}
