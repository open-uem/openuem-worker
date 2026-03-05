package models

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/open-uem/ent"
	"github.com/open-uem/ent/agent"
	"github.com/open-uem/ent/profile"
	"github.com/open-uem/ent/profileissue"
	"github.com/open-uem/ent/site"
	"github.com/open-uem/ent/tag"
	"github.com/open-uem/ent/task"
	"github.com/open-uem/ent/taskreport"
	"github.com/open-uem/nats"
)

func (m *Model) GetProfilesAppliedToAll(siteID int) ([]*ent.Profile, error) {
	return m.Client.Profile.Query().WithTasks().Where(profile.DisabledEQ(false), profile.ApplyToAll(true), profile.HasSiteWith(site.ID(siteID))).All(context.Background())
}

func (m *Model) GetProfilesAppliedToAllFilteredByProfile(siteID int, profileID int) ([]*ent.Profile, error) {
	return m.Client.Profile.Query().WithTasks().Where(profile.ID(profileID), profile.DisabledEQ(false), profile.ApplyToAll(true), profile.HasSiteWith(site.ID(siteID))).All(context.Background())
}

func (m *Model) GetProfilesAppliedToAgent(siteID int, agentID string) ([]*ent.Profile, error) {
	agent, err := m.Client.Agent.Query().WithTags().Where(agent.ID(agentID), agent.HasSiteWith(site.ID(siteID))).Only(context.Background())
	if err != nil {
		return nil, err
	}

	if agent.Edges.Tags != nil {
		tags := []int{}

		for _, tag := range agent.Edges.Tags {
			tags = append(tags, tag.ID)
		}

		return m.Client.Profile.Query().WithTasks().Where(profile.DisabledEQ(false), profile.HasTagsWith(tag.IDIn(tags...)), profile.HasSiteWith(site.ID(siteID))).All(context.Background())
	}

	return []*ent.Profile{}, nil
}

func (m *Model) GetProfilesAppliedToAgentFilteredByProfile(siteID int, agentID string, profileID int) ([]*ent.Profile, error) {
	agent, err := m.Client.Agent.Query().WithTags().Where(agent.ID(agentID), agent.HasSiteWith(site.ID(siteID))).Only(context.Background())
	if err != nil {
		return nil, err
	}

	if agent.Edges.Tags != nil {
		tags := []int{}

		for _, tag := range agent.Edges.Tags {
			tags = append(tags, tag.ID)
		}

		return m.Client.Profile.Query().WithTasks().Where(profile.ID(profileID), profile.DisabledEQ(false), profile.HasTagsWith(tag.IDIn(tags...)), profile.HasSiteWith(site.ID(siteID))).All(context.Background())
	}

	return []*ent.Profile{}, nil
}

