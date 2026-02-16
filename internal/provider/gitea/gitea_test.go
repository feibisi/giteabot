package gitea

import (
    "testing"

    "github.com/go-gitea/giteabot/internal/provider"
)

func TestFilterBackportLabels(t *testing.T) {
    labels := provider.DefaultLabels()
    input := []provider.Label{
        {Name: "lgtm/done"},
        {Name: "backport/v1.2"},
        {Name: "reviewed/wait-merge"},
        {Name: "size/S"},
        {Name: "pr/breaking"},
        {Name: "kind/bug"},
    }
    filtered := filterBackportLabels(input, labels)
    if len(filtered) != 1 || filtered[0] != "kind/bug" {
        t.Fatalf("unexpected filtered labels: %v", filtered)
    }
}

func TestCountBackportLabels(t *testing.T) {
    labels := provider.DefaultLabels()
    input := []provider.Label{{Name: "backport/v1.1"}, {Name: "backport/v1.2"}, {Name: "kind/bug"}}
    if countBackportLabels(input, labels) != 2 {
        t.Fatalf("expected 2 backport labels")
    }
}
