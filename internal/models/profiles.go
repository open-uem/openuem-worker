package models

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

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
		log.Println(p.AgentID, p.ProfileID)

		profileIssue, err = m.Client.ProfileIssue.Create().SetError(p.Error).SetAgentsID(p.AgentID).SetProfileID(p.ProfileID).Save(context.Background())
		if err != nil {
			return err
		}
		profileIssueID = profileIssue.ID
	}

	// Now, we must store the report for the profile's tasks
	if p.Tasks != nil {
		for _, report := range p.Tasks {

			taskID, err := strconv.Atoi(strings.TrimRight(strings.TrimPrefix(report.Name, "task_"), "_"))
			if err != nil {
				return errors.New("could not convert the task ID from string to int")
			}

			taskExist, err := m.Client.Task.Query().Where(task.ID(taskID)).Exist(context.Background())
			if err != nil {
				log.Printf("[ERROR]: could not check if profile task exists, reason: %v", err)
				return err
			}

			if !taskExist {
				log.Printf("[ERROR]: the task %d for profile %d, doesn't exist", taskID, profileIssueID)
				continue
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
