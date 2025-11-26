package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"slices"
	"strconv"
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
	"github.com/open-uem/ent/physicaldisk"
	"github.com/open-uem/ent/printer"
	"github.com/open-uem/ent/release"
	"github.com/open-uem/ent/settings"
	"github.com/open-uem/ent/share"
	"github.com/open-uem/ent/site"
	"github.com/open-uem/ent/systemupdate"
	"github.com/open-uem/ent/tenant"
	"github.com/open-uem/ent/update"
	"github.com/open-uem/nats"
	"github.com/open-uem/utils"
)

func (m *Model) SaveAgentInfo(data *nats.AgentReport, servers string, autoAdmitAgents bool) error {
	ctx := context.Background()

	exists := true
	existingAgent, err := m.Client.Agent.Query().WithSite().Where(agent.ID(data.AgentID)).First(ctx)
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
		SetRemoteAssistance(!data.RemoteAssistanceDisabled).
		SetHasRustdesk(data.HasRustDesk).
		SetIsFlatpakRustdesk(data.IsFlatpakRustDesk).
		SetIsWayland(data.IsWayland)

	if exists {
		// Status
		if existingAgent.AgentStatus != agent.AgentStatusWaitingForAdmission {
			if data.Enabled {
				query.SetAgentStatus(agent.AgentStatusEnabled)
			} else {
				query.SetAgentStatus(agent.AgentStatusDisabled)
			}
		}

		// nickname, type and description must not be overwritten
		query.SetNickname(existingAgent.Nickname).SetEndpointType(existingAgent.EndpointType).SetDescription(existingAgent.Description)

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
		// Check if we must add the site, only if no site has been assigned yet
		associatedSites := existingAgent.Edges.Site
		if len(associatedSites) > 1 {
			log.Println("[ERROR]: agent cannot be associated to two or more sites")
			return fmt.Errorf("agent cannot be associated to two or more sites")
		}

		if len(associatedSites) == 0 {
			if data.Site == "" {
				s, err := m.GetDefaultSite()
				if err != nil {
					log.Printf("[ERROR]: could not get default site, reason: %v", err)
					return err
				}
				query.AddSite(s)
			} else {
				tenantID, err := strconv.Atoi(data.Tenant)
				if err != nil {
					log.Printf("[ERROR]: could not convert tenant ID to int, reason: %v", err)
					return err
				}

				siteID, err := strconv.Atoi(data.Site)
				if err != nil {
					log.Printf("[ERROR]: could not convert site ID to int, reason: %v", err)
					return err
				}

				// Check if tenantID is right and associated with the site
				valid, err := m.ValidateTenantAndSite(tenantID, siteID)
				if err != nil {
					log.Printf("[ERROR]: could not check if tenant and site are valid, reason: %v", err)
					return err
				}

				if valid {
					query.AddSiteIDs(siteID)
				} else {
					log.Printf("[ERROR]: tenant and site are not valid")
					return errors.New("tenant and site are not valid")
				}
			}
		}

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

		// This is a new agent, we must set the nickname to the agent's hostname initially
		query.SetNickname(data.Hostname)

		// Set the associated site
		if data.Site == "" {
			s, err := m.GetDefaultSite()
			if err != nil {
				log.Printf("[ERROR]: could not get default site, reason: %v", err)
				return err
			}
			query.AddSite(s)
		} else {
			siteID, err := strconv.Atoi(data.Site)
			if err != nil {
				log.Printf("[ERROR]: could not convert site ID to int, reason: %v", err)
				return err
			}

			tenantID, err := strconv.Atoi(data.Tenant)
			if err != nil {
				log.Printf("[ERROR]: could not convert tenant ID to int, reason: %v", err)
				return err
			}

			// Check if tenantID is right and associated with the site
			valid, err := m.ValidateTenantAndSite(tenantID, siteID)
			if err != nil {
				log.Printf("[ERROR]: could not check if tenant and site are valid, reason: %v", err)
				return err
			}

			if valid {
				query.AddSiteIDs(siteID)
			} else {
				log.Printf("[ERROR]: tenant and site are not valid")
				return errors.New("tenant and site are not valid")
			}
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
		SetDomain(data.OperatingSystem.Domain).
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

func (m *Model) SavePhysicalDisksInfo(data *nats.AgentReport) error {
	ctx := context.Background()

	tx, err := m.Client.Tx(ctx)
	if err != nil {
		return err
	}

	_, err = tx.PhysicalDisk.Delete().Where(physicaldisk.HasOwnerWith(agent.ID(data.AgentID))).Exec(ctx)
	if err != nil {
		log.Printf("could not delete previous physical disks information: %v", err)
		return tx.Rollback()
	}

	for _, driveData := range data.PhysicalDisks {
		if err := tx.PhysicalDisk.
			Create().
			SetDeviceID(driveData.DeviceID).
			SetModel(driveData.Model).
			SetSerialNumber(driveData.SerialNumber).
			SetSizeInUnits(driveData.SizeInUnits).
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

func (m *Model) GetDefaultAgentFrequency(request nats.RemoteConfigRequest) (int, error) {
	var err error

	tenantID, err := m.GetTenantFromAgentID(request)
	if err != nil {
		settings, err := m.Client.Settings.Query().Where(settings.Not(settings.HasTenant())).Select(settings.FieldAgentReportFrequenceInMinutes).Only(context.Background())
		if err != nil {
			return 0, err
		}
		return settings.AgentReportFrequenceInMinutes, nil
	}

	settings, err := m.Client.Settings.Query().Where(settings.HasTenantWith(tenant.ID(tenantID))).Select(settings.FieldAgentReportFrequenceInMinutes).Only(context.Background())
	if err != nil {
		return 0, err
	}

	return settings.AgentReportFrequenceInMinutes, nil
}

func (m *Model) GetWingetFrequency(request nats.RemoteConfigRequest) (int, error) {
	var err error

	tenantID, err := m.GetTenantFromAgentID(request)
	if err != nil {
		settings, err := m.Client.Settings.Query().Where(settings.Not(settings.HasTenant())).Select(settings.FieldProfilesApplicationFrequenceInMinutes).Only(context.Background())
		if err != nil {
			return 0, err
		}

		return settings.ProfilesApplicationFrequenceInMinutes, nil
	}

	settings, err := m.Client.Settings.Query().Where(settings.HasTenantWith(tenant.ID(tenantID))).Select(settings.FieldProfilesApplicationFrequenceInMinutes).Only(context.Background())
	if err != nil {
		return 0, err
	}

	return settings.ProfilesApplicationFrequenceInMinutes, nil
}

func (m *Model) GetSFTPAgentSetting(request nats.RemoteConfigRequest) (bool, error) {
	agent, err := m.Client.Agent.Query().Select(agent.FieldSftpService).Where(agent.ID(request.AgentID)).First(context.Background())
	if err != nil {
		tenantID, err := m.GetTenantFromAgentID(request)
		if err != nil {
			settings, err := m.Client.Settings.Query().Where(settings.Not(settings.HasTenant())).Select(settings.FieldProfilesApplicationFrequenceInMinutes).Only(context.Background())
			if err != nil {
				return false, err
			}

			return !settings.DisableSftp, nil
		}

		settings, err := m.Client.Settings.Query().Where(settings.HasTenantWith(tenant.ID(tenantID))).Select(settings.FieldProfilesApplicationFrequenceInMinutes).Only(context.Background())
		if err != nil {
			return false, err
		}
		return !settings.DisableSftp, nil
	}

	return agent.SftpService, nil
}

func (m *Model) SaveSFTPAgentSetting(request nats.RemoteConfigRequest, status bool) error {
	return m.Client.Agent.UpdateOneID(request.AgentID).SetSftpService(status).Exec(context.Background())
}

func (m *Model) GetRemoteAssistanceAgentSetting(request nats.RemoteConfigRequest) (bool, error) {
	agent, err := m.Client.Agent.Query().Select(agent.FieldRemoteAssistance).Where(agent.ID(request.AgentID)).First(context.Background())
	if err != nil {
		tenantID, err := m.GetTenantFromAgentID(request)
		if err != nil {
			settings, err := m.Client.Settings.Query().Where(settings.Not(settings.HasTenant())).Select(settings.FieldProfilesApplicationFrequenceInMinutes).Only(context.Background())
			if err != nil {
				return false, err
			}

			return !settings.DisableRemoteAssistance, nil
		}

		settings, err := m.Client.Settings.Query().Where(settings.HasTenantWith(tenant.ID(tenantID))).Select(settings.FieldProfilesApplicationFrequenceInMinutes).Only(context.Background())
		if err != nil {
			return false, err
		}
		return !settings.DisableRemoteAssistance, nil
	}

	return agent.RemoteAssistance, nil
}

func (m *Model) SaveRemoteAssistanceAgentSetting(request nats.RemoteConfigRequest, status bool) error {
	return m.Client.Agent.UpdateOneID(request.AgentID).SetRemoteAssistance(status).Exec(context.Background())
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
		return false
	}

	// If the agent's IP address is not contained in the addresses list resolved by DNS, the agent is in a remote location
	return !slices.Contains(addresses, data.IP)
}

func (m *Model) GetTenantFromAgentID(request nats.RemoteConfigRequest) (int, error) {

	a, err := m.Client.Agent.Query().WithSite().Where(agent.ID(request.AgentID)).Only(context.Background())
	if err != nil {
		if request.TenantID != "" {
			return 0, err
		}
		return strconv.Atoi(request.TenantID)
	}

	sites := a.Edges.Site
	if len(sites) != 1 {
		return 0, fmt.Errorf("the agent should belong to only one site")
	}

	s, err := m.Client.Site.Query().WithTenant().Where(site.ID(sites[0].ID)).Only(context.Background())
	if err != nil {
		return 0, err
	}

	t := s.Edges.Tenant
	if t == nil {
		return 0, fmt.Errorf("could not find tenant associated with agent's site")
	}

	return t.ID, nil
}

func (m *Model) GetDefaultTenant() (*ent.Tenant, error) {
	return m.Client.Tenant.Query().Where(tenant.IsDefault(true)).Only(context.Background())
}

func (m *Model) GetDefaultSite() (*ent.Site, error) {
	t, err := m.GetDefaultTenant()
	if err != nil {
		return nil, err
	}

	return m.Client.Site.Query().Where(site.IsDefault(true), site.HasTenantWith(tenant.ID(t.ID))).Only(context.Background())
}

func (m *Model) ValidateTenantAndSite(tenantID, siteID int) (bool, error) {
	return m.Client.Site.Query().Where(site.ID(siteID), site.HasTenantWith(tenant.ID(tenantID))).Exist(context.Background())
}

func (m *Model) GetAgentApps(agentId string) ([]*ent.App, error) {
	return m.Client.App.Query().Where(app.HasOwnerWith(agent.ID(agentId), agent.AgentStatusNEQ(agent.AgentStatusWaitingForAdmission))).All(context.Background())
}
