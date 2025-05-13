package common

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/open-uem/ent"
	"github.com/open-uem/ent/agent"
	"github.com/open-uem/ent/task"
	openuem_nats "github.com/open-uem/nats"
	"github.com/open-uem/wingetcfg/wingetcfg"
	"gopkg.in/yaml.v3"
)

type ProfileConfig struct {
	ProfileID   int                  `yaml:"profileID"`
	Exclusions  []string             `yaml:"exclusions"`
	Deployments []string             `yaml:"deployments"`
	Config      *wingetcfg.WinGetCfg `yaml:"config"`
}

func (w *Worker) SubscribeToAgentWorkerQueues() error {
	_, err := w.NATSConnection.QueueSubscribe("report", "openuem-agents", w.ReportReceivedHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to report NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message report")

	_, err = w.NATSConnection.QueueSubscribe("deployresult", "openuem-agents", w.DeployResultReceivedHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to deployresult NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message deployresult")

	_, err = w.NATSConnection.QueueSubscribe("ping.agentworker", "openuem-agents", w.PingHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to ping.agentworker NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message ping.agentworker")

	_, err = w.NATSConnection.QueueSubscribe("agentconfig", "openuem-agents", w.AgentConfigHandler)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to agentconfig NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message agentconfig")

	_, err = w.NATSConnection.QueueSubscribe("wingetcfg.profiles", "openuem-agents", w.ApplyEndpointProfiles)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to wingetcfg.profiles NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message wingetcfg.profiles")

	_, err = w.NATSConnection.QueueSubscribe("wingetcfg.deploy", "openuem-agents", w.WinGetCfgDeploymentReport)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to wingetcfg.deploy NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message wingetcfg.deploy")

	_, err = w.NATSConnection.QueueSubscribe("wingetcfg.exclude", "openuem-agents", w.WinGetCfgMarkPackageAsExcluded)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to wingetcfg.exclude NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message wingetcfg.exclude")

	_, err = w.NATSConnection.QueueSubscribe("wingetcfg.report", "openuem-agents", w.WinGetCfgApplicationReport)
	if err != nil {
		log.Printf("[ERROR]: could not subscribe to wingetcfg.report NATS message, reason: %v", err)
		return err
	}
	log.Printf("[INFO]: subscribed to message wingetcfg.report")
	return nil
}

func (w *Worker) ReportReceivedHandler(msg *nats.Msg) {
	data := openuem_nats.AgentReport{}
	tenantID := ""

	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("[ERROR]: could not unmarshal agent report, reason: %v\n", err)
	}

	requestConfig := openuem_nats.RemoteConfigRequest{
		AgentID:  data.AgentID,
		TenantID: data.Tenant,
		SiteID:   data.Site,
	}

	autoAdmitAgents := false

	// Check if agent exists
	exists, err := w.Model.Client.Agent.Query().Where(agent.ID(data.AgentID)).Exist(context.Background())
	if err != nil {
		log.Printf("[ERROR]: could not check if agent exists, reason: %v\n", err)
	} else {
		if exists {
			id, err := w.Model.GetTenantFromAgentID(requestConfig)
			if err != nil {
				log.Printf("[ERROR]: could not get tenant ID, reason: %v\n", err)
			} else {
				tenantID = strconv.Itoa(id)
			}
		} else {
			tenantID = data.Tenant
		}

		settings, err := w.Model.GetSettings(tenantID)
		if err != nil {
			log.Printf("[ERROR]: could not get OpenUEM general settings, reason: %v\n", err)
		} else {
			autoAdmitAgents = settings.AutoAdmitAgents
		}
	}

	if err := w.Model.SaveAgentInfo(&data, w.NATSServers, autoAdmitAgents); err != nil {
		log.Printf("[ERROR]: could not save agent info into database, reason: %v\n", err.Error())
	}

	if err := w.Model.SaveComputerInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save computer info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveOSInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save operating system info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveAntivirusInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save antivirus info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveSystemUpdateInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save system updates info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveAppsInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save apps info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveMonitorsInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save monitors info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveMemorySlotsInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save memory slots info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveLogicalDisksInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save logical disks info into database, reason: %v\n", err)
	}

	if err := w.Model.SavePrintersInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save printers info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveNetworkAdaptersInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save network adapters info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveSharesInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save shares info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveUpdatesInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save updates info into database, reason: %v\n", err)
	}

	if err := w.Model.SaveReleaseInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save release info into database, reason: %v\n", err)
	}

	if err := msg.Respond([]byte("Report received!")); err != nil {
		log.Printf("[ERROR]: could not respond to report message, reason: %v\n", err)
	}
}

