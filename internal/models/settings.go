package models

import (
	"context"

	"github.com/open-uem/ent"
)

func (m *Model) GetSettings() (*ent.Settings, error) {
	return m.Client.Settings.Query().Only(context.Background())
}
