package models

import (
	"context"
	"log"
	"time"

	"github.com/doncicuto/openuem_ent/agent"
	"github.com/doncicuto/openuem_ent/antivirus"
	"github.com/doncicuto/openuem_ent/app"
	"github.com/doncicuto/openuem_ent/computer"
	"github.com/doncicuto/openuem_ent/logicaldisk"
	"github.com/doncicuto/openuem_ent/monitor"
	"github.com/doncicuto/openuem_ent/networkadapter"
	"github.com/doncicuto/openuem_ent/operatingsystem"
	"github.com/doncicuto/openuem_ent/printer"
	"github.com/doncicuto/openuem_ent/share"
	"github.com/doncicuto/openuem_ent/systemupdate"
	"github.com/doncicuto/openuem_ent/update"
	"github.com/doncicuto/openuem_nats"
)

func (m *Model) SaveAgentInfo(data *openuem_nats.AgentReport) error {
	ctx := context.Background()
	exists, err := m.Client.Agent.Query().Where(agent.ID(data.AgentID)).Exist(ctx)
	if err != nil {
		return err
	}

	query := m.Client.Agent.Create().
		SetID(data.AgentID).
		SetOs(data.OS).
		SetHostname(data.Hostname).
		SetVersion(data.Version).
		SetEnabled(true).
		SetIP(data.IP).
		SetMAC(data.MACAddress).
		SetVnc(data.SupportedVNCServer).
		SetUpdateTaskExecution(data.LastUpdateTaskExecutionTime).
		SetUpdateTaskResult(data.LastUpdateTaskResult).
		SetUpdateTaskStatus(data.LastUpdateTaskStatus)

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
		log.Printf("could not delete previous apps information: %s", err.Error())
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
		log.Printf("could not delete previous monitors information: %s", err.Error())
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
		log.Printf("could not delete previous logical disks information: %s", err.Error())
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
		log.Printf("could not delete previous printers information: %s", err.Error())
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
		log.Printf("could not delete previous network adapters information: %s", err.Error())
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
		log.Printf("could not delete previous shares information: %s", err.Error())
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
		log.Printf("could not delete previous updates information: %s", err.Error())
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
