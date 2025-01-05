package models

import (
	"context"

	"github.com/open-uem/openuem_ent"
)

func (m *Model) GetSettings() (*openuem_ent.Settings, error) {
	return m.Client.Settings.Query().Only(context.Background())
}
