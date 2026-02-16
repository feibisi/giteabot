package bot

import "time"

func (b *Bot) RunFeedback() error {
    issues, err := b.Provider.FetchOpenIssuesWithLabel(b.Config.Labels.NeedsFeedback)
    if err != nil {
        return err
    }
    cutoff := time.Now().Add(-30 * 24 * time.Hour)
    for _, issue := range issues {
        if issue.UpdatedAt.Before(cutoff) {
            _ = b.Provider.AddComment(issue.Number, "We close issues that need feedback from the author if there were no new comments for a month. :tea:")
            _ = b.Provider.CloseIssue(issue.Number)
        }
    }
    return nil
}
