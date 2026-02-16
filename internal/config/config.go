package config

import (
    "errors"
    "fmt"
    "net/url"
    "os"
    "strconv"
    "strings"

    "github.com/go-gitea/giteabot/internal/provider"
)

type Features struct {
    Backport         bool
    LabelMaintenance bool
    MergeQueue       bool
    Milestones       bool
    LGTM             bool
    Comments         bool
    Locking          bool
    Feedback         bool
    LastCall         bool
    PrActions        bool
}

type Config struct {
    Provider      string
    APIBaseURL    string
    RepoOwner     string
    RepoName      string
    Fork          string
    Token         string
    WebhookSecret string
    WorkDir       string
    Mode          string
    EventPath     string
    EventName     string
    ListenAddr    string
    TriggerPath   string
    Features      Features
    Labels        provider.Labels
}

func Load() (Config, error) {
    cfg := Config{
        Provider:    strings.ToLower(strings.TrimSpace(os.Getenv("GITEABOT_PROVIDER"))),
        APIBaseURL:  strings.TrimSpace(os.Getenv("GITEABOT_API_BASE")),
        Fork:        strings.TrimSpace(os.Getenv("GITEABOT_FORK")),
        Token:       strings.TrimSpace(os.Getenv("GITEABOT_TOKEN")),
        WebhookSecret: strings.TrimSpace(os.Getenv("GITEABOT_WEBHOOK_SECRET")),
        WorkDir:     strings.TrimSpace(os.Getenv("GITEABOT_WORKDIR")),
        Mode:        strings.ToLower(strings.TrimSpace(os.Getenv("GITEABOT_MODE"))),
        EventPath:   strings.TrimSpace(os.Getenv("GITEABOT_EVENT_PATH")),
        EventName:   strings.TrimSpace(os.Getenv("GITEABOT_EVENT_NAME")),
        ListenAddr:  strings.TrimSpace(os.Getenv("GITEABOT_LISTEN")),
        TriggerPath: strings.TrimSpace(os.Getenv("GITEABOT_TRIGGER_PATH")),
    }

    if cfg.Mode == "" {
        cfg.Mode = "event"
    }
    if cfg.ListenAddr == "" {
        cfg.ListenAddr = ":8000"
    }
    if cfg.TriggerPath == "" {
        cfg.TriggerPath = "/trigger"
    }

    if cfg.Provider == "" {
        cfg.Provider = inferProvider()
    }
    if cfg.APIBaseURL == "" {
        cfg.APIBaseURL = defaultAPIBase(cfg.Provider)
    }

    repo := strings.TrimSpace(os.Getenv("GITEABOT_REPO"))
    if repo == "" {
        repo = strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY"))
    }
    if repo == "" {
        repo = strings.TrimSpace(os.Getenv("GITEA_REPOSITORY"))
    }
    if repo != "" {
        parts := strings.Split(repo, "/")
        if len(parts) != 2 {
            return cfg, fmt.Errorf("invalid GITEABOT_REPO: %s", repo)
        }
        cfg.RepoOwner = parts[0]
        cfg.RepoName = parts[1]
    }

    if cfg.EventPath == "" {
        if envPath := strings.TrimSpace(os.Getenv("GITHUB_EVENT_PATH")); envPath != "" {
            cfg.EventPath = envPath
        }
        if cfg.EventPath == "" {
            cfg.EventPath = strings.TrimSpace(os.Getenv("GITEA_EVENT_PATH"))
        }
    }
    if cfg.EventName == "" {
        if envName := strings.TrimSpace(os.Getenv("GITHUB_EVENT_NAME")); envName != "" {
            cfg.EventName = envName
        }
        if cfg.EventName == "" {
            cfg.EventName = strings.TrimSpace(os.Getenv("GITEA_EVENT_NAME"))
        }
    }

    if cfg.WorkDir == "" {
        cfg.WorkDir = ".giteabot"
    }

    cfg.Features = defaultFeatures()
    if err := applyFeatureOverrides(&cfg); err != nil {
        return cfg, err
    }

    cfg.Labels = provider.DefaultLabels()
    applyLabelOverrides(&cfg)

    if cfg.Token == "" {
        return cfg, errors.New("GITEABOT_TOKEN must be set")
    }
    if cfg.RepoOwner == "" || cfg.RepoName == "" {
        return cfg, errors.New("GITEABOT_REPO must be set (or rely on GITHUB_REPOSITORY/GITEA_REPOSITORY in Actions)")
    }

    if _, err := url.Parse(cfg.APIBaseURL); err != nil {
        return cfg, fmt.Errorf("invalid GITEABOT_API_BASE: %w", err)
    }

    return cfg, nil
}

func defaultAPIBase(provider string) string {
    if provider == "gitea" {
        server := strings.TrimSpace(os.Getenv("GITEA_SERVER_URL"))
        if server == "" {
            server = "https://gitea.com"
        }
        return strings.TrimRight(server, "/") + "/api/v1"
    }
    return "https://api.github.com"
}

func inferProvider() string {
    if os.Getenv("GITEA_SERVER_URL") != "" || os.Getenv("GITEA_ACTIONS") != "" {
        return "gitea"
    }
    return "github"
}