func (w *Worker) DeployResultReceivedHandler(msg *nats.Msg) {
	data := openuem_nats.DeployAction{}

	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("[ERROR]: could not unmarshal deploy message, reason: %v\n", err)
	}

	if err := w.Model.SaveDeployInfo(&data); err != nil {
		log.Printf("[ERROR]: could not save deployment info into database, reason: %v\n", err)

		if err := msg.Respond([]byte(err.Error())); err != nil {
			log.Printf("[ERROR]: could not respond to deploy message, reason: %v\n", err)
		}
		return
	}

	if err := msg.Respond([]byte("")); err != nil {
		log.Printf("[ERROR]: could not respond to deploy message, reason: %v\n", err)
	}
}

func (w *Worker) ApplyEndpointProfiles(msg *nats.Msg) {
	configurations := []ProfileConfig{}
	profileRequest := openuem_nats.WingetCfgProfiles{}

	// log.Println("[DEBUG]: received a wingetcfg.profiles message")

	// Unmarshal data and get agentID
	if err := json.Unmarshal(msg.Data, &profileRequest); err != nil {
		log.Println("[ERROR]: could not unmarshall profile request")
		return
	}

	// Check agentID
	if profileRequest.AgentID == "" {
		log.Println("[ERROR]: agentID must not be empty")
		return
	}

	// log.Println("[DEBUG]: received a wingetcfg.profiles message for: ", profileRequest.AgentID)

	// Get profiles that should apply to this agent
	profiles, err := w.GetAppliedProfiles(profileRequest.AgentID)
	if err != nil {
		log.Printf("[ERROR]: could not get applied profiles, reason: %v", err)
		return
	}

	exclusions, err := w.Model.GetExcludedWinGetPackages(profileRequest.AgentID)
	if err != nil {
		log.Printf("[ERROR]: could not get WinGetCfg packages exclusions, reason: %v", err)
		return
	}

	deployments, err := w.Model.GetDeployedPackages(profileRequest.AgentID)
	if err != nil {
		log.Printf("[ERROR]: could not get deployed packages with WinGet, reason: %v", err)
		return
	}

	// Generate config for each profile to be applied
	for _, profile := range profiles {
		p := ProfileConfig{
			ProfileID:   profile.ID,
			Exclusions:  exclusions,
			Deployments: deployments,
		}

		// Generate WinGet config
		p.Config, err = w.GenerateWinGetConfig(profile)
		if err != nil {
			log.Printf("[ERROR]: could not generate config for profile: %s, reason: %v", profile.Name, err)
			continue
		}

		configurations = append(configurations, p)
	}

	// Send response
	data, err := yaml.Marshal(configurations)
	if err != nil {
		log.Printf("[ERROR]: could not marshal configurations, reason: %v", err)
	}

	// log.Println("[DEBUG]: going to respond wingetcfg.profiles message for: ", profileRequest.AgentID)

	if err := msg.Respond(data); err != nil {
		log.Printf("[ERROR]: could not send wingetcfg message with profiles to the agent, reason: %v\n", err)
	}

	// log.Println("[DEBUG]: should have responded to wingetcfg.profiles message for: ", profileRequest.AgentID)
}

