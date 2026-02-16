package bot

import (
    "strings"

    "github.com/go-gitea/giteabot/internal/provider"
)

func (b *Bot) RunLabelMaintenance() error {
    labelsToRemove := []string{
        b.Config.Labels.ReviewedWaitMerge,
        b.Config.Labels.ReviewedPrioritize,
        b.Config.Labels.PrLastCall,
    }
    for _, label := range labelsToRemove {
        merged, err := b.Provider.FetchMergedWithLabel(label)
        if err != nil {
            return err
        }
        for _, pr := range merged {
            _ = b.Provider.RemoveLabel(pr.Number, label)
        }
    }

    if err := b.removeBackportLabelsFromReleasePRs(); err != nil {
        return err
    }
    breaking, err := b.Provider.FetchBreakingWithoutLabel()
    if err != nil {
        return err
    }
    for _, pr := range breaking {
        _ = b.Provider.AddLabels(pr.Number, []string{b.Config.Labels.PrBreaking})
    }
    return nil
}

func (b *Bot) removeBackportLabelsFromReleasePRs() error {
    versions, err := b.Provider.FetchMilestones()
    if err != nil {
        return err
    }
    for _, version := range versions {
        prs, err := b.Provider.FetchTargeting("release/v" + version.MajorMinor)
        if err != nil {
            return err
        }
        for _, pr := range prs {
            for _, label := range pr.Labels {
                if isBackportLabel(label, b.Config.Labels.BackportPrefix) {
                    _ = b.Provider.RemoveLabel(pr.Number, label.Name)
                }
            }
        }
    }
    return nil
}

func isBackportLabel(label provider.Label, prefix string) bool {
    return strings.HasPrefix(label.Name, prefix)
}
