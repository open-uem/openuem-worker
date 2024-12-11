package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/doncicuto/openuem_ent"
	"github.com/doncicuto/openuem_ent/agent"
	"github.com/doncicuto/openuem_ent/antivirus"
	"github.com/doncicuto/openuem_ent/app"
	"github.com/doncicuto/openuem_ent/computer"
	"github.com/doncicuto/openuem_ent/logicaldisk"
	"github.com/doncicuto/openuem_ent/monitor"
	"github.com/doncicuto/openuem_ent/networkadapter"
	"github.com/doncicuto/openuem_ent/operatingsystem"
	"github.com/doncicuto/openuem_ent/printer"
	"github.com/doncicuto/openuem_ent/release"
	"github.com/doncicuto/openuem_ent/settings"
	"github.com/doncicuto/openuem_ent/share"
	"github.com/doncicuto/openuem_ent/systemupdate"
	"github.com/doncicuto/openuem_ent/update"
	"github.com/doncicuto/openuem_nats"
	"github.com/doncicuto/openuem_utils"
)

func (m *Model) SaveAgentInfo(data *openuem_nats.AgentReport) error {
	ctx := context.Background()

	exists := true
	existingAgent, err := m.Client.Agent.Query().Where(agent.ID(data.AgentID)).First(ctx)
	if err != nil {
		if !openuem_ent.IsNotFound(err) {
			return err
		} else {
			exists = false
		}
	}

	query := m.Client.Agent.Create().
		SetID(data.AgentID).
		SetOs(data.OS).
		SetHostname(data.Hostname).
		SetIP(data.IP).
		SetMAC(data.MACAddress).
		SetVnc(data.SupportedVNCServer).
		SetVncProxyPort(data.VNCProxyPort).
		SetSftpPort(data.SFTPPort).
		SetCertificateReady(data.CertificateReady)

	if exists {
		// Status
		if existingAgent.Status != agent.StatusWaitingForAdmission {
			if data.Enabled {
				query.SetStatus(agent.StatusEnabled)
			} else {
				query.SetStatus(agent.StatusDisabled)
			}
		}

		// Check update task
		query.SetUpdateTaskDescription(existingAgent.UpdateTaskDescription)
		if data.LastUpdateTaskExecutionTime.After(existingAgent.UpdateTaskExecution) {
			query.SetUpdateTaskExecution(data.LastUpdateTaskExecutionTime)
			if existingAgent.UpdateTaskVersion == data.Release.Version {
				if data.LastUpdateTaskStatus == "admin.update.agents.task_status_success" {
					query.SetUpdateTaskStatus(openuem_nats.UPDATE_SUCCESS)
					query.SetUpdateTaskResult("")
				}

				if data.LastUpdateTaskStatus == "admin.update.agents.task_status_error" {
					query.SetUpdateTaskStatus(openuem_nats.UPDATE_ERROR)
					query.SetUpdateTaskResult(data.LastUpdateTaskResult)
				}

			} else {
				query.SetUpdateTaskStatus(openuem_nats.UPDATE_ERROR)
				if data.LastUpdateTaskResult != "" {
					query.SetUpdateTaskResult(data.LastUpdateTaskResult)
				}
			}
			query.SetUpdateTaskVersion("")
		} else {
			query.SetUpdateTaskExecution(existingAgent.UpdateTaskExecution).SetUpdateTaskResult(existingAgent.UpdateTaskResult).SetUpdateTaskStatus(existingAgent.UpdateTaskStatus).SetUpdateTaskVersion(existingAgent.UpdateTaskVersion)
		}
	}

	if exists {
		return query.
			SetLastContact(time.Now()).
			OnConflictColumns(agent.FieldID).
			UpdateNewValues().
			Exec(context.Background())
	} else {
		return query.
			SetFirstContact(time.Now()).
			SetLastContact(time.Now()).
			OnConflictColumns(agent.FieldID).
			UpdateNewValues().
			Exec(context.Background())
	}
}

func (m *Model) SaveComputerInfo(data *openuem_nats.AgentReport) error {
	return m.Client.Computer.
		Create().
		SetManufacturer(data.Computer.Manufacturer).
		SetModel(data.Computer.Model).
		SetSerial(data.Computer.Serial).
		SetMemory(data.Computer.Memory).
		SetProcessor(data.Computer.Processor).
		SetProcessorArch(data.Computer.ProcessorArch).
		SetProcessorCores(data.Computer.ProcessorCores).
		SetOwnerID(data.AgentID).
		OnConflictColumns(computer.OwnerColumn).
		UpdateNewValues().
		Exec(context.Background())
}

func (m *Model) SaveOSInfo(data *openuem_nats.AgentReport) error {
	return m.Client.OperatingSystem.
		Create().
		SetType(data.OS).
		SetVersion(data.OperatingSystem.Version).
		SetDescription(data.OperatingSystem.Description).
		SetEdition(data.OperatingSystem.Edition).
		SetInstallDate(data.OperatingSystem.InstallDate).
		SetArch(data.OperatingSystem.Arch).
		SetUsername(data.OperatingSystem.Username).
		SetLastBootupTime(data.OperatingSystem.LastBootUpTime).
		SetOwnerID(data.AgentID).
		OnConflictColumns(operatingsystem.OwnerColumn).
		UpdateNewValues().
		Exec(context.Background())
}

