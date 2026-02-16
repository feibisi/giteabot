package bot

import (
    "fmt"

    "github.com/go-gitea/giteabot/internal/provider"
)

func (b *Bot) AssignMilestone(pr provider.PullRequest) error {
    versions, err := b.Provider.FetchMilestones()
    if err != nil {
        return err
    }
    var target provider.Version
    found := false
    for _, version := range versions {
        if pr.BaseRef == "release/v"+version.MajorMinor {
            target = version
            found = true
            break
        }
    }
    if !found {
        if len(versions) == 0 {
            return fmt.Errorf("no milestones available")
        }
        target = versions[0]
        for _, version := range versions[1:] {
            if provider.CompareMajorMinor(version, target) > 0 {
                target = version
            }
        }
    }
    return b.Provider.SetMilestone(pr.Number, target.Milestone.Number)
}

func (b *Bot) RemoveMilestonesFromClosedUnmerged() error {
    versions, err := b.Provider.FetchMilestones()
    if err != nil {
        return err
    }
    for _, version := range versions {
        prs, err := b.Provider.FetchUnmergedClosedPullRequestsWithMilestone(version.Milestone.Title)
        if err != nil {
            return err
        }
        for _, pr := range prs {
            _ = b.Provider.RemoveMilestone(pr.Number)
        }
    }
    return nil
}
