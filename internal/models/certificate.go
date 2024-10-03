package models

import (
	"context"
	"time"

	"github.com/doncicuto/openuem_ent/certificate"
)

func (m *Model) SaveCertificate(serial int64, certType certificate.Type, uid, description string, expiry time.Time) error {

	_, err := m.Client.Certificate.Create().SetID(serial).SetType(certType).SetDescription(description).SetExpiry(expiry).SetUID(uid).Save(context.Background())
	if err != nil {
		return err
	}
	return nil
}
