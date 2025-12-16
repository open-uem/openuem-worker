package models

import (
	"context"

	"github.com/open-uem/ent"
	"github.com/open-uem/ent/netbirdsettings"
	"github.com/open-uem/ent/tenant"
)

func (m *Model) GetNetbirdSettings(tenantID int) (*ent.NetbirdSettings, error) {
	return m.Client.NetbirdSettings.Query().Where(netbirdsettings.HasTenantWith(tenant.ID(tenantID))).Only(context.Background())
}
