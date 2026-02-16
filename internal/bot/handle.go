package bot

import "strings"

func (b *Bot) HandleEvent(eventName string, event Event) error {
    switch eventName {
    case "push":
        return b.handlePush(event)
    case "pull_request":
        return b.handlePullRequest(event)
    case "pull_request_review":
        return b.handlePullRequestReview(event)
    }
    return nil
}

func (b *Bot) handlePush(event Event) error {
    defaultBranch := "main"
    if event.Repository != nil && event.Repository.DefaultBranch != "" {
        defaultBranch = event.Repository.DefaultBranch
    }
    if event.Ref == "refs/heads/"+defaultBranch && b.Config.Features.Backport {
        if err := b.RunBackport(); err != nil {
            return err
        }
    }
    if b.Config.Features.LabelMaintenance {
        if err := b.RunLabelMaintenance(); err != nil {
            return err
        }
    }
    if b.Config.Features.MergeQueue {
        if err := b.RunMergeQueue(); err != nil {
            return err
        }
    }
    if b.Config.Features.Locking {
        if err := b.RunLocking(); err != nil {
            return err
        }
    }
    if b.Config.Features.Feedback {
        if err := b.RunFeedback(); err != nil {
            return err
        }
    }
    if b.Config.Features.LastCall {
        return b.RunLastCall()
    }
    return nil
}

func (b *Bot) handlePullRequest(event Event) error {
    if event.PullRequest == nil {
        return nil
    }
    pr := event.PullRequest.ToProvider()
    action := event.Action
    switch action {
    case "labeled", "unlabeled":
        if event.Label != nil && b.IsRelevantLabel(event.Label.Name) {
            if b.Config.Features.LabelMaintenance {
                if err := b.RunLabelMaintenance(); err != nil {
                    return err
                }
            }
            if b.Config.Features.MergeQueue {
                if err := b.RunMergeQueue(); err != nil {
                    return err
                }
            }
            return b.HandleLabelAction(event.Label.Name, pr)
        }
    case "opened":
        if b.Config.Features.LabelMaintenance {
            if err := b.RunLabelMaintenance(); err != nil {
                return err
            }
        }
        if pr.BaseRef == "main" && b.Config.Features.Comments {
            if err := b.CommentIfTranslationsChanged(pr); err != nil {
                return err
            }
        }
        if strings.HasPrefix(pr.BaseRef, "release/") && b.Config.Features.Milestones {
            if err := b.AssignMilestone(pr); err != nil {
                return err
            }
        }
        if b.Config.Features.LGTM {
            return b.UpdateLgtm(pr)
        }
        return nil
    case "synchronize", "review_requested", "review_request_removed":
        if b.Config.Features.LGTM {
            return b.UpdateLgtm(pr)
        }
        return nil
    case "closed":
        if b.Config.Features.Milestones {
            if pr.Merged && pr.Milestone == nil {
                if err := b.AssignMilestone(pr); err != nil {
                    return err
                }
            }
            return b.RemoveMilestonesFromClosedUnmerged()
        }
        return nil
    }
    return nil
}

func (b *Bot) handlePullRequestReview(event Event) error {
    if event.PullRequest == nil {
        return nil
    }
    pr := event.PullRequest.ToProvider()
    if b.Config.Features.LGTM {
        return b.UpdateLgtm(pr)
    }
    return nil
}
