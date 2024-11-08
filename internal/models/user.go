package models

import (
	"context"

	"github.com/doncicuto/openuem_ent/user"
	"github.com/doncicuto/openuem_nats"
)

func (m *Model) SetCertificateSent(uid string) error {
	return m.Client.User.Update().SetRegister(openuem_nats.REGISTER_CERTIFICATE_SENT).Where(user.ID(uid)).Exec(context.Background())
}
