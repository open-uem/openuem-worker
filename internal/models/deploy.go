package models

import (
	"context"

	"github.com/open-uem/ent/agent"
	"github.com/open-uem/ent/deployment"
	"github.com/open-uem/nats"
)

func (m *Model) SaveDeployInfo(data *nats.DeployAction) error {

	if data.Action == "install" {
		return m.Client.Deployment.Update().
			SetInstalled(data.When).
			SetUpdated(data.When).
			Where(deployment.And(deployment.PackageID(data.PackageId), deployment.HasOwnerWith(agent.ID(data.AgentId)))).
			Exec(context.Background())
	}

	if data.Action == "update" {
		return m.Client.Deployment.Update().
			SetUpdated(data.When).
			Where(deployment.And(deployment.PackageID(data.PackageId), deployment.HasOwnerWith(agent.ID(data.AgentId)))).
			Exec(context.Background())
	}

	if data.Action == "uninstall" {
		_, err := m.Client.Deployment.Delete().
			Where(deployment.And(deployment.PackageID(data.PackageId), deployment.HasOwnerWith(agent.ID(data.AgentId)))).
			Exec(context.Background())
		if err != nil {
			return err
		}
	}

	return nil
}
