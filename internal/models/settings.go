package models

import (
	"context"

	"github.com/open-uem/ent"
	"github.com/open-uem/ent/settings"
)

func (m *Model) GetSettings() (*ent.Settings, error) {
	return m.Client.Settings.Query().Only(context.Background())
}

func (m *Model) GetSMTPSettings() (*ent.Settings, error) {
	return m.Client.Settings.Query().Where(settings.Not(settings.HasTenant())).
		Select(settings.FieldSMTPAuth, settings.FieldSMTPPassword,
			settings.FieldSMTPPort, settings.FieldSMTPServer,
			settings.FieldSMTPStarttls, settings.FieldSMTPTLS,
			settings.FieldSMTPUser, settings.FieldMessageFrom).Only(context.Background())
}
