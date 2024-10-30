package common

import (
	"log"
	"path/filepath"

	"github.com/doncicuto/openuem_ent"
	"github.com/doncicuto/openuem_utils"
	"golang.org/x/sys/windows/registry"
)

func (w *Worker) SubscribeToNotificationWorkerQueues() error {
	var err error

	// read SMTP settings from database
	w.Settings, err = w.Model.GetSettings()
	if err != nil {
		if openuem_ent.IsNotFound(err) {
			log.Println("[INFO]: no SMTP settings found")
		} else {
			log.Printf("[ERROR]: could not get settings from DB, reason: %s", err.Error())
			return err
		}
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

	_, err = w.NATSConnection.Subscribe("notification.reload_settings", w.ReloadSettingsHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %s", err.Error())
		return err
	}
	log.Println("[INFO]: subscribed to queue notification.reload_setting")

	return nil
}

func (w *Worker) GenerateNotificationWorkerConfig() error {
	var err error

	cwd, err := GetWd()
	if err != nil {
		log.Println("[ERROR]: could not get working directory")
		return err
	}

	k, err := openuem_utils.OpenRegistryForQuery(registry.LOCAL_MACHINE, `SOFTWARE\OpenUEM\Server`)
	if err != nil {
		log.Println("[ERROR]: could not open registry")
		return err
	}
	defer k.Close()

	w.DBUrl, err = openuem_utils.CreatePostgresDatabaseURL()
	if err != nil {
		log.Printf("[ERROR]: %v", err)
		return err
	}

	w.ClientCertPath = filepath.Join(cwd, "certificates", "notification-worker", "worker.cer")
	w.ClientKeyPath = filepath.Join(cwd, "certificates", "notification-worker", "worker.key")
	w.CACertPath = filepath.Join(cwd, "certificates", "ca", "ca.cer")
	w.CAKeyPath = filepath.Join(cwd, "certificates", "ca", "ca.key")

	w.NATSServers, err = openuem_utils.GetValueFromRegistry(k, "NATSServers")
	if err != nil {
		log.Println("[ERROR]: could not read NATS servers from registry")
		return err
	}

	// read required certificates and private keys
	w.CACert, err = openuem_utils.ReadPEMCertificate(w.CACertPath)
	if err != nil {
		log.Println("[ERROR]: could not read CA cert file")
		return err
	}

	return nil
}