func defaultFeatures() Features {
    return Features{
        Backport:         true,
        LabelMaintenance: true,
        MergeQueue:       true,
        Milestones:       true,
        LGTM:             true,
        Comments:         true,
        Locking:          true,
        Feedback:         true,
        LastCall:         true,
        PrActions:        true,
    }
}

func applyFeatureOverrides(cfg *Config) error {
    var err error
    cfg.Features.Backport, err = readBool("GITEABOT_FEATURE_BACKPORT", cfg.Features.Backport)
    if err != nil {
        return err
    }
    cfg.Features.LabelMaintenance, err = readBool("GITEABOT_FEATURE_LABEL_MAINTENANCE", cfg.Features.LabelMaintenance)
    if err != nil {
        return err
    }
    cfg.Features.MergeQueue, err = readBool("GITEABOT_FEATURE_MERGE_QUEUE", cfg.Features.MergeQueue)
    if err != nil {
        return err
    }
    cfg.Features.Milestones, err = readBool("GITEABOT_FEATURE_MILESTONES", cfg.Features.Milestones)
    if err != nil {
        return err
    }
    cfg.Features.LGTM, err = readBool("GITEABOT_FEATURE_LGTM", cfg.Features.LGTM)
    if err != nil {
        return err
    }
    cfg.Features.Comments, err = readBool("GITEABOT_FEATURE_COMMENTS", cfg.Features.Comments)
    if err != nil {
        return err
    }
    cfg.Features.Locking, err = readBool("GITEABOT_FEATURE_LOCKING", cfg.Features.Locking)
    if err != nil {
        return err
    }
    cfg.Features.Feedback, err = readBool("GITEABOT_FEATURE_FEEDBACK", cfg.Features.Feedback)
    if err != nil {
        return err
    }
    cfg.Features.LastCall, err = readBool("GITEABOT_FEATURE_LAST_CALL", cfg.Features.LastCall)
    if err != nil {
        return err
    }
    cfg.Features.PrActions, err = readBool("GITEABOT_FEATURE_PR_ACTIONS", cfg.Features.PrActions)
    if err != nil {
        return err
    }
    return nil
}

func applyLabelOverrides(cfg *Config) {
    labels := &cfg.Labels
    labels.BackportPrefix = readString("GITEABOT_LABEL_BACKPORT_PREFIX", labels.BackportPrefix)
    labels.BackportDone = readString("GITEABOT_LABEL_BACKPORT_DONE", labels.BackportDone)
    labels.BackportManual = readString("GITEABOT_LABEL_BACKPORT_MANUAL", labels.BackportManual)
    labels.ReviewedPrefix = readString("GITEABOT_LABEL_REVIEWED_PREFIX", labels.ReviewedPrefix)
    labels.ReviewedWaitMerge = readString("GITEABOT_LABEL_REVIEWED_WAIT_MERGE", labels.ReviewedWaitMerge)
    labels.ReviewedPrioritize = readString("GITEABOT_LABEL_REVIEWED_PRIORITIZE_MERGE", labels.ReviewedPrioritize)
    labels.PrLastCall = readString("GITEABOT_LABEL_PR_LAST_CALL", labels.PrLastCall)
    labels.PrBreaking = readString("GITEABOT_LABEL_PR_BREAKING", labels.PrBreaking)
    labels.LGTMPrefix = readString("GITEABOT_LABEL_LGTM_PREFIX", labels.LGTMPrefix)
    labels.LGTMNeed2 = readString("GITEABOT_LABEL_LGTM_NEED_2", labels.LGTMNeed2)
    labels.LGTMNeed1 = readString("GITEABOT_LABEL_LGTM_NEED_1", labels.LGTMNeed1)
    labels.LGTMDone = readString("GITEABOT_LABEL_LGTM_DONE", labels.LGTMDone)
    labels.LGTMBlocked = readString("GITEABOT_LABEL_LGTM_BLOCKED", labels.LGTMBlocked)
    labels.BotPrefix = readString("GITEABOT_LABEL_BOT_PREFIX", labels.BotPrefix)
    labels.UpdateBranch = readString("GITEABOT_LABEL_UPDATE_BRANCH", labels.UpdateBranch)
    labels.TocReminded = readString("GITEABOT_LABEL_TOC_REMINDED", labels.TocReminded)
    labels.NeedsFeedback = readString("GITEABOT_LABEL_NEEDS_FEEDBACK", labels.NeedsFeedback)
    labels.SizePrefix = readString("GITEABOT_LABEL_SIZE_PREFIX", labels.SizePrefix)
    labels.PrPrefix = readString("GITEABOT_LABEL_PR_PREFIX", labels.PrPrefix)
}

func readBool(name string, fallback bool) (bool, error) {
    value := strings.TrimSpace(os.Getenv(name))
    if value == "" {
        return fallback, nil
    }
    parsed, err := strconv.ParseBool(value)
    if err != nil {
        return false, fmt.Errorf("invalid boolean for %s", name)
    }
    return parsed, nil
}

func readString(name string, fallback string) string {
    value := strings.TrimSpace(os.Getenv(name))
    if value == "" {
        return fallback
    }
    return value
}
