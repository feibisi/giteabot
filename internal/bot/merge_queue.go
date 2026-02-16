package bot

func (b *Bot) RunMergeQueue() error {
    pending, err := b.Provider.FetchPendingMerge()
    if err != nil {
        return err
    }
    for _, pr := range pending {
        full, err := b.Provider.FetchPullRequest(pr.Number)
        if err != nil {
            return err
        }
        if err := b.UpdateBranch(full); err != nil {
            if err.Error() == "merge conflicts in PR" {
                _ = b.Provider.RemoveLabel(pr.Number, b.Config.Labels.ReviewedWaitMerge)
                continue
            }
            return err
        }
    }
    return nil
}
