package bot

import (
    "testing"

    "github.com/go-gitea/giteabot/internal/config"
    "github.com/go-gitea/giteabot/internal/provider"
)

func TestIsRelevantLabel(t *testing.T) {
    cfg := config.Config{Labels: provider.DefaultLabels()}
    bot := &Bot{Config: cfg}

    if !bot.IsRelevantLabel("reviewed/wait-merge") {
        t.Fatalf("expected reviewed label to be relevant")
    }
    if !bot.IsRelevantLabel("backport/v1.2") {
        t.Fatalf("expected backport label to be relevant")
    }
    if !bot.IsRelevantLabel("giteabot/update-branch") {
        t.Fatalf("expected bot label to be relevant")
    }
    if bot.IsRelevantLabel("kind/bug") {
        t.Fatalf("expected unrelated label to be irrelevant")
    }
}
