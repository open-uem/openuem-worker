package models

import (
	"context"
	"strconv"

	"github.com/open-uem/ent"
	"github.com/open-uem/ent/settings"
	"github.com/open-uem/ent/tenant"
)

func (m *Model) GetSettings(t string) (*ent.Settings, error) {
	if t == "" {
		return m.Client.Settings.Query().Where(settings.Not(settings.HasTenant())).Only(context.Background())
	} else {
		tenantID, err := strconv.Atoi(t)
		if err != nil {
			return m.Client.Settings.Query().Where(settings.Not(settings.HasTenant())).Only(context.Background())
		}

		s, err := m.Client.Settings.Query().Where(settings.HasTenantWith(tenant.ID(tenantID))).Only(context.Background())
		if err != nil {
			return m.Client.Settings.Query().Where(settings.Not(settings.HasTenant())).Only(context.Background())
		}
		return s, nil
	}
}

func (m *Model) GetSMTPSettings() (*ent.Settings, error) {
	return m.Client.Settings.Query().Where(settings.Not(settings.HasTenant())).
		Select(settings.FieldSMTPAuth, settings.FieldSMTPPassword,
			settings.FieldSMTPPort, settings.FieldSMTPServer,
			settings.FieldSMTPStarttls, settings.FieldSMTPTLS,
			settings.FieldSMTPUser, settings.FieldMessageFrom).Only(context.Background())
}