func (w *Worker) GetAppliedProfiles(agentID string) ([]*ent.Profile, error) {

	a, err := w.Model.Client.Agent.Query().WithSite().Where(agent.ID(agentID)).Only(context.Background())
	if err != nil {
		return nil, err
	}

	sites := a.Edges.Site
	if len(sites) != 1 {
		return nil, fmt.Errorf("agent should be associated with only one site")
	}

	profilesAppliedToAll, err := w.Model.GetProfilesAppliedToAll(sites[0].ID)
	if err != nil {
		return nil, err
	}

	profilesAppliedToAgent, err := w.Model.GetProfilesAppliedToAgent(sites[0].ID, agentID)
	if err != nil {
		return nil, err
	}

	return append(profilesAppliedToAll, profilesAppliedToAgent...), nil
}

func (w *Worker) GenerateWinGetConfig(profile *ent.Profile) (*wingetcfg.WinGetCfg, error) {
	if len(profile.Edges.Tasks) == 0 {
		return nil, errors.New("profile has no tasks")
	}

	cfg := wingetcfg.NewWingetCfg()

	idCmp := func(a, b *ent.Task) int {
		return cmp.Compare(a.ID, b.ID)
	}

	slices.SortFunc(profile.Edges.Tasks, idCmp)

	for i, t := range profile.Edges.Tasks {
		switch t.Type {
		case task.TypeWingetInstall:
			installPackage, err := wingetcfg.InstallPackage(fmt.Sprintf("task_%d", i), t.PackageName, t.PackageID, "winget", "", true)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(installPackage)
		case task.TypeWingetDelete:
			uninstallPackage, err := wingetcfg.UninstallPackage(fmt.Sprintf("task_%d", i), t.PackageName, t.PackageID, "winget", "", true)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(uninstallPackage)
		case task.TypeAddRegistryKey:
			registryKey, err := wingetcfg.AddRegistryKey(fmt.Sprintf("task_%d", i), t.Name, t.RegistryKey)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(registryKey)
		case task.TypeRemoveRegistryKey:
			registryKey, err := wingetcfg.RemoveRegistryKey(fmt.Sprintf("task_%d", i), t.Name, t.RegistryKey, t.RegistryForce)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(registryKey)
		case task.TypeUpdateRegistryKeyDefaultValue:
			registryKey, err := wingetcfg.UpdateRegistryKeyDefaultValue(fmt.Sprintf("task_%d", i), t.Name, t.RegistryKey, string(t.RegistryKeyValueType), strings.Split(t.RegistryKeyValueData, "\n"), t.RegistryForce)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(registryKey)
		case task.TypeAddRegistryKeyValue:
			registryKey, err := wingetcfg.AddRegistryValue(fmt.Sprintf("task_%d", i), t.Name, t.RegistryKey, t.RegistryKeyValueName, string(t.RegistryKeyValueType), strings.Split(t.RegistryKeyValueData, "\n"), t.RegistryHex, t.RegistryForce)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(registryKey)
		case task.TypeRemoveRegistryKeyValue:
			registryKey, err := wingetcfg.RemoveRegistryValue(fmt.Sprintf("task_%d", i), t.Name, t.RegistryKey, t.RegistryKeyValueName)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(registryKey)
		case task.TypeAddLocalUser:
			localUser, err := wingetcfg.AddOrModifyLocalUser(fmt.Sprintf("task_%d", i), t.LocalUserUsername, t.LocalUserDescription, t.LocalUserDisable, t.LocalUserFullname, t.LocalUserPassword, t.LocalUserPasswordChangeNotAllowed, t.LocalUserPasswordChangeRequired, t.LocalUserPasswordNeverExpires)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(localUser)
		case task.TypeRemoveLocalUser:
			localUser, err := wingetcfg.RemoveLocalUser(fmt.Sprintf("task_%d", i), t.LocalUserUsername)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(localUser)
		case task.TypeAddLocalGroup:
			localGroup, err := wingetcfg.AddOrModifyLocalGroup(fmt.Sprintf("task_%d", i), t.LocalGroupName, t.LocalGroupDescription, t.LocalGroupMembers)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(localGroup)
		case task.TypeRemoveLocalGroup:
			localGroup, err := wingetcfg.RemoveLocalGroup(fmt.Sprintf("task_%d", i), t.LocalGroupName)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(localGroup)
		case task.TypeAddUsersToLocalGroup:
			localGroup, err := wingetcfg.IncludeMembersToGroup(fmt.Sprintf("task_%d", i), t.LocalGroupName, t.LocalGroupMembersToInclude)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(localGroup)
		case task.TypeRemoveUsersFromLocalGroup:
			localGroup, err := wingetcfg.ExcludeMembersFromGroup(fmt.Sprintf("task_%d", i), t.LocalGroupName, t.LocalGroupMembersToExclude)
			if err != nil {
				return nil, err
			}
			cfg.AddResource(localGroup)
		default:
			return nil, errors.New("task type is not valid")
		}
	}
	return cfg, nil
}

