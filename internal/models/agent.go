package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/open-uem/ent"
	"github.com/open-uem/ent/agent"
	"github.com/open-uem/ent/antivirus"
	"github.com/open-uem/ent/app"
	"github.com/open-uem/ent/computer"
	"github.com/open-uem/ent/logicaldisk"
	"github.com/open-uem/ent/memoryslot"
	"github.com/open-uem/ent/monitor"
	"github.com/open-uem/ent/networkadapter"
	"github.com/open-uem/ent/operatingsystem"
	"github.com/open-uem/ent/printer"
	"github.com/open-uem/ent/release"
	"github.com/open-uem/ent/settings"
	"github.com/open-uem/ent/share"
	"github.com/open-uem/ent/systemupdate"
	"github.com/open-uem/ent/update"
	"github.com/open-uem/nats"
	"github.com/open-uem/utils"
)

func (m *Model) SaveAgentInfo(data *nats.AgentReport, servers string, autoAdmitAgents bool) error {
	ctx := context.Background()

	exists := true
	existingAgent, err := m.Client.Agent.Query().Where(agent.ID(data.AgentID)).First(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return err
		} else {
			exists = false
		}
	}

	isRemoteAgent := checkIfRemote(data, servers)

	query := m.Client.Agent.Create().
		SetID(data.AgentID).
		SetOs(data.OS).
		SetHostname(data.Hostname).
		SetIP(data.IP).
		SetMAC(data.MACAddress).
		SetVnc(data.SupportedVNCServer).
		SetVncProxyPort(data.VNCProxyPort).
		SetSftpPort(data.SFTPPort).
		SetCertificateReady(data.CertificateReady).
		SetDebugMode(data.DebugMode).
		SetIsRemote(isRemoteAgent).
		SetSftpService(!data.SftpServiceDisabled).
		SetRemoteAssistance(!data.RemoteAssistanceDisabled)

	if exists {
		// Status
		if existingAgent.AgentStatus != agent.AgentStatusWaitingForAdmission {
			if data.Enabled {
				query.SetAgentStatus(agent.AgentStatusEnabled)
			} else {
				query.SetAgentStatus(agent.AgentStatusDisabled)
			}
		}

		// Check update task
		query.SetUpdateTaskDescription(existingAgent.UpdateTaskDescription)
		if data.LastUpdateTaskExecutionTime.After(existingAgent.UpdateTaskExecution) {
			query.SetUpdateTaskExecution(data.LastUpdateTaskExecutionTime)
			if existingAgent.UpdateTaskVersion == data.Release.Version {
				if data.LastUpdateTaskStatus == "admin.update.agents.task_status_success" {
					query.SetUpdateTaskStatus(nats.UPDATE_SUCCESS)
					query.SetUpdateTaskResult("")
				}

				if data.LastUpdateTaskStatus == "admin.update.agents.task_status_error" {
					query.SetUpdateTaskStatus(nats.UPDATE_ERROR)
					query.SetUpdateTaskResult(data.LastUpdateTaskResult)
				}

			} else {
				query.SetUpdateTaskStatus(nats.UPDATE_ERROR)
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
		// This is a new agent, we must create a record and set enabled if auto admit agents is enabled
		if autoAdmitAgents {
			query.SetAgentStatus(agent.AgentStatusEnabled)
		}

		return query.
			SetFirstContact(time.Now()).
			SetLastContact(time.Now()).
			OnConflictColumns(agent.FieldID).
			UpdateNewValues().
			Exec(context.Background())
	}
}

func (m *Model) SaveComputerInfo(data *nats.AgentReport) error {
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

func (m *Model) SaveOSInfo(data *nats.AgentReport) error {
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

func (m *Model) SaveAntivirusInfo(data *nats.AgentReport) error {
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

func (m *Model) SaveSystemUpdateInfo(data *nats.AgentReport) error {
	return m.Client.SystemUpdate.
		Create().
		SetSystemUpdateStatus(data.SystemUpdate.Status).
		SetLastInstall(data.SystemUpdate.LastInstall).
		SetLastSearch(data.SystemUpdate.LastSearch).
		SetPendingUpdates(data.SystemUpdate.PendingUpdates).
		SetOwnerID(data.AgentID).
		OnConflictColumns(systemupdate.OwnerColumn).
		UpdateNewValues().
		Exec(context.Background())
}

func (m *Model) SaveAppsInfo(data *nats.AgentReport) error {
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

func (m *Model) SaveMonitorsInfo(data *nats.AgentReport) error {
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
			SetWeekOfManufacture(monitorData.WeekOfManufacture).
			SetYearOfManufacture(monitorData.YearOfManufacture).
			Exec(ctx); err != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (m *Model) SaveMemorySlotsInfo(data *nats.AgentReport) error {
	ctx := context.Background()

	tx, err := m.Client.Tx(ctx)
	if err != nil {
		return err
	}

	_, err = tx.MemorySlot.Delete().Where(memoryslot.HasOwnerWith(agent.ID(data.AgentID))).Exec(ctx)
	if err != nil {
		log.Printf("could not delete previous memory slots information: %v", err)
		return tx.Rollback()
	}

	for _, slotsData := range data.MemorySlots {
		if err := tx.MemorySlot.
			Create().
			SetSlot(slotsData.Slot).
			SetType(slotsData.MemoryType).
			SetPartNumber(slotsData.PartNumber).
			SetSerialNumber(slotsData.SerialNumber).
			SetSize(slotsData.Size).
			SetSpeed(slotsData.Speed).
			SetManufacturer(slotsData.Manufacturer).
			SetOwnerID(data.AgentID).
			Exec(ctx); err != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (m *Model) SaveLogicalDisksInfo(data *nats.AgentReport) error {
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
			SetVolumeName(driveData.VolumeName).
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

func (m *Model) SavePrintersInfo(data *nats.AgentReport) error {
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
			SetIsShared(printerData.IsShared).
			SetOwnerID(data.AgentID).
			Exec(ctx); err != nil {
			return tx.Rollback()
		}
	}

	return tx.Commit()
}

func (m *Model) SaveNetworkAdaptersInfo(data *nats.AgentReport) error {
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

func (m *Model) SaveSharesInfo(data *nats.AgentReport) error {
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

func (m *Model) SaveUpdatesInfo(data *nats.AgentReport) error {
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

func (m *Model) GetWingetFrequency() (int, error) {
	var err error

	settings, err := m.Client.Settings.Query().Select(settings.FieldProfilesApplicationFrequenceInMinutes).Only(context.Background())
	if err != nil {
		return 0, err
	}

	return settings.ProfilesApplicationFrequenceInMinutes, nil
}

func (m *Model) GetSFTPDisabledSettings() (bool, error) {
	var err error

	settings, err := m.Client.Settings.Query().Select(settings.FieldDisableSftp).Only(context.Background())
	if err != nil {
		return false, err
	}

	return settings.DisableSftp, nil
}

func (m *Model) GetSFTPAgentSetting(agentID string) (bool, error) {
	agent, err := m.Client.Agent.Query().Select(agent.FieldSftpService).Where(agent.ID(agentID)).First(context.Background())
	if err != nil {
		return false, err
	}

	return agent.SftpService, nil
}

func (m *Model) SaveSFTPAgentSetting(agentID string, status bool) error {
	return m.Client.Agent.UpdateOneID(agentID).SetSftpService(status).Exec(context.Background())
}

func (m *Model) GetRemoteAssistanceDisabledSettings() (bool, error) {
	var err error

	settings, err := m.Client.Settings.Query().Select(settings.FieldDisableRemoteAssistance).Only(context.Background())
	if err != nil {
		return false, err
	}

	return settings.DisableRemoteAssistance, nil
}

func (m *Model) GetRemoteAssistanceAgentSetting(agentID string) (bool, error) {
	agent, err := m.Client.Agent.Query().Select(agent.FieldRemoteAssistance).Where(agent.ID(agentID)).First(context.Background())
	if err != nil {
		return false, err
	}

	return agent.RemoteAssistance, nil
}

func (m *Model) SaveRemoteAssistanceAgentSetting(agentID string, status bool) error {
	return m.Client.Agent.UpdateOneID(agentID).SetRemoteAssistance(status).Exec(context.Background())
}

func (m *Model) SaveReleaseInfo(data *nats.AgentReport) error {
	var err error
	var r *ent.Release
	releaseExists := false

	r, err = m.Client.Release.Query().
		WithAgents().
		Where(release.ReleaseTypeEQ(release.ReleaseTypeAgent), release.Version(data.Release.Version), release.Channel(data.Release.Channel), release.Os(data.Release.Os), release.Arch(data.Release.Arch)).
		Only(context.Background())

	// First check if the release is in our database
	if err != nil {
		if !ent.IsNotFound(err) {
			return err
		}
	} else {
		releaseExists = true
	}

	// If not exists add it
	if !releaseExists {
		// Get release info from API
		url := fmt.Sprintf("https://releases.openuem.eu/api?action=agentReleaseInfo&version=%s", data.Release.Version)

		body, err := utils.QueryReleasesEndpoint(url)
		if err != nil {
			return err
		}

		releaseFromApi := nats.OpenUEMRelease{}
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
		newAgent := true
		for _, a := range r.Edges.Agents {
			if a.ID == data.AgentID {
				newAgent = false
				break
			}
		}

		// Finally connect the release with the agent if new, and disconnect from previous release
		if newAgent {

			existingAgent, err := m.Client.Agent.Query().WithRelease().Where(agent.ID(data.AgentID)).First(context.Background())
			if err != nil {
				return err
			}

			if existingAgent.Edges.Release != nil {
				previousReleaseID := existingAgent.Edges.Release.ID
				if err := m.Client.Release.UpdateOneID(previousReleaseID).RemoveAgentIDs(data.AgentID).Exec(context.Background()); err != nil {
					return err
				}
			}

			if err := m.Client.Release.UpdateOneID(r.ID).AddAgentIDs(data.AgentID).Exec(context.Background()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Model) SetAgentIsWaitingForAdmissionAgain(agentId string) error {
	return m.Client.Agent.Update().SetAgentStatus(agent.AgentStatusWaitingForAdmission).Where(agent.ID(agentId)).Exec(context.Background())
}

func checkIfRemote(data *nats.AgentReport, servers string) bool {
	// Check if agent's IP is IPv6 and ignore if it is
	ip := net.ParseIP(data.IP)
	if ip == nil {
		return false
	}
	if ip.To4() == nil {
		return false
	}

	// Try to parse the NATS servers to get the domain
	serversHostnames := strings.Split(servers, ",")
	if len(serversHostnames) == 0 {
		return false
	}
	serverDomain := strings.Split(serversHostnames[0], ".")
	if len(serverDomain) < 2 {
		return false
	}
	domain := strings.Split(strings.Replace(serversHostnames[0], serverDomain[0], "", 1), ":")[0]

	// Check if we can find the DNS record for the agent
	addresses, err := net.LookupHost(strings.ToLower(data.Hostname) + domain)
	if err != nil {
		log.Printf("[ERROR]: there was an issue trying to lookup host, reason: %v", err)
		return false
	}

	// If the agent's IP address is not contained in the addresses list resolved by DNS, the agent is in a remote location
	return !slices.Contains(addresses, data.IP)
}
