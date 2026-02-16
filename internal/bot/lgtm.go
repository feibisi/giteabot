package bot

import (
    "fmt"
    "sort"
    "strings"

    "github.com/go-gitea/giteabot/internal/provider"
)

func (b *Bot) UpdateLgtm(pr provider.PullRequest) error {
    reviewers, err := b.Provider.GetPullRequestReviewers(pr)
    if err != nil {
        return err
    }
    state, message, desiredLabel := lgtmStatus(reviewers, b.Config.Labels)
    currentLabels := filterLabels(pr.Labels, b.Config.Labels.LGTMPrefix)
    for _, label := range currentLabels {
        if label != desiredLabel {
            _ = b.Provider.RemoveLabel(pr.Number, label)
        }
    }
    if !contains(currentLabels, desiredLabel) {
        _ = b.Provider.AddLabels(pr.Number, []string{desiredLabel})
    }
    return b.Provider.SetCommitStatus(pr.HeadSHA, state, message)
}

func lgtmStatus(reviewers provider.Reviewers, labels provider.Labels) (string, string, string) {
    if len(reviewers.Blockers) > 0 {
        blocked := make([]string, 0, len(reviewers.Blockers))
        for name := range reviewers.Blockers {
            blocked = append(blocked, name)
        }
        sort.Strings(blocked)
        return "failure", "Blocked by " + strings.Join(blocked, ", "), labels.LGTMBlocked
    }
    switch len(reviewers.Approvers) {
    case 0:
        return "pending", "Needs two more approvals", labels.LGTMNeed2
    case 1:
        return "pending", "Needs one more approval", labels.LGTMNeed1
    default:
        return "success", fmt.Sprintf("Approved by %d people", len(reviewers.Approvers)), labels.LGTMDone
    }
}

func filterLabels(labels []provider.Label, prefix string) []string {
    filtered := []string{}
    for _, label := range labels {
        if len(label.Name) >= len(prefix) && label.Name[:len(prefix)] == prefix {
            filtered = append(filtered, label.Name)
        }
    }
    return filtered
}

func contains(values []string, target string) bool {
    for _, value := range values {
        if value == target {
            return true
        }
    }
    return false
}