func (w *Worker) WinGetCfgDeploymentReport(msg *nats.Msg) {
	deploy := openuem_nats.DeployAction{}

	// log.Println("[DEBUG]: received a wingetcfg.deploy message")

	// Unmarshal data and get agentID
	if err := json.Unmarshal(msg.Data, &deploy); err != nil {
		log.Println("[ERROR]: could not unmarshall WinGetCfg deployment action report from agent")
	}

	// log.Printf("[DEBUG]: deplou info: %v", deploy)

	if err := w.Model.SaveWinGetDeployInfo(deploy); err != nil {
		log.Printf("[ERROR]: could not save WinGetCfg deployment action report from agent, reason: %v", err)
	}

	if err := msg.Respond(nil); err != nil {
		log.Printf("[ERROR]: could not respond to WinGetCfg deployment action report, reason: %v\n", err)
	}

	// log.Println("[DEBUG]: should have responded to wingetcfg.deploy message")
}

func (w *Worker) WinGetCfgMarkPackageAsExcluded(msg *nats.Msg) {
	deploy := openuem_nats.DeployAction{}

	// log.Println("[DEBUG]: received a wingetcfg.deploy message")

	if err := json.Unmarshal(msg.Data, &deploy); err != nil {
		log.Println("[ERROR]: could not unmarshall WinGetCfg deployment action report from agent")
	}

	if err := w.Model.MarkPackageAsExcluded(deploy); err != nil {
		log.Printf("[ERROR]: could not mark package as excluded, reason: %v", err)
	}

	if err := msg.Respond(nil); err != nil {
		log.Printf("[ERROR]: could not respond to WinGetCfg deployment action report, reason: %v\n", err)
	}

	// log.Println("[DEBUG]: should have responded to wingetcfg.deploy message")
}

func (w *Worker) WinGetCfgApplicationReport(msg *nats.Msg) {
	report := openuem_nats.WingetCfgReport{}

	// log.Println("[DEBUG]: received a wingetcfg.report message")

	// Unmarshal data
	if err := json.Unmarshal(msg.Data, &report); err != nil {
		log.Println("[ERROR]: could not unmarshall WinGetCfg report from agent")
	}

	// log.Printf("[DEBUG]: wingetcfg.report data, %v", report)

	if err := w.Model.SaveProfileApplicationIssues(report.ProfileID, report.AgentID, report.Success, report.Error); err != nil {
		log.Printf("[ERROR]: could not save WinGetCfg profile issue, reason: %v", err)
	}

	if err := msg.Respond(nil); err != nil {
		log.Printf("[ERROR]: could not respond to WinGetCfg report, reason: %v\n", err)
	}

	// log.Println("[DEBUG]: should have responded to wingetcfg.report message")
}
