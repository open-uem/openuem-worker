package models

import (
	"context"
)

func (m *Model) AddRevocation(serial int64, reason int, info string) error {
	_, err := m.Client.Revocation.Create().SetID(serial).SetReason(reason).SetInfo(info).Save(context.Background())
	if err != nil {
		return err
	}
	return nil
}
