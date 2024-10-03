package models

import (
	"context"

	"github.com/doncicuto/openuem_ent/user"
)

func (m *Model) SetCertificateSent(uid string) error {
	return m.Client.User.Update().SetRegister("users.certificate_sent").Where(user.ID(uid)).Exec(context.Background())
}
