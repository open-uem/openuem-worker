package models

import (
	"context"

	"github.com/doncicuto/openuem_ent/agent"
	"github.com/doncicuto/openuem_ent/deployment"
	"github.com/doncicuto/openuem_nats"
)

func (m *Model) SaveDeployInfo(data *openuem_nats.DeployAction) error {

	if data.Action == "install" {
		return m.Client.Deployment.Create().
			SetPackageID(data.PackageId).
			SetName(data.PackageName).
			SetVersion(data.PackageVersion).
			SetInstalled(data.When).
			SetOwnerID(data.AgentId).
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
