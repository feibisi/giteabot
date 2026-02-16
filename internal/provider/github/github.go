package github

import (
    "context"
    "fmt"
    "net/http"
    "sort"
    "strings"
    "time"

    "github.com/go-gitea/giteabot/internal/provider"
    "github.com/google/go-github/v82/github"
    "golang.org/x/oauth2"
)

type Client struct {
    client *github.Client
    ctx    context.Context
    owner  string
    repo   string
    fork   string
    labels provider.Labels
}

func New(apiBase, owner, name, fork, token string, labels provider.Labels) (*Client, error) {
    ctx := context.Background()
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    httpClient := oauth2.NewClient(ctx, ts)

    ghClient := newGitHubClient(apiBase, httpClient)
    return &Client{
        client: ghClient,
        ctx:    ctx,
        owner:  owner,
        repo:   name,
        fork:   fork,
        labels: labels,
    }, nil
}

func newGitHubClient(apiBase string, httpClient *http.Client) *github.Client {
    trimmed := strings.TrimRight(apiBase, "/")
    if trimmed == "" || trimmed == "https://api.github.com" {
        return github.NewClient(httpClient)
    }
    client, err := github.NewEnterpriseClient(trimmed, trimmed, httpClient)
    if err != nil {
        return github.NewClient(httpClient)
    }
    return client
}

func (c *Client) Name() string {
    return "github"
}

func (c *Client) FetchCurrentUser() (provider.User, error) {
    user, _, err := c.client.Users.Get(c.ctx, "")
    if err != nil {
        return provider.User{}, err
    }
    return provider.User{Login: user.GetLogin(), Email: user.GetEmail()}, nil
}