func (m *Model) SaveProfileApplicationIssues(p nats.ProfileReport) error {
	var err error

	exists := true
	profileIssueID := -1
	// Create issue or update the issue
	profileIssue, err := m.Client.ProfileIssue.Query().Where(profileissue.HasProfileWith(profile.ID(p.ProfileID)), profileissue.HasAgentsWith(agent.ID(p.AgentID))).Only(context.Background())
	if err != nil {
		if !ent.IsNotFound(err) {
			return err
		}
		exists = false
	} else {
		profileIssueID = profileIssue.ID
	}

	if exists {
		if err := m.Client.ProfileIssue.Update().Where(profileissue.ID(profileIssueID)).SetError(p.Error).Exec(context.Background()); err != nil {
			return err
		}
	} else {
		profileIssue, err = m.Client.ProfileIssue.Create().SetError(p.Error).SetAgentsID(p.AgentID).SetProfileID(p.ProfileID).Save(context.Background())
		if err != nil {
			return err
		}
		profileIssueID = profileIssue.ID
	}

	// Now, we must store the report for the profile's tasks
	if p.Tasks != nil {
		for _, report := range p.Tasks {

			taskID, err := strconv.Atoi(strings.Split(strings.TrimPrefix(report.Name, "task_"), "_")[0])
			if err != nil {
				return errors.New("could not convert the task ID from string to int")
			}

			taskExist := true

			theTask, err := m.Client.Task.Query().Where(task.ID(taskID)).First(context.Background())
			if err != nil {
				if ent.IsNotFound(err) {
					taskExist = false
				} else {
					log.Printf("[ERROR]: could not check if profile task exists, reason: %v", err)
					return err
				}
			}

			if !taskExist {
				log.Printf("[ERROR]: the task %d for profile %d, doesn't exist", taskID, profileIssueID)
				continue
			}

			// if we have a task to install, update or delete a package we must try to update the deployment info
			if theTask.Type == task.TypeFlatpakInstall || theTask.Type == task.TypeBrewCaskInstall || theTask.Type == task.TypeBrewFormulaInstall {
				m.SetFlatpakOrBrewDeploymentInfo(taskID, p, report, theTask, "install")
			}

			if theTask.Type == task.TypeBrewCaskUpgrade {
				m.SetFlatpakOrBrewDeploymentInfo(taskID, p, report, theTask, "update")
			}

			if theTask.Type == task.TypeFlatpakUninstall || theTask.Type == task.TypeBrewCaskUninstall || theTask.Type == task.TypeBrewFormulaUninstall {
				m.SetFlatpakOrBrewDeploymentInfo(taskID, p, report, theTask, "uninstall")
			}

			exists, err := m.Client.TaskReport.Query().
				Where(
					taskreport.HasProfileissueWith(profileissue.ID(profileIssueID)),
					taskreport.HasTaskWith(task.ID(taskID)),
				).
				Exist(context.Background())

			if err != nil {
				log.Printf("[ERROR]: could not check if profile issue exists, reason: %v", err)
				return err
			}

			if exists {
				err := m.Client.TaskReport.Update().
					Where(
						taskreport.HasProfileissueWith(profileissue.ID(profileIssueID)),
						taskreport.HasTaskWith(task.ID(taskID)),
					).
					SetStdError(report.StdErr).
					SetStdOutput(report.StdOut).
					SetEnd(report.EndTime).
					SetFailed(report.Failed).
					Exec(context.Background())
				if err != nil {
					log.Printf("[ERROR]: could not save task %d report for profile %d, reason: %v", taskID, profileIssueID, err)
				}
			} else {
				err := m.Client.TaskReport.Create().
					SetProfileissueID(profileIssueID).
					SetTaskID(taskID).
					SetStdError(report.StdErr).
					SetStdOutput(report.StdOut).
					SetEnd(report.EndTime).
					SetFailed(report.Failed).
					Exec(context.Background())
				if err != nil {
					log.Printf("[ERROR]: could not save task %d report for profile %d, reason: %v", taskID, profileIssueID, err)
				}
			}
		}
	}

	return nil
}

func (m *Model) SetFlatpakOrBrewDeploymentInfo(taskID int, p nats.ProfileReport, report nats.TaskReport, t *ent.Task, action string) {
	deployAction := nats.DeployAction{
		Failed:         report.Failed,
		PackageId:      t.PackageID,
		PackageName:    t.PackageName,
		AgentId:        p.AgentID,
		Action:         action,
		PackageVersion: "",
	}

	if t.Type == task.TypeBrewCaskInstall || t.Type == task.TypeBrewCaskUpgrade || t.Type == task.TypeBrewCaskUninstall {
		deployAction.PackageId = "cask-" + t.PackageID
	}

	when, err := time.Parse(time.RFC3339Nano, report.EndTime)
	if err == nil {
		deployAction.When = when
	}

	if report.StdErr != "" {
		deployAction.Info = report.StdErr
	}

	if err := m.SaveFlatpakOrBrewDeployInfo(deployAction); err != nil {
		log.Printf("[ERROR]: could not save deployment action for flatpak install task %d, reason: %v", taskID, err)
	}
}
