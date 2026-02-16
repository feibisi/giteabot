package bot

import (
    "time"

    "github.com/go-gitea/giteabot/internal/provider"
)

func (b *Bot) RunLastCall() error {
    prs, err := b.Provider.FetchOpenPullRequestsWithLabel(b.Config.Labels.PrLastCall)
    if err != nil {
        return err
    }
    oneWeekAgo := time.Now().Add(-7 * 24 * time.Hour)
    twoWeeksAgo := time.Now().Add(-14 * 24 * time.Hour)
    for _, issue := range prs {
        updatedAt := issue.UpdatedAt
        hasNeedOne := hasLabel(issue.Labels, b.Config.Labels.LGTMNeed1)
        tocReminded := hasLabel(issue.Labels, b.Config.Labels.TocReminded)

        if updatedAt.Before(oneWeekAgo) && hasNeedOne && !tocReminded {
            _ = b.Provider.AddComment(issue.Number, "This pull request has a last call and has not had any activity in the past week. @go-gitea/technical-oversight-committee please review this pull request and either merge it or request a review from a maintainer. :tea:")
            _ = b.Provider.AddLabels(issue.Number, []string{b.Config.Labels.TocReminded})
            continue
        }

        if updatedAt.Before(twoWeeksAgo) {
            _ = b.Provider.AddComment(issue.Number, "This pull request has a last call and has not had any activity in the past two weeks. Consider it to be a polite refusal: https://github.com/go-gitea/gitea/blob/main/CONTRIBUTING.md#final-call . :tea:")
            _ = b.Provider.ClosePullRequest(issue.Number)
        }
    }
    return nil
}

func hasLabel(labels []provider.Label, name string) bool {
    for _, label := range labels {
        if label.Name == name {
            return true
        }
    }
    return false
}