func (c *Client) FetchMilestones() ([]provider.Version, error) {
    versionsByMinor := map[string]provider.Version{}
    opt := &github.MilestoneListOptions{State: "open", ListOptions: github.ListOptions{PerPage: 100}}
    for {
        milestones, resp, err := c.client.Issues.ListMilestones(c.ctx, c.owner, c.repo, opt)
        if err != nil {
            return nil, err
        }
        for _, milestone := range milestones {
            if milestone.Title == nil || milestone.Number == nil {
                continue
            }
            version, ok := provider.ParseVersionTitle(milestone.GetTitle(), provider.Milestone{
                Title:  milestone.GetTitle(),
                Number: milestone.GetNumber(),
            })
            if !ok {
                continue
            }
            existing, exists := versionsByMinor[version.MajorMinor]
            if !exists || provider.ComparePatch(version, existing) < 0 {
                versionsByMinor[version.MajorMinor] = version
            }
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
        opt.Page = resp.NextPage
    }
    versions := make([]provider.Version, 0, len(versionsByMinor))
    for _, version := range versionsByMinor {
        versions = append(versions, version)
    }
    sort.Slice(versions, func(i, j int) bool {
        return provider.CompareMajorMinor(versions[i], versions[j]) > 0
    })
    return versions, nil
}

func (c *Client) FetchCandidates(version string) ([]provider.Issue, error) {
    query := fmt.Sprintf(
        "is:pr is:merged base:main label:%sv%s -label:%s -label:%s repo:%s/%s",
        c.labels.BackportPrefix,
        version,
        c.labels.BackportDone,
        c.labels.BackportManual,
        c.owner,
        c.repo,
    )
    return c.searchIssues(query)
}

func (c *Client) FetchMergedWithLabel(label string) ([]provider.Issue, error) {
    query := fmt.Sprintf("is:pr is:merged label:%s repo:%s/%s", label, c.owner, c.repo)
    return c.searchIssues(query)
}

func (c *Client) FetchOpenIssuesWithLabel(label string) ([]provider.Issue, error) {
    query := fmt.Sprintf("is:issue is:open label:%s repo:%s/%s", label, c.owner, c.repo)
    return c.searchIssues(query)
}

func (c *Client) FetchOpenPullRequestsWithLabel(label string) ([]provider.Issue, error) {
    query := fmt.Sprintf("is:pr is:open label:%s repo:%s/%s", label, c.owner, c.repo)
    return c.searchIssues(query)
}

func (c *Client) FetchPendingMerge() ([]provider.Issue, error) {
    query := fmt.Sprintf("is:pr is:open label:%s sort:created-asc repo:%s/%s", c.labels.ReviewedWaitMerge, c.owner, c.repo)
    return c.searchIssues(query)
}

func (c *Client) FetchTargeting(branch string) ([]provider.PullRequest, error) {
    query := fmt.Sprintf("is:pr base:%s repo:%s/%s", branch, c.owner, c.repo)
    issues, err := c.searchIssues(query)
    if err != nil {
        return nil, err
    }
    prs := make([]provider.PullRequest, 0, len(issues))
    for _, issue := range issues {
        pr, err := c.FetchPullRequest(issue.Number)
        if err != nil {
            return nil, err
        }
        prs = append(prs, pr)
    }
    return prs, nil
}

func (c *Client) FetchUnmergedClosedPullRequestsWithMilestone(title string) ([]provider.PullRequest, error) {
    query := fmt.Sprintf("is:pr is:closed is:unmerged milestone:%s repo:%s/%s", title, c.owner, c.repo)
    issues, err := c.searchIssues(query)
    if err != nil {
        return nil, err
    }
    prs := make([]provider.PullRequest, 0, len(issues))
    for _, issue := range issues {
        pr, err := c.FetchPullRequest(issue.Number)
        if err != nil {
            return nil, err
        }
        prs = append(prs, pr)
    }
    return prs, nil
}

func (c *Client) FetchBreakingWithoutLabel() ([]provider.Issue, error) {
    query := fmt.Sprintf("is:pr \"## :warning: BREAKING\" -label:%s repo:%s/%s", c.labels.PrBreaking, c.owner, c.repo)
    return c.searchIssues(query)
}

func (c *Client) FetchPullRequestFileNames(pullRequestNumber int) (map[string]struct{}, error) {
    files := make(map[string]struct{})
    opt := &github.ListOptions{PerPage: 100}
    for {
        commitFiles, resp, err := c.client.PullRequests.ListFiles(c.ctx, c.owner, c.repo, pullRequestNumber, opt)
        if err != nil {
            return nil, err
        }
        for _, file := range commitFiles {
            if file.Filename != nil {
                files[file.GetFilename()] = struct{}{}
            }
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
        opt.Page = resp.NextPage
    }
    return files, nil
}

func (c *Client) FetchPullRequest(pullRequestNumber int) (provider.PullRequest, error) {
    pr, _, err := c.client.PullRequests.Get(c.ctx, c.owner, c.repo, pullRequestNumber)
    if err != nil {
        return provider.PullRequest{}, err
    }
    return convertPullRequest(pr), nil
}

func (c *Client) NeedsUpdate(pullRequestNumber int) (bool, error) {
    pr, _, err := c.client.PullRequests.Get(c.ctx, c.owner, c.repo, pullRequestNumber)
    if err != nil {
        return false, err
    }
    if !pr.GetMaintainerCanModify() || pr.GetState() != "open" {
        return false, nil
    }
    branch, _, err := c.client.Repositories.GetBranch(c.ctx, c.owner, c.repo, pr.GetBase().GetRef(), 0)
    if err != nil {
        return false, err
    }
    return pr.GetBase().GetSHA() != branch.GetCommit().GetSHA(), nil
}

func (c *Client) UpdatePullRequest(pullRequestNumber int) error {
    pr, _, err := c.client.PullRequests.Get(c.ctx, c.owner, c.repo, pullRequestNumber)
    if err != nil {
        return err
    }
    opts := &github.PullRequestBranchUpdateOptions{ExpectedHeadSHA: github.String(pr.GetHead().GetSHA())}
    _, _, err = c.client.PullRequests.UpdateBranch(c.ctx, c.owner, c.repo, pullRequestNumber, opts)
    if err != nil {
        if _, ok := err.(*github.AcceptedError); ok {
            return nil
        }
        return err
    }
    return nil
}

func (c *Client) RemoveLabel(number int, label string) error {
    _, err := c.client.Issues.RemoveLabelForIssue(c.ctx, c.owner, c.repo, number, label)
    return err
}

func (c *Client) AddLabels(number int, labels []string) error {
    _, _, err := c.client.Issues.AddLabelsToIssue(c.ctx, c.owner, c.repo, number, labels)
    return err
}

func (c *Client) AddComment(number int, comment string) error {
    _, _, err := c.client.Issues.CreateComment(c.ctx, c.owner, c.repo, number, &github.IssueComment{Body: github.String(comment)})
    return err
}

func (c *Client) SetMilestone(prNumber int, milestoneID int) error {
    _, _, err := c.client.Issues.Edit(c.ctx, c.owner, c.repo, prNumber, &github.IssueRequest{Milestone: github.Int(milestoneID)})
    return err
}

func (c *Client) RemoveMilestone(prNumber int) error {
    _, _, err := c.client.Issues.Edit(c.ctx, c.owner, c.repo, prNumber, &github.IssueRequest{Milestone: github.Int(0)})
    return err
}

func (c *Client) BackportPullRequestExists(pullRequestNumber int, version string) (bool, error) {
    query := fmt.Sprintf("is:pr is:open repo:%s/%s base:release/v%s %d in:title", c.owner, c.repo, version, pullRequestNumber)
    issues, err := c.searchIssues(query)
    if err != nil {
        return false, err
    }
    if len(issues) > 0 {
        return true, nil
    }
    if c.fork == "" {
        return false, nil
    }
    forkOwner, forkRepo := splitRepo(c.fork)
    if forkOwner == "" {
        return false, nil
    }
    branch := backportBranch(pullRequestNumber, version)
    _, resp, err := c.client.Repositories.GetBranch(c.ctx, forkOwner, forkRepo, branch, 0)
    if err != nil {
        if resp != nil && resp.StatusCode == http.StatusNotFound {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

func (c *Client) CreateBackportPullRequest(original provider.PullRequest, version provider.Version, headBranch string) error {
    description := fmt.Sprintf("Backport #%d by @%s", original.Number, original.User.Login)
    if original.Body != "" {
        description += "\n\n" + original.Body
    }
    forkOwner, _ := splitRepo(c.fork)
    pr, _, err := c.client.PullRequests.Create(c.ctx, c.owner, c.repo, &github.NewPullRequest{
        Title:               github.String(fmt.Sprintf("%s (#%d)", original.Title, original.Number)),
        Head:                github.String(fmt.Sprintf("%s:%s", forkOwner, headBranch)),
        Base:                github.String(fmt.Sprintf("release/v%s", version.MajorMinor)),
        Body:                github.String(description),
        MaintainerCanModify: github.Bool(true),
    })
    if err != nil {
        return err
    }
    labels := filterBackportLabels(original.Labels, c.labels)
    if len(labels) > 0 {
        if err := c.AddLabels(pr.GetNumber(), labels); err != nil {
            return err
        }
    }
    if original.User.Login != "" {
        _, _, _ = c.client.Issues.AddAssignees(c.ctx, c.owner, c.repo, pr.GetNumber(), []string{original.User.Login})
    }
    reviewers, err := c.GetPullRequestReviewers(original)
    if err == nil && len(reviewers.Approvers) > 0 {
        approvers := make([]string, 0, len(reviewers.Approvers))
        for login := range reviewers.Approvers {
            approvers = append(approvers, login)
        }
        _, _, _ = c.client.PullRequests.RequestReviewers(c.ctx, c.owner, c.repo, pr.GetNumber(), github.ReviewersRequest{Reviewers: approvers})
    }
    if countBackportLabels(original.Labels, c.labels) == 1 {
        _ = c.AddLabels(original.Number, []string{c.labels.BackportDone})
    }
    return nil
}

func (c *Client) GetPullRequestReviewers(pullRequest provider.PullRequest) (provider.Reviewers, error) {
    approvers := map[string]struct{}{}
    blockers := map[string]struct{}{}
    opt := &github.ListOptions{PerPage: 100}
    for {
        reviews, resp, err := c.client.PullRequests.ListReviews(c.ctx, c.owner, c.repo, pullRequest.Number, opt)
        if err != nil {
            return provider.Reviewers{}, err
        }
        for _, review := range reviews {
            if review.User == nil {
                continue
            }
            login := review.User.GetLogin()
            switch review.GetState() {
            case "APPROVED":
                approvers[login] = struct{}{}
                delete(blockers, login)
            case "CHANGES_REQUESTED":
                blockers[login] = struct{}{}
                delete(approvers, login)
            case "DISMISSED":
                delete(approvers, login)
                delete(blockers, login)
            default:
            }
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
        opt.Page = resp.NextPage
    }
    for _, reviewer := range pullRequest.RequestedReviewers {
        delete(approvers, reviewer.Login)
    }
    return provider.Reviewers{Approvers: approvers, Blockers: blockers}, nil
}

func (c *Client) SetCommitStatus(sha string, state string, description string) error {
    _, _, err := c.client.Repositories.CreateStatus(c.ctx, c.owner, c.repo, sha, github.RepoStatus{
        State:       github.String(state),
        Context:     github.String("giteabot/lgtm"),
        Description: github.String(description),
    })
    return err
}

func (c *Client) FetchClosedOldIssuesAndPullRequests(before time.Time) ([]provider.Issue, error) {
    query := fmt.Sprintf("is:closed is:unlocked closed:<%s repo:%s/%s", before.Format(time.RFC3339), c.owner, c.repo)
    return c.searchIssues(query)
}

func (c *Client) FetchLastComment(issueNumber int) (*provider.Comment, error) {
    opts := &github.IssueListCommentsOptions{
        ListOptions: github.ListOptions{PerPage: 1},
        Sort:        github.String("created"),
        Direction:   github.String("desc"),
    }
    comments, _, err := c.client.Issues.ListComments(c.ctx, c.owner, c.repo, issueNumber, opts)
    if err != nil {
        return nil, err
    }
    if len(comments) == 0 {
        return nil, nil
    }
    return &provider.Comment{CreatedAt: comments[0].GetCreatedAt().Time}, nil
}

func (c *Client) LockIssue(issueNumber int, reason string) (bool, error) {
    _, err := c.client.Issues.Lock(c.ctx, c.owner, c.repo, issueNumber, &github.LockIssueOptions{LockReason: reason})
    if err != nil {
        return false, err
    }
    return true, nil
}

func (c *Client) CloseIssue(issueNumber int) error {
    _, _, err := c.client.Issues.Edit(c.ctx, c.owner, c.repo, issueNumber, &github.IssueRequest{State: github.String("closed")})
    return err
}

func (c *Client) ClosePullRequest(pullRequestNumber int) error {
    _, _, err := c.client.PullRequests.Edit(c.ctx, c.owner, c.repo, pullRequestNumber, &github.PullRequest{State: github.String("closed")})
    return err
}

func (c *Client) searchIssues(query string) ([]provider.Issue, error) {
    results := []provider.Issue{}
    opt := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 100}}
    for {
        searchResult, resp, err := c.client.Search.Issues(c.ctx, query, opt)
        if err != nil {
            return nil, err
        }
        for _, issue := range searchResult.Issues {
            var closedAt *time.Time
            if issue.ClosedAt != nil {
                t := issue.ClosedAt.Time
                closedAt = &t
            }
            results = append(results, provider.Issue{
                Number:    issue.GetNumber(),
                Title:     issue.GetTitle(),
                Labels:    convertLabels(issue.Labels),
                UpdatedAt: issue.GetUpdatedAt().Time,
                IsPull:    issue.IsPullRequest(),
                ClosedAt:  closedAt,
            })
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
        opt.Page = resp.NextPage
    }
    return results, nil
}

func convertPullRequest(pr *github.PullRequest) provider.PullRequest {
    labels := convertLabels(pr.Labels)
    requested := make([]provider.User, 0, len(pr.RequestedReviewers))
    for _, reviewer := range pr.RequestedReviewers {
        requested = append(requested, provider.User{Login: reviewer.GetLogin()})
    }
    var milestone *provider.Milestone
    if pr.Milestone != nil {
        milestone = &provider.Milestone{Title: pr.Milestone.GetTitle(), Number: pr.Milestone.GetNumber()}
    }
    return provider.PullRequest{
        Number:              pr.GetNumber(),
        Title:               pr.GetTitle(),
        Body:                pr.GetBody(),
        State:               pr.GetState(),
        BaseRef:             pr.GetBase().GetRef(),
        BaseSHA:             pr.GetBase().GetSHA(),
        HeadSHA:             pr.GetHead().GetSHA(),
        Labels:              labels,
        User:                provider.User{Login: pr.GetUser().GetLogin()},
        RequestedReviewers:  requested,
        MaintainerCanModify: pr.GetMaintainerCanModify(),
        Milestone:           milestone,
        UpdatedAt:           pr.GetUpdatedAt().Time,
        Merged:              pr.GetMerged(),
        MergeCommitSHA:      pr.GetMergeCommitSHA(),
    }
}

func convertLabels(labels []*github.Label) []provider.Label {
    converted := make([]provider.Label, 0, len(labels))
    for _, label := range labels {
        converted = append(converted, provider.Label{Name: label.GetName()})
    }
    return converted
}

func backportBranch(prNumber int, version string) string {
    return fmt.Sprintf("backport-%d-v%s", prNumber, version)
}

func filterBackportLabels(labels []provider.Label, labelConfig provider.Labels) []string {
    filtered := make([]string, 0, len(labels))
    for _, label := range labels {
        if strings.HasPrefix(label.Name, labelConfig.LGTMPrefix) ||
            strings.HasPrefix(label.Name, labelConfig.BackportPrefix) ||
            strings.HasPrefix(label.Name, labelConfig.ReviewedPrefix) ||
            strings.HasPrefix(label.Name, labelConfig.SizePrefix) ||
            strings.HasPrefix(label.Name, labelConfig.PrPrefix) {
            continue
        }
        filtered = append(filtered, label.Name)
    }
    return filtered
}

func countBackportLabels(labels []provider.Label, labelConfig provider.Labels) int {
    count := 0
    for _, label := range labels {
        if strings.HasPrefix(label.Name, labelConfig.BackportPrefix) {
            count++
        }
    }
    return count
}

func splitRepo(repo string) (string, string) {
    parts := strings.Split(repo, "/")
    if len(parts) != 2 {
        return "", ""
    }
    return parts[0], parts[1]
}
