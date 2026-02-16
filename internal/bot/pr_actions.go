package bot

import (
    "fmt"
    "strings"

    "github.com/go-gitea/giteabot/internal/provider"
)

func (b *Bot) UpdateBranch(pr provider.PullRequest) error {
    needsUpdate, err := b.Provider.NeedsUpdate(pr.Number)
    if err != nil {
        return err
    }
    if !needsUpdate {
        return nil
    }
    if err := b.Provider.UpdatePullRequest(pr.Number); err != nil {
        if isMergeConflict(err) {
            _ = b.Provider.AddComment(pr.Number, fmt.Sprintf("@%s please fix the merge conflicts. :tea:", pr.User.Login))
            return fmt.Errorf("merge conflicts in PR")
        }
        return err
    }
    return nil
}

func isMergeConflict(err error) bool {
    if err == nil {
        return false
    }
    msg := err.Error()
    return strings.Contains(msg, "merge conflict between base and head") ||
        strings.Contains(msg, "merge conflicts")
}
