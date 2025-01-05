package models

import (
	"context"

	"github.com/open-uem/ent/user"
	"github.com/open-uem/nats"
)

func (m *Model) SetCertificateSent(uid string) error {
	return m.Client.User.Update().SetRegister(nats.REGISTER_CERTIFICATE_SENT).Where(user.ID(uid)).Exec(context.Background())
}

func (m *Model) SetEmailVerified(uid string) error {
	return m.Client.User.Update().SetEmailVerified(true).Where(user.ID(uid)).Exec(context.Background())
}
