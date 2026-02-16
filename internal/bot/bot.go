package bot

import (
    "fmt"
    "strings"

    "github.com/go-gitea/giteabot/internal/config"
    "github.com/go-gitea/giteabot/internal/gitops"
    "github.com/go-gitea/giteabot/internal/provider"
)

type Bot struct {
    Config   config.Config
    Provider provider.Provider
    Repo     *gitops.Repo
}

func New(cfg config.Config, prov provider.Provider) *Bot {
    return &Bot{Config: cfg, Provider: prov}
}

func (b *Bot) EnsureRepo() (*gitops.Repo, error) {
    if b.Repo != nil {
        return b.Repo, nil
    }
    user, err := b.Provider.FetchCurrentUser()
    if err != nil {
        return nil, err
    }
    repo, err := gitops.Initialize(b.Config, user.Login, user.Email)
    if err != nil {
        return nil, err
    }
    b.Repo = &repo
    return b.Repo, nil
}

func (b *Bot) IsRelevantLabel(label string) bool {
    return strings.HasPrefix(label, b.Config.Labels.ReviewedPrefix) ||
        strings.HasPrefix(label, b.Config.Labels.BackportPrefix) ||
        strings.HasPrefix(label, b.Config.Labels.BotPrefix)
}

func (b *Bot) HandleLabelAction(label string, pr provider.PullRequest) error {
    if !b.Config.Features.PrActions {
        return nil
    }
    switch label {
    case b.Config.Labels.UpdateBranch:
        err := b.UpdateBranch(pr)
        if err != nil {
            _ = b.Provider.AddComment(pr.Number, fmt.Sprintf("I failed to update the branch because of the following error: %s Sorry about that. :tea:", err.Error()))
            return err
        }
        _ = b.Provider.RemoveLabel(pr.Number, b.Config.Labels.UpdateBranch)
    }
    return nil
}
