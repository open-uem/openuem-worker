package models

import (
	"context"
	"time"
)

func (m *Model) AddRevocation(serial int64, reason int, info string, expiry time.Time) error {
	_, err := m.Client.Revocation.Create().SetID(serial).SetReason(reason).SetInfo(info).SetExpiry(expiry).SetRevoked(time.Now()).Save(context.Background())
	if err != nil {
		return err
	}
	return nil
}
