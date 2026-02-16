package bot

import (
    "testing"

    "github.com/go-gitea/giteabot/internal/provider"
)

func TestLgtmStatusBlocked(t *testing.T) {
    labels := provider.DefaultLabels()
    reviewers := provider.Reviewers{
        Approvers: map[string]struct{}{},
        Blockers:  map[string]struct{}{"user": {}},
    }
    state, message, label := lgtmStatus(reviewers, labels)
    if state != "failure" {
        t.Fatalf("expected failure state, got %s", state)
    }
    if label != labels.LGTMBlocked {
        t.Fatalf("expected %s label, got %s", labels.LGTMBlocked, label)
    }
    if message == "" {
        t.Fatalf("expected message")
    }
}

func TestLgtmStatusApprovals(t *testing.T) {
    labels := provider.DefaultLabels()
    reviewers := provider.Reviewers{
        Approvers: map[string]struct{}{"user": {}},
        Blockers:  map[string]struct{}{},
    }
    state, _, label := lgtmStatus(reviewers, labels)
    if state != "pending" || label != labels.LGTMNeed1 {
        t.Fatalf("expected pending/%s, got %s/%s", labels.LGTMNeed1, state, label)
    }

    reviewers.Approvers["user2"] = struct{}{}
    state, _, label = lgtmStatus(reviewers, labels)
    if state != "success" || label != labels.LGTMDone {
        t.Fatalf("expected success/%s, got %s/%s", labels.LGTMDone, state, label)
    }
}
