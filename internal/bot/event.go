package bot

import (
    "encoding/json"
    "time"

    "github.com/go-gitea/giteabot/internal/provider"
)

type Event struct {
    Action      string            `json:"action"`
    Label       *eventLabel       `json:"label"`
    PullRequest *pullRequestEvent `json:"pull_request"`
    Repository  *eventRepo        `json:"repository"`
    Ref         string            `json:"ref"`
}

type eventLabel struct {
    Name string `json:"name"`
}

type eventRepo struct {
    DefaultBranch string `json:"default_branch"`
}

type pullRequestEvent struct {
    Number              int                `json:"number"`
    Title               string             `json:"title"`
    Body                string             `json:"body"`
    State               string             `json:"state"`
    Base                struct {
        Ref string `json:"ref"`
        SHA string `json:"sha"`
    } `json:"base"`
    Head struct {
        SHA string `json:"sha"`
    } `json:"head"`
    Labels              []provider.Label   `json:"labels"`
    User                provider.User      `json:"user"`
    RequestedReviewers  []provider.User    `json:"requested_reviewers"`
    MaintainerCanModify bool               `json:"maintainer_can_modify"`
    AllowMaintainerEdit bool               `json:"allow_maintainer_edit"`
    Milestone           *provider.Milestone `json:"milestone"`
    UpdatedAt           time.Time          `json:"updated_at"`
    Merged              bool               `json:"merged"`
    MergeCommitSHA      string             `json:"merge_commit_sha"`
}

func ParseEvent(data []byte) (Event, error) {
    var event Event
    err := json.Unmarshal(data, &event)
    return event, err
}

func (pr pullRequestEvent) ToProvider() provider.PullRequest {
    canModify := pr.MaintainerCanModify
    if !canModify {
        canModify = pr.AllowMaintainerEdit
    }
    return provider.PullRequest{
        Number:              pr.Number,
        Title:               pr.Title,
        Body:                pr.Body,
        State:               pr.State,
        BaseRef:             pr.Base.Ref,
        BaseSHA:             pr.Base.SHA,
        HeadSHA:             pr.Head.SHA,
        Labels:              pr.Labels,
        User:                pr.User,
        RequestedReviewers:  pr.RequestedReviewers,
        MaintainerCanModify: canModify,
        Milestone:           pr.Milestone,
        UpdatedAt:           pr.UpdatedAt,
        Merged:              pr.Merged,
        MergeCommitSHA:      pr.MergeCommitSHA,
    }
}
