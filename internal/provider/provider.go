package provider

import "time"

type Provider interface {
    Name() string
    FetchCurrentUser() (User, error)
    FetchMilestones() ([]Version, error)
    FetchCandidates(version string) ([]Issue, error)
    BackportPullRequestExists(pullRequestNumber int, version string) (bool, error)
    FetchPullRequest(prNumber int) (PullRequest, error)
    CreateBackportPullRequest(original PullRequest, version Version, headBranch string) error
    AddLabels(number int, labels []string) error
    RemoveLabel(number int, label string) error
    AddComment(number int, comment string) error
    FetchMergedWithLabel(label string) ([]Issue, error)
    FetchTargeting(branch string) ([]PullRequest, error)
    FetchBreakingWithoutLabel() ([]Issue, error)
    FetchPendingMerge() ([]Issue, error)
    FetchOpenIssuesWithLabel(label string) ([]Issue, error)
    FetchOpenPullRequestsWithLabel(label string) ([]Issue, error)
    FetchPullRequestFileNames(pullRequestNumber int) (map[string]struct{}, error)
    NeedsUpdate(prNumber int) (bool, error)
    UpdatePullRequest(pullRequestNumber int) error
    FetchUnmergedClosedPullRequestsWithMilestone(title string) ([]PullRequest, error)
    SetMilestone(prNumber int, milestoneID int) error
    RemoveMilestone(prNumber int) error
    GetPullRequestReviewers(pullRequest PullRequest) (Reviewers, error)
    SetCommitStatus(sha string, state string, description string) error
    FetchClosedOldIssuesAndPullRequests(before time.Time) ([]Issue, error)
    FetchLastComment(issueNumber int) (*Comment, error)
    LockIssue(issueNumber int, reason string) (bool, error)
    CloseIssue(issueNumber int) error
    ClosePullRequest(pullRequestNumber int) error
}
