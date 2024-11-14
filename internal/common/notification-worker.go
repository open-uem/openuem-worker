package common

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/doncicuto/openuem_ent"
	"github.com/doncicuto/openuem_utils"
	"github.com/nats-io/nats.go/jetstream"
	"golang.org/x/sys/windows/registry"
)

func (w *Worker) SubscribeToNotificationWorkerQueues() error {
	var err error
	var ctx context.Context

	js, err := jetstream.New(w.NATSConnection)
	if err != nil {
		log.Printf("[ERROR]: could not intantiate JetStream: %v", err)
		return err
	}

	// read SMTP settings from database
	w.Settings, err = w.Model.GetSettings()
	if err != nil {
		if openuem_ent.IsNotFound(err) {
			log.Println("[INFO]: no SMTP settings found")
		} else {
			log.Printf("[ERROR]: could not get settings from DB, reason: %v", err)
			return err
		}
	}

	ctx, w.JetstreamContextCancel = context.WithTimeout(context.Background(), 60*time.Minute)
	s, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "NOTIFICATION_STREAM",
		Subjects: []string{"notification.confirm_email", "notification.send_certificate"},
	})
	if err != nil {
		log.Printf("[ERROR]: could not create stream: %v", err)
		return err
	}

	c1, err := s.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "NotificationsConsumer",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		log.Printf("[ERROR]: could not create Jetstream consumer: %v", err)
		return err
	}
	// TODO stop consume context ()
	_, err = c1.Consume(w.JetStreamNotificationsHandler, jetstream.ConsumeErrHandler(func(consumeCtx jetstream.ConsumeContext, err error) {
		log.Printf("[ERROR]: consumer error: %v", err)
	}))
	if err != nil {
		log.Printf("[ERROR]: could not start Notifications consumer: %v", err)
		return err
	}
	log.Println("[INFO]: Notifications consumer is ready to serve")

	_, err = w.NATSConnection.Subscribe("notification.reload_settings", w.ReloadSettingsHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %v", err)
		return err
	}
	log.Println("[INFO]: subscribed to queue notification.reload_setting")

	_, err = w.NATSConnection.QueueSubscribe("ping.notificationworker", "openuem-notification", w.PingHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to queue ping")

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
