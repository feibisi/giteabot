package bot

import "time"

func (b *Bot) RunLocking() error {
    twoWeeksAgo := time.Now().Add(-14 * 24 * time.Hour)
    threeMonthsAgo := time.Now().Add(-90 * 24 * time.Hour)
    issues, err := b.Provider.FetchClosedOldIssuesAndPullRequests(threeMonthsAgo)
    if err != nil {
        return err
    }
    for _, issue := range issues {
        locked, err := b.Provider.LockIssue(issue.Number, "resolved")
        if err != nil || !locked {
            continue
        }
        lastComment, err := b.Provider.FetchLastComment(issue.Number)
        if err != nil || lastComment == nil {
            continue
        }
        if lastComment.CreatedAt.After(twoWeeksAgo) {
            _ = b.Provider.AddComment(issue.Number, "We lock pull requests and issues 3 months after they were closed. If there's any need for further discussion, please open a new issue. :tea:")
        }
    }
    return nil
}
