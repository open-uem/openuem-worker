package models

import (
	"context"

	"github.com/open-uem/ent"
	"github.com/open-uem/ent/agent"
	"github.com/open-uem/ent/profile"
	"github.com/open-uem/ent/profileissue"
	"github.com/open-uem/ent/tag"
)

func (m *Model) GetProfilesAppliedToAll() ([]*ent.Profile, error) {
	return m.Client.Profile.Query().WithTasks().Where(profile.ApplyToAll(true)).All(context.Background())
}

func (m *Model) GetProfilesAppliedToAgent(agentID string) ([]*ent.Profile, error) {
	agent, err := m.Client.Agent.Query().WithTags().Where(agent.ID(agentID)).Only(context.Background())
	if err != nil {
		return nil, err
	}

	if agent.Edges.Tags != nil {
		tags := []int{}

		for _, tag := range agent.Edges.Tags {
			tags = append(tags, tag.ID)
		}

		return m.Client.Profile.Query().WithTasks().Where(profile.HasTagsWith(tag.IDIn(tags...))).All(context.Background())
	}

	return []*ent.Profile{}, nil
}

func (m *Model) SaveProfileApplicationIssues(profileID int, agentID string, success bool, errorData string) error {
	var issue *ent.ProfileIssue
	var err error

	if success {
		issue, err = m.Client.ProfileIssue.Query().Where(profileissue.HasProfileWith(profile.ID(profileID)), profileissue.HasAgentsWith(agent.ID(agentID))).Only(context.Background())
		if err != nil {
			if ent.IsNotFound(err) {
				return nil
			}
			return err
		}

		if err := m.Client.ProfileIssue.DeleteOneID(issue.ID).Exec(context.Background()); err != nil {
			return err
		}

		return m.Client.Profile.UpdateOneID(profileID).RemoveIssueIDs(issue.ID).Exec(context.Background())
	} else {
		// Create issue or update the issue
		issue, err = m.Client.ProfileIssue.Query().Where(profileissue.HasProfileWith(profile.ID(profileID)), profileissue.HasAgentsWith(agent.ID(agentID))).Only(context.Background())
		if err != nil && !ent.IsNotFound(err) {
			return err
		}

		if ent.IsNotFound(err) {
			issue, err = m.Client.ProfileIssue.Create().SetError(errorData).SetAgentsID(agentID).SetProfileID(profileID).Save(context.Background())
			if err != nil {
				return err
			}
		}

		return m.Client.ProfileIssue.UpdateOneID(issue.ID).SetError(errorData).Exec(context.Background())
	}
}
