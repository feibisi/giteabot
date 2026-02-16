package config

import "testing"

func TestLoadDefaults(t *testing.T) {
    t.Setenv("GITEABOT_TOKEN", "token")
    t.Setenv("GITEABOT_REPO", "owner/repo")

    cfg, err := Load()
    if err != nil {
        t.Fatalf("expected config load to succeed: %v", err)
    }

    if !cfg.Features.Backport || !cfg.Features.LabelMaintenance || !cfg.Features.MergeQueue {
        t.Fatalf("expected default feature flags to be enabled")
    }

    if cfg.Labels.BackportPrefix != "backport/" {
        t.Fatalf("expected default backport prefix, got %q", cfg.Labels.BackportPrefix)
    }
    if cfg.Labels.ReviewedWaitMerge != "reviewed/wait-merge" {
        t.Fatalf("expected default reviewed label, got %q", cfg.Labels.ReviewedWaitMerge)
    }
}

func TestLoadOverrides(t *testing.T) {
    t.Setenv("GITEABOT_TOKEN", "token")
    t.Setenv("GITEABOT_REPO", "owner/repo")
    t.Setenv("GITEABOT_FEATURE_BACKPORT", "false")
    t.Setenv("GITEABOT_LABEL_BACKPORT_PREFIX", "bp/")
    t.Setenv("GITEABOT_LABEL_REVIEWED_WAIT_MERGE", "queue/wait")

    cfg, err := Load()
    if err != nil {
        t.Fatalf("expected config load to succeed: %v", err)
    }

    if cfg.Features.Backport {
        t.Fatalf("expected backport feature to be disabled")
    }
    if cfg.Labels.BackportPrefix != "bp/" {
        t.Fatalf("expected backport prefix override, got %q", cfg.Labels.BackportPrefix)
    }
    if cfg.Labels.ReviewedWaitMerge != "queue/wait" {
        t.Fatalf("expected reviewed label override, got %q", cfg.Labels.ReviewedWaitMerge)
    }
}
