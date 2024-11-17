package common

import (
	"context"
	"log"
	"time"

	"github.com/doncicuto/openuem_ent"
	"github.com/nats-io/nats.go/jetstream"
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
