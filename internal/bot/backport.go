package bot

import (
    "fmt"

    "github.com/go-gitea/giteabot/internal/gitops"
    "github.com/go-gitea/giteabot/internal/provider"
)

func (b *Bot) RunBackport() error {
    versions, err := b.Provider.FetchMilestones()
    if err != nil {
        return err
    }
    repo, err := b.EnsureRepo()
    if err != nil {
        return err
    }
    for _, version := range versions {
        candidates, err := b.Provider.FetchCandidates(version.MajorMinor)
        if err != nil {
            return err
        }
        for _, candidate := range candidates {
            if err := b.handleBackportCandidate(*repo, candidate, version); err != nil {
                return err
            }
        }
    }
    return nil
}

func (b *Bot) handleBackportCandidate(repo gitops.Repo, candidate provider.Issue, version provider.Version) error {
    exists, err := b.Provider.BackportPullRequestExists(candidate.Number, version.MajorMinor)
    if err != nil {
        return err
    }
    if exists {
        return nil
    }
    original, err := b.Provider.FetchPullRequest(candidate.Number)
    if err != nil {
        return err
    }
    success, err := gitops.CherryPick(repo, original.MergeCommitSHA, original.Number, version.MajorMinor)
    if err != nil {
        return err
    }
    if !success {
        message := fmt.Sprintf(
            "I was unable to create a backport for %s. @%s, please send one manually. :tea:\n\n```\ngo run ./contrib/backport %d\n...  // fix git conflicts if any\ngo run ./contrib/backport --continue\n```\n",
            version.MajorMinor,
            original.User.Login,
            original.Number,
        )
        _ = b.Provider.AddComment(original.Number, message)
        _ = b.Provider.AddLabels(original.Number, []string{b.Config.Labels.BackportManual})
        return nil
    }
    return b.Provider.CreateBackportPullRequest(original, version, fmt.Sprintf("backport-%d-v%s", original.Number, version.MajorMinor))
}
