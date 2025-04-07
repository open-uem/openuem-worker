package models

import (
	"context"
	"log"
	"strings"

	"github.com/open-uem/ent/agent"
	"github.com/open-uem/ent/deployment"
	"github.com/open-uem/ent/wingetconfigexclusion"
	"github.com/open-uem/nats"
)

func (m *Model) SaveDeployInfo(data *nats.DeployAction) error {

	if data.Action == "install" {
		return m.Client.Deployment.Update().
			SetInstalled(data.When).
			SetUpdated(data.When).
			SetFailed(data.Failed).
			Where(deployment.And(deployment.PackageID(data.PackageId), deployment.HasOwnerWith(agent.ID(data.AgentId)))).
			Exec(context.Background())
	}

	if data.Action == "update" {
		return m.Client.Deployment.Update().
			SetUpdated(data.When).
			SetFailed(data.Failed).
			Where(deployment.And(deployment.PackageID(data.PackageId), deployment.HasOwnerWith(agent.ID(data.AgentId)))).
			Exec(context.Background())
	}

	if data.Action == "uninstall" {

		d, err := m.Client.Deployment.Query().Where(deployment.And(deployment.PackageID(data.PackageId), deployment.HasOwnerWith(agent.ID(data.AgentId)))).Only(context.Background())
		if err != nil {
			return err
		}

		// TODO: if package remove failed don't delete deployment, save error
		if data.Failed {
			return m.Client.Deployment.Update().
				SetUpdated(data.When).
				SetFailed(data.Failed).
				Where(deployment.And(deployment.PackageID(data.PackageId), deployment.HasOwnerWith(agent.ID(data.AgentId)))).
				Exec(context.Background())
		} else {
			_, err = m.Client.Deployment.Delete().
				Where(deployment.And(deployment.PackageID(data.PackageId), deployment.HasOwnerWith(agent.ID(data.AgentId)))).
				Exec(context.Background())
			if err != nil {
				return err
			}

			// If package was installed due to a profile, add a new exclusion as we've removed it using OpenUEM Console
			if d.ByProfile {
				return m.Client.WingetConfigExclusion.Create().SetPackageID(data.PackageId).SetOwnerID(data.AgentId).Exec(context.Background())
			}
		}
	}

	return nil
}

func (m *Model) SaveWinGetDeployInfo(data nats.DeployAction) error {

	exists, err := m.Client.Deployment.Query().Where(deployment.PackageID(data.PackageId), deployment.HasOwnerWith(agent.ID(data.AgentId))).Exist(context.Background())
	if err != nil {
		return err
	}

	if !exists {
		if data.Action == "install" {
			return m.Client.Deployment.Create().
				SetOwnerID(data.AgentId).
				SetPackageID(data.PackageId).
				SetName(strings.TrimPrefix(data.PackageName, "Install ")).
				SetInstalled(data.When).
				SetUpdated(data.When).
				SetByProfile(true).
				Exec(context.Background())
		}
	} else {
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
	}

	return nil
}

func (m *Model) GetDeployedPackages(agentID string) ([]string, error) {
	return m.Client.Deployment.Query().Where(deployment.HasOwnerWith(agent.ID(agentID))).Select(wingetconfigexclusion.FieldPackageID).Strings(context.Background())
}

func (m *Model) GetExcludedWinGetPackages(agentID string) ([]string, error) {
	return m.Client.WingetConfigExclusion.Query().Where(wingetconfigexclusion.HasOwnerWith(agent.ID(agentID))).Select(wingetconfigexclusion.FieldPackageID).Strings(context.Background())
}

func (m *Model) MarkPackageAsExcluded(data nats.DeployAction) error {
	_, err := m.Client.Deployment.Delete().Where(deployment.PackageID(data.PackageId), deployment.HasOwnerWith(agent.ID(data.AgentId))).Exec(context.Background())
	if err != nil {
		log.Printf("[ERROR]: could not delete entry for package %s and agent %s", data.PackageId, data.AgentId)
	}

	exists, err := m.Client.WingetConfigExclusion.Query().Where(wingetconfigexclusion.PackageID(data.PackageId), wingetconfigexclusion.HasOwnerWith(agent.ID(data.AgentId))).Exist(context.Background())
	if err != nil {
		log.Printf("[ERROR]: could not check if  entry for package %s and agent %s", data.PackageId, data.AgentId)
	}

	if !exists {
		return m.Client.WingetConfigExclusion.Create().SetPackageID(data.PackageId).SetOwnerID(data.AgentId).Exec(context.Background())
	}

	return nil
}