func (m *Model) SaveAntivirusInfo(data *openuem_nats.AgentReport) error {
	return m.Client.Antivirus.
		Create().
		SetName(data.Antivirus.Name).
		SetIsActive(data.Antivirus.IsActive).
		SetIsUpdated(data.Antivirus.IsUpdated).
		SetOwnerID(data.AgentID).
		OnConflictColumns(antivirus.OwnerColumn).
		UpdateNewValues().
		Exec(context.Background())
}

func (m *Model) SaveSystemUpdateInfo(data *openuem_nats.AgentReport) error {
	return m.Client.SystemUpdate.
		Create().
		SetStatus(data.SystemUpdate.Status).
		SetLastInstall(data.SystemUpdate.LastInstall).
		SetLastSearch(data.SystemUpdate.LastSearch).
		SetPendingUpdates(data.SystemUpdate.PendingUpdates).
		SetOwnerID(data.AgentID).
		OnConflictColumns(systemupdate.OwnerColumn).
		UpdateNewValues().
		Exec(context.Background())
}

func (m *Model) SaveAppsInfo(data *openuem_nats.AgentReport) error {
	ctx := context.Background()

	tx, err := m.Client.Tx(ctx)
	if err != nil {
		return err
	}

	_, err = tx.App.Delete().Where(app.HasOwnerWith(agent.ID(data.AgentID))).Exec(ctx)
	if err != nil {
		log.Printf("could not delete previous apps information: %v", err)
		return tx.Rollback()
	}

	for _, appData := range data.Applications {
		if err := tx.App.
			Create().
			SetName(appData.Name).
			SetVersion(appData.Version).
			SetPublisher(appData.Publisher).
			SetInstallDate(appData.InstallDate).
			SetOwnerID(data.AgentID).
			Exec(ctx); err != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (m *Model) SaveMonitorsInfo(data *openuem_nats.AgentReport) error {
	ctx := context.Background()

	tx, err := m.Client.Tx(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Monitor.Delete().Where(monitor.HasOwnerWith(agent.ID(data.AgentID))).Exec(ctx)
	if err != nil {
		log.Printf("could not delete previous monitors information: %v", err)
		return tx.Rollback()
	}

	for _, monitorData := range data.Monitors {
		if err := tx.Monitor.
			Create().
			SetManufacturer(monitorData.Manufacturer).
			SetModel(monitorData.Model).
			SetSerial(monitorData.Serial).
			SetOwnerID(data.AgentID).
			Exec(ctx); err != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (m *Model) SaveLogicalDisksInfo(data *openuem_nats.AgentReport) error {
	ctx := context.Background()

	tx, err := m.Client.Tx(ctx)
	if err != nil {
		return err
	}

	_, err = tx.LogicalDisk.Delete().Where(logicaldisk.HasOwnerWith(agent.ID(data.AgentID))).Exec(ctx)
	if err != nil {
		log.Printf("could not delete previous logical disks information: %v", err)
		return tx.Rollback()
	}

	for _, driveData := range data.LogicalDisks {
		if err := tx.LogicalDisk.
			Create().
			SetLabel(driveData.Label).
			SetUsage(driveData.Usage).
			SetSizeInUnits(driveData.SizeInUnits).
			SetFilesystem(driveData.Filesystem).
			SetRemainingSpaceInUnits(driveData.RemainingSpaceInUnits).
			SetBitlockerStatus(driveData.BitLockerStatus).
			SetOwnerID(data.AgentID).
			Exec(ctx); err != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (m *Model) SavePrintersInfo(data *openuem_nats.AgentReport) error {
	ctx := context.Background()

	tx, err := m.Client.Tx(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Printer.Delete().Where(printer.HasOwnerWith(agent.ID(data.AgentID))).Exec(ctx)
	if err != nil {
		log.Printf("could not delete previous printers information: %v", err)
		return tx.Rollback()
	}

	for _, printerData := range data.Printers {
		if err := tx.Printer.
			Create().
			SetName(printerData.Name).
			SetPort(printerData.Port).
			SetIsDefault(printerData.IsDefault).
			SetIsNetwork(printerData.IsNetwork).
			SetOwnerID(data.AgentID).
			Exec(ctx); err != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (m *Model) SaveNetworkAdaptersInfo(data *openuem_nats.AgentReport) error {
	ctx := context.Background()

	tx, err := m.Client.Tx(ctx)
	if err != nil {
		return err
	}

	_, err = tx.NetworkAdapter.Delete().Where(networkadapter.HasOwnerWith(agent.ID(data.AgentID))).Exec(ctx)
	if err != nil {
		log.Printf("could not delete previous network adapters information: %v", err)
		return tx.Rollback()
	}

	for _, networkAdapterData := range data.NetworkAdapters {
		if err := tx.NetworkAdapter.
			Create().
			SetName(networkAdapterData.Name).
			SetMACAddress(networkAdapterData.MACAddress).
			SetAddresses(networkAdapterData.Addresses).
			SetSubnet(networkAdapterData.Subnet).
			SetDNSDomain(networkAdapterData.DNSDomain).
			SetDNSServers(networkAdapterData.DNSServers).
			SetDefaultGateway(networkAdapterData.DefaultGateway).
			SetDhcpEnabled(networkAdapterData.DHCPEnabled).
			SetDhcpLeaseExpired(networkAdapterData.DHCPLeaseExpired).
			SetDhcpLeaseObtained(networkAdapterData.DHCPLeaseObtained).
			SetSpeed(networkAdapterData.Speed).
			SetOwnerID(data.AgentID).
			Exec(ctx); err != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (m *Model) SaveSharesInfo(data *openuem_nats.AgentReport) error {
	ctx := context.Background()

	tx, err := m.Client.Tx(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Share.Delete().Where(share.HasOwnerWith(agent.ID(data.AgentID))).Exec(ctx)
	if err != nil {
		log.Printf("could not delete previous shares information: %v", err)
		return tx.Rollback()
	}

	for _, shareData := range data.Shares {
		if err := tx.Share.
			Create().
			SetName(shareData.Name).
			SetDescription(shareData.Description).
			SetPath(shareData.Path).
			SetOwnerID(data.AgentID).
			Exec(ctx); err != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (m *Model) SaveUpdatesInfo(data *openuem_nats.AgentReport) error {
	ctx := context.Background()

	tx, err := m.Client.Tx(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Update.Delete().Where(update.HasOwnerWith(agent.ID(data.AgentID))).Exec(ctx)
	if err != nil {
		log.Printf("could not delete previous updates information: %v", err)
		return tx.Rollback()
	}

	for _, updatesData := range data.Updates {
		if err := tx.Update.
			Create().
			SetTitle(updatesData.Title).
			SetDate(updatesData.Date).
			SetSupportURL(updatesData.SupportURL).
			SetOwnerID(data.AgentID).
			Exec(ctx); err != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (m *Model) GetDefaultAgentFrequency() (int, error) {
	var err error

	settings, err := m.Client.Settings.Query().Select(settings.FieldAgentReportFrequenceInMinutes).Only(context.Background())
	if err != nil {
		return 0, err
	}

	return settings.AgentReportFrequenceInMinutes, nil
}

func (m *Model) SaveReleaseInfo(data *openuem_nats.AgentReport) error {
	var err error
	var r *openuem_ent.Release
	releaseExists := false

	r, err = m.Client.Release.Query().
		Where(release.ReleaseTypeEQ(release.ReleaseTypeAgent), release.Version(data.Release.Version), release.Channel(data.Release.Channel), release.Os(data.Release.Os), release.Arch(data.Release.Arch)).
		Only(context.Background())

	// First check if the release is in our database
	if err != nil {
		if !openuem_ent.IsNotFound(err) {
			return err
		}
	} else {
		releaseExists = true
	}

	// Get release info from API
	url := fmt.Sprintf("https://releases.openuem.eu/api?action=agentReleaseInfo&version=%s", data.Release.Version)

	body, err := openuem_utils.QueryReleasesEndpoint(url)
	if err != nil {
		return err
	}

	releaseFromApi := openuem_nats.OpenUEMRelease{}
	if err := json.Unmarshal(body, &releaseFromApi); err != nil {
		return err
	}

	fileURL := ""
	checksum := ""

	for _, item := range releaseFromApi.Files {
		if item.Arch == data.Release.Arch && item.Os == data.Release.Os {
			fileURL = item.FileURL
			checksum = item.Checksum
			break
		}
	}

	// If not exists add it
	if !releaseExists {
		r, err = m.Client.Release.Create().
			SetReleaseType(release.ReleaseTypeAgent).
			SetVersion(data.Release.Version).
			SetChannel(releaseFromApi.Channel).
			SetSummary(releaseFromApi.Summary).
			SetFileURL(fileURL).
			SetReleaseNotes(releaseFromApi.ReleaseNotesURL).
			SetChecksum(checksum).
			SetIsCritical(releaseFromApi.IsCritical).
			SetReleaseDate(releaseFromApi.ReleaseDate).
			SetArch(data.Release.Arch).
			SetOs(data.Release.Os).
			AddAgentIDs(data.AgentID).
			Save(context.Background())
		if err != nil {
			return err
		}
	} else {
		if err := m.Client.Release.UpdateOneID(r.ID).AddAgentIDs(data.AgentID).Exec(context.Background()); err != nil {
			return err
		}
	}

	// Finally connect the release with the agent
	return m.Client.Agent.Update().Where(agent.ID(data.AgentID)).SetReleaseID(r.ID).Exec(context.Background())
}

func (m *Model) SetAgentIsWaitingForAdmissionAgain(agentId string) error {
	return m.Client.Agent.Update().SetStatus(agent.StatusWaitingForAdmission).Where(agent.ID(agentId)).Exec(context.Background())
}
