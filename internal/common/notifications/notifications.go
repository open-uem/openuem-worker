package notifications

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/doncicuto/openuem_ent"
	"github.com/doncicuto/openuem_nats"
	"github.com/wneessen/go-mail"
)

func PrepareMessage(notification *openuem_nats.Notification, settings *openuem_ent.Settings) (*mail.Msg, error) {
	if notification.From == "" {
		if settings.MessageFrom != "" {
			notification.From = settings.MessageFrom
		} else {
			return nil, fmt.Errorf("from cannot be empty")
		}
	}

	m := mail.NewMsg()
	if err := m.From(settings.MessageFrom); err != nil {
		return nil, fmt.Errorf("failed to set From address: %s", err.Error())
	}
	if err := m.To(notification.To); err != nil {
		return nil, fmt.Errorf("failed to set To address: %s", err.Error())
	}

	m.Subject(notification.Subject)
	templateBuffer := new(bytes.Buffer)
	if err := EmailTemplate(notification).Render(context.Background(), templateBuffer); err != nil {
		return nil, fmt.Errorf("failed to set To address: %s", err.Error())
	}
	m.SetBodyString(mail.TypeTextHTML, templateBuffer.String())

	if notification.MessageAttachFileName != "" {
		data, err := base64.StdEncoding.DecodeString(notification.MessageAttachFile)
		if err != nil {
			return nil, fmt.Errorf("failed to decode file content: %s", err.Error())
		}
		reader := bytes.NewReader(data)
		err = m.AttachReader(notification.MessageAttachFileName, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to attach file: %s", err.Error())
		}
	}

	if notification.MessageAttachFileName2 != "" {
		data, err := base64.StdEncoding.DecodeString(notification.MessageAttachFile2)
		if err != nil {
			return nil, fmt.Errorf("failed to decode file content: %s", err.Error())
		}
		reader := bytes.NewReader(data)
		err = m.AttachReader(notification.MessageAttachFileName2, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to attach file: %s", err.Error())
		}
	}
	return m, nil
}

func PrepareSMTPClient(settings *openuem_ent.Settings) (*mail.Client, error) {
	var err error
	var c *mail.Client
	if settings.SMTPAuth == "NOAUTH" || (settings.SMTPUser == "" && settings.SMTPPassword == "") {
		c, err = mail.NewClient(settings.SMTPServer, mail.WithPort(settings.SMTPPort))
	} else {
		c, err = mail.NewClient(settings.SMTPServer, mail.WithPort(settings.SMTPPort), mail.WithSMTPAuth(mail.SMTPAuthType(settings.SMTPAuth)),
			mail.WithUsername(settings.SMTPUser), mail.WithPassword(settings.SMTPPassword))
	}

	if err != nil {
		return nil, err
	}
	return c, nil
}
