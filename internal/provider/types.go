package provider

import "time"

type Label struct {
    Name string `json:"name"`
}

type Labels struct {
    BackportPrefix        string
    BackportDone          string
    BackportManual        string
    ReviewedPrefix        string
    ReviewedWaitMerge     string
    ReviewedPrioritize    string
    PrLastCall            string
    PrBreaking            string
    LGTMPrefix            string
    LGTMNeed2             string
    LGTMNeed1             string
    LGTMDone              string
    LGTMBlocked           string
    BotPrefix             string
    UpdateBranch          string
    TocReminded           string
    NeedsFeedback         string
    SizePrefix            string
    PrPrefix              string
}

type User struct {
    Login string `json:"login"`
    Email string `json:"email"`
}

type Milestone struct {
    Title  string `json:"title"`
    Number int    `json:"number"`
}

type PullRequest struct {
    Number               int
    Title                string
    Body                 string
    State                string
    BaseRef              string
    BaseSHA              string
    HeadSHA              string
    Labels               []Label
    User                 User
    RequestedReviewers   []User
    MaintainerCanModify  bool
    Milestone            *Milestone
    UpdatedAt            time.Time
    Merged               bool
    MergeCommitSHA       string
}

type Issue struct {
    Number    int
    Title     string
    Labels    []Label
    UpdatedAt time.Time
    IsPull    bool
    ClosedAt  *time.Time
}

type Reviewers struct {
    Approvers map[string]struct{}
    Blockers  map[string]struct{}
}

type Comment struct {
    CreatedAt time.Time `json:"created_at"`
}

type Version struct {
    MajorMinor string
    Milestone  Milestone
    Patch      int
    Major      int
    Minor      int
}

func DefaultLabels() Labels {
    return Labels{
        BackportPrefix:    "backport/",
        BackportDone:      "backport/done",
        BackportManual:    "backport/manual",
        ReviewedPrefix:    "reviewed/",
        ReviewedWaitMerge: "reviewed/wait-merge",
        ReviewedPrioritize: "reviewed/prioritize-merge",
        PrLastCall:        "pr/last-call",
        PrBreaking:        "pr/breaking",
        LGTMPrefix:        "lgtm/",
        LGTMNeed2:         "lgtm/need 2",
        LGTMNeed1:         "lgtm/need 1",
        LGTMDone:          "lgtm/done",
        LGTMBlocked:       "lgtm/blocked",
        BotPrefix:         "giteabot/",
        UpdateBranch:      "giteabot/update-branch",
        TocReminded:       "giteabot/toc-reminded",
        NeedsFeedback:     "issue/needs-feedback",
        SizePrefix:        "size/",
        PrPrefix:          "pr/",
    }
}
