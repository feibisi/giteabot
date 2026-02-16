package gitea

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "sort"
    "strings"
    "time"

    "code.gitea.io/sdk/gitea"
    "github.com/go-gitea/giteabot/internal/provider"
)

type Client struct {
    client    *gitea.Client
    owner     string
    repo      string
    fork      string
    labels    provider.Labels
    labelIDs  map[string]int64
    apiBase   string
    token     string
    httpClient *http.Client
}

func New(apiBase, owner, name, fork, token string, labels provider.Labels) (*Client, error) {
    sdkClient, err := gitea.NewClient(apiBase, gitea.SetToken(token))
    if err != nil {
        return nil, err
    }
    return &Client{
        client:    sdkClient,
        owner:     owner,
        repo:      name,
        fork:      fork,
        labels:    labels,
        apiBase:   strings.TrimRight(apiBase, "/"),
        token:     token,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }, nil
}

func (c *Client) Name() string {
    return "gitea"
}

func (c *Client) FetchCurrentUser() (provider.User, error) {
    user, _, err := c.client.GetMyUserInfo()
    if err != nil {
        return provider.User{}, err
    }
    return provider.User{Login: user.UserName, Email: user.Email}, nil
}

func (c *Client) FetchMilestones() ([]provider.Version, error) {
    versionsByMinor := map[string]provider.Version{}
    opt := gitea.ListMilestoneOption{State: gitea.StateOpen}
    opt.PageSize = 50
    for page := 1; ; page++ {
        opt.Page = page
        milestones, resp, err := c.client.ListRepoMilestones(c.owner, c.repo, opt)
        if err != nil {
            return nil, err
        }
        for _, milestone := range milestones {
            version, ok := provider.ParseVersionTitle(milestone.Title, provider.Milestone{Title: milestone.Title, Number: int(milestone.ID)})
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
    prs, err := c.listPullRequests(gitea.StateClosed)
    if err != nil {
        return nil, err
    }
    issues := []provider.Issue{}
    backportLabel := c.labels.BackportPrefix + "v" + version
    for _, pr := range prs {
        if !pr.HasMerged || pr.Base == nil || pr.Base.Ref != "main" {
            continue
        }
        labels := convertLabels(pr.Labels)
        if !hasLabel(labels, backportLabel) {
            continue
        }
        if hasLabel(labels, c.labels.BackportDone) || hasLabel(labels, c.labels.BackportManual) {
            continue
        }
        issues = append(issues, provider.Issue{
            Number:    int(pr.Index),
            Title:     pr.Title,
            Labels:    labels,
            UpdatedAt: safeTime(pr.Updated),
            IsPull:    true,
        })
    }
    return issues, nil
}

func (c *Client) BackportPullRequestExists(pullRequestNumber int, version string) (bool, error) {
    prs, err := c.listPullRequests(gitea.StateOpen)
    if err != nil {
        return false, err
    }
    token := fmt.Sprintf("#%d", pullRequestNumber)
    base := fmt.Sprintf("release/v%s", version)
    for _, pr := range prs {
        if pr.Base != nil && pr.Base.Ref == base && strings.Contains(pr.Title, token) {
            return true, nil
        }
    }
    if c.fork == "" {
        return false, nil
    }
    forkOwner, forkRepo := splitRepo(c.fork)
    if forkOwner == "" {
        return false, nil
    }
    branch := backportBranch(pullRequestNumber, version)
    _, resp, err := c.client.GetRepoBranch(forkOwner, forkRepo, branch)
    if err != nil {
        if resp != nil && resp.StatusCode == http.StatusNotFound {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

func (c *Client) FetchPullRequest(pullRequestNumber int) (provider.PullRequest, error) {
    pr, _, err := c.client.GetPullRequest(c.owner, c.repo, int64(pullRequestNumber))
    if err != nil {
        return provider.PullRequest{}, err
    }
    return convertPullRequest(pr), nil
}

func (c *Client) CreateBackportPullRequest(original provider.PullRequest, version provider.Version, headBranch string) error {
    description := fmt.Sprintf("Backport #%d by @%s", original.Number, original.User.Login)
    if original.Body != "" {
        description += "\n\n" + original.Body
    }
    forkOwner, _ := splitRepo(c.fork)
    pr, _, err := c.client.CreatePullRequest(c.owner, c.repo, gitea.CreatePullRequestOption{
        Title: fmt.Sprintf("%s (#%d)", original.Title, original.Number),
        Head:  fmt.Sprintf("%s:%s", forkOwner, headBranch),
        Base:  fmt.Sprintf("release/v%s", version.MajorMinor),
        Body:  description,
    })
    if err != nil {
        return err
    }
    labels := filterBackportLabels(original.Labels, c.labels)
    if len(labels) > 0 {
        if err := c.AddLabels(int(pr.Index), labels); err != nil {
            return err
        }
    }
    if original.User.Login != "" {
        _, _, _ = c.client.EditIssue(c.owner, c.repo, pr.Index, gitea.EditIssueOption{Assignees: []string{original.User.Login}})
    }
    if countBackportLabels(original.Labels, c.labels) == 1 {
        _ = c.AddLabels(original.Number, []string{c.labels.BackportDone})
    }
    return nil
}

func (c *Client) AddLabels(number int, labels []string) error {
    ids, err := c.labelIDsFor(labels)
    if err != nil {
        return err
    }
    _, _, err = c.client.AddIssueLabels(c.owner, c.repo, int64(number), gitea.IssueLabelsOption{Labels: ids})
    return err
}

func (c *Client) RemoveLabel(number int, label string) error {
    id, err := c.labelID(label)
    if err != nil {
        return err
    }
    _, err = c.client.DeleteIssueLabel(c.owner, c.repo, int64(number), id)
    return err
}

func (c *Client) AddComment(number int, comment string) error {
    _, _, err := c.client.CreateIssueComment(c.owner, c.repo, int64(number), gitea.CreateIssueCommentOption{Body: comment})
    return err
}

func (c *Client) FetchMergedWithLabel(label string) ([]provider.Issue, error) {
    issues, err := c.listIssues(gitea.StateClosed, gitea.IssueTypePull, label)
    if err != nil {
        return nil, err
    }
    merged := []provider.Issue{}
    for _, issue := range issues {
        pr, err := c.FetchPullRequest(issue.Number)
        if err != nil {
            return nil, err
        }
        if pr.Merged {
            merged = append(merged, issue)
        }
    }
    return merged, nil
}

func (c *Client) FetchTargeting(branch string) ([]provider.PullRequest, error) {
    prs, err := c.listPullRequests(gitea.StateAll)
    if err != nil {
        return nil, err
    }
    filtered := []provider.PullRequest{}
    for _, pr := range prs {
        if pr.Base != nil && pr.Base.Ref == branch {
            filtered = append(filtered, convertPullRequest(pr))
        }
    }
    return filtered, nil
}

func (c *Client) FetchBreakingWithoutLabel() ([]provider.Issue, error) {
    prs, err := c.listPullRequests(gitea.StateOpen)
    if err != nil {
        return nil, err
    }
    issues := []provider.Issue{}
    for _, pr := range prs {
        labels := convertLabels(pr.Labels)
        if strings.Contains(pr.Body, "## :warning: BREAKING") && !hasLabel(labels, c.labels.PrBreaking) {
            issues = append(issues, provider.Issue{
                Number:    int(pr.Index),
                Title:     pr.Title,
                Labels:    labels,
                UpdatedAt: safeTime(pr.Updated),
                IsPull:    true,
            })
        }
    }
    return issues, nil
}

func (c *Client) FetchPendingMerge() ([]provider.Issue, error) {
    return c.listIssues(gitea.StateOpen, gitea.IssueTypePull, c.labels.ReviewedWaitMerge)
}

func (c *Client) FetchOpenIssuesWithLabel(label string) ([]provider.Issue, error) {
    return c.listIssues(gitea.StateOpen, gitea.IssueTypeIssue, label)
}

func (c *Client) FetchOpenPullRequestsWithLabel(label string) ([]provider.Issue, error) {
    return c.listIssues(gitea.StateOpen, gitea.IssueTypePull, label)
}

func (c *Client) FetchPullRequestFileNames(pullRequestNumber int) (map[string]struct{}, error) {
    files := make(map[string]struct{})
    opt := gitea.ListPullRequestFilesOptions{}
    opt.PageSize = 50
    for page := 1; ; page++ {
        opt.Page = page
        changedFiles, resp, err := c.client.ListPullRequestFiles(c.owner, c.repo, int64(pullRequestNumber), opt)
        if err != nil {
            return nil, err
        }
        for _, file := range changedFiles {
            files[file.Filename] = struct{}{}
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
    }
    return files, nil
}

func (c *Client) NeedsUpdate(pullRequestNumber int) (bool, error) {
    pr, err := c.FetchPullRequest(pullRequestNumber)
    if err != nil {
        return false, err
    }
    if !pr.MaintainerCanModify || pr.State != string(gitea.StateOpen) {
        return false, nil
    }
    branch, _, err := c.client.GetRepoBranch(c.owner, c.repo, pr.BaseRef)
    if err != nil {
        return false, err
    }
    return pr.BaseSHA != branch.Commit.ID, nil
}

func (c *Client) UpdatePullRequest(pullRequestNumber int) error {
    payload := map[string]string{}
    body, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    req, err := http.NewRequest("POST", fmt.Sprintf("%s/repos/%s/%s/pulls/%d/update", c.apiBase, c.owner, c.repo, pullRequestNumber), bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "token "+c.token)
    req.Header.Set("Content-Type", "application/json")
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        return nil
    }
    var message struct {
        Message string `json:"message"`
    }
    _ = json.NewDecoder(resp.Body).Decode(&message)
    if message.Message != "" {
        return fmt.Errorf("%s", message.Message)
    }
    return fmt.Errorf("update pull request failed: %s", resp.Status)
}

func (c *Client) FetchUnmergedClosedPullRequestsWithMilestone(title string) ([]provider.PullRequest, error) {
    milestoneID, err := c.milestoneID(title)
    if err != nil {
        return nil, err
    }
    opt := gitea.ListIssueOption{State: gitea.StateClosed, Type: gitea.IssueTypePull, Milestones: []string{fmt.Sprintf("%d", milestoneID)}}
    opt.PageSize = 50
    prs := []provider.PullRequest{}
    for page := 1; ; page++ {
        opt.Page = page
        issues, resp, err := c.client.ListRepoIssues(c.owner, c.repo, opt)
        if err != nil {
            return nil, err
        }
        for _, issue := range issues {
            pr, err := c.FetchPullRequest(int(issue.Index))
            if err != nil {
                return nil, err
            }
            if !pr.Merged {
                prs = append(prs, pr)
            }
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
    }
    return prs, nil
}

func (c *Client) SetMilestone(prNumber int, milestoneID int) error {
    id := int64(milestoneID)
    _, _, err := c.client.EditIssue(c.owner, c.repo, int64(prNumber), gitea.EditIssueOption{Milestone: &id})
    return err
}

func (c *Client) RemoveMilestone(prNumber int) error {
    zero := int64(0)
    _, _, err := c.client.EditIssue(c.owner, c.repo, int64(prNumber), gitea.EditIssueOption{Milestone: &zero})
    return err
}

func (c *Client) GetPullRequestReviewers(pullRequest provider.PullRequest) (provider.Reviewers, error) {
    approvers := map[string]struct{}{}
    blockers := map[string]struct{}{}
    opt := gitea.ListPullReviewsOptions{}
    opt.PageSize = 50
    for page := 1; ; page++ {
        opt.Page = page
        reviews, resp, err := c.client.ListPullReviews(c.owner, c.repo, int64(pullRequest.Number), opt)
        if err != nil {
            return provider.Reviewers{}, err
        }
        for _, review := range reviews {
            if review.Reviewer == nil {
                continue
            }
            login := review.Reviewer.UserName
            switch review.State {
            case gitea.ReviewStateApproved:
                approvers[login] = struct{}{}
                delete(blockers, login)
            case gitea.ReviewStateRequestChanges:
                blockers[login] = struct{}{}
                delete(approvers, login)
            default:
            }
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
    }
    for _, reviewer := range pullRequest.RequestedReviewers {
        delete(approvers, reviewer.Login)
    }
    return provider.Reviewers{Approvers: approvers, Blockers: blockers}, nil
}

func (c *Client) SetCommitStatus(sha string, state string, description string) error {
    _, _, err := c.client.CreateStatus(c.owner, c.repo, sha, gitea.CreateStatusOption{
        State:       gitea.StatusState(state),
        Context:     "giteabot/lgtm",
        Description: description,
    })
    return err
}

func (c *Client) FetchClosedOldIssuesAndPullRequests(before time.Time) ([]provider.Issue, error) {
    opt := gitea.ListIssueOption{State: gitea.StateClosed, Type: gitea.IssueTypeAll}
    opt.PageSize = 50
    issues := []provider.Issue{}
    for page := 1; ; page++ {
        opt.Page = page
        results, resp, err := c.client.ListRepoIssues(c.owner, c.repo, opt)
        if err != nil {
            return nil, err
        }
        for _, issue := range results {
            if issue.IsLocked || issue.Closed == nil {
                continue
            }
            if issue.Closed.Before(before) {
                issues = append(issues, provider.Issue{
                    Number:    int(issue.Index),
                    Title:     issue.Title,
                    Labels:    convertLabels(issue.Labels),
                    UpdatedAt: issue.Updated,
                    IsPull:    issue.PullRequest != nil,
                    ClosedAt:  issue.Closed,
                })
            }
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
    }
    return issues, nil
}

func (c *Client) FetchLastComment(issueNumber int) (*provider.Comment, error) {
    opt := gitea.ListIssueCommentOptions{}
    opt.PageSize = 1
    opt.Page = 1
    comments, _, err := c.client.ListIssueComments(c.owner, c.repo, int64(issueNumber), opt)
    if err != nil {
        return nil, err
    }
    if len(comments) == 0 {
        return nil, nil
    }
    return &provider.Comment{CreatedAt: comments[0].Created}, nil
}

func (c *Client) LockIssue(issueNumber int, reason string) (bool, error) {
    _, err := c.client.LockIssue(c.owner, c.repo, int64(issueNumber), gitea.LockIssueOption{LockReason: reason})
    if err != nil {
        return false, err
    }
    return true, nil
}

func (c *Client) CloseIssue(issueNumber int) error {
    closed := gitea.StateClosed
    _, _, err := c.client.EditIssue(c.owner, c.repo, int64(issueNumber), gitea.EditIssueOption{State: &closed})
    return err
}

func (c *Client) ClosePullRequest(pullRequestNumber int) error {
    closed := gitea.StateClosed
    _, _, err := c.client.EditIssue(c.owner, c.repo, int64(pullRequestNumber), gitea.EditIssueOption{State: &closed})
    return err
}

func (c *Client) listPullRequests(state gitea.StateType) ([]*gitea.PullRequest, error) {
    opt := gitea.ListPullRequestsOptions{State: state}
    opt.PageSize = 50
    prs := []*gitea.PullRequest{}
    for page := 1; ; page++ {
        opt.Page = page
        results, resp, err := c.client.ListRepoPullRequests(c.owner, c.repo, opt)
        if err != nil {
            return nil, err
        }
        prs = append(prs, results...)
        if resp == nil || resp.NextPage == 0 {
            break
        }
    }
    return prs, nil
}

func (c *Client) listIssues(state gitea.StateType, issueType gitea.IssueType, label string) ([]provider.Issue, error) {
    opt := gitea.ListIssueOption{State: state, Type: issueType, Labels: []string{label}}
    opt.PageSize = 50
    issues := []provider.Issue{}
    for page := 1; ; page++ {
        opt.Page = page
        results, resp, err := c.client.ListRepoIssues(c.owner, c.repo, opt)
        if err != nil {
            return nil, err
        }
        for _, issue := range results {
            issues = append(issues, provider.Issue{
                Number:    int(issue.Index),
                Title:     issue.Title,
                Labels:    convertLabels(issue.Labels),
                UpdatedAt: issue.Updated,
                IsPull:    issue.PullRequest != nil,
                ClosedAt:  issue.Closed,
            })
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
    }
    return issues, nil
}

func (c *Client) milestoneID(title string) (int64, error) {
    opt := gitea.ListMilestoneOption{State: gitea.StateOpen}
    opt.PageSize = 50
    for page := 1; ; page++ {
        opt.Page = page
        milestones, resp, err := c.client.ListRepoMilestones(c.owner, c.repo, opt)
        if err != nil {
            return 0, err
        }
        for _, milestone := range milestones {
            if milestone.Title == title {
                return milestone.ID, nil
            }
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
    }
    return 0, fmt.Errorf("milestone not found: %s", title)
}

func (c *Client) labelID(name string) (int64, error) {
    ids, err := c.labelIDsFor([]string{name})
    if err != nil {
        return 0, err
    }
    if len(ids) == 0 {
        return 0, fmt.Errorf("label not found: %s", name)
    }
    return ids[0], nil
}

func (c *Client) labelIDsFor(labels []string) ([]int64, error) {
    if c.labelIDs == nil {
        if err := c.populateLabelCache(); err != nil {
            return nil, err
        }
    }
    ids := []int64{}
    for _, label := range labels {
        id, ok := c.labelIDs[label]
        if !ok {
            return nil, fmt.Errorf("label not found: %s", label)
        }
        ids = append(ids, id)
    }
    return ids, nil
}

func (c *Client) populateLabelCache() error {
    labels := map[string]int64{}
    opt := gitea.ListLabelsOptions{}
    opt.PageSize = 50
    for page := 1; ; page++ {
        opt.Page = page
        results, resp, err := c.client.ListRepoLabels(c.owner, c.repo, opt)
        if err != nil {
            return err
        }
        for _, label := range results {
            labels[label.Name] = label.ID
        }
        if resp == nil || resp.NextPage == 0 {
            break
        }
    }
    c.labelIDs = labels
    return nil
}

func convertPullRequest(pr *gitea.PullRequest) provider.PullRequest {
    requested := make([]provider.User, 0, len(pr.RequestedReviewers))
    for _, reviewer := range pr.RequestedReviewers {
        requested = append(requested, provider.User{Login: reviewer.UserName, Email: reviewer.Email})
    }
    var milestone *provider.Milestone
    if pr.Milestone != nil {
        milestone = &provider.Milestone{Title: pr.Milestone.Title, Number: int(pr.Milestone.ID)}
    }
    user := provider.User{}
    if pr.Poster != nil {
        user = provider.User{Login: pr.Poster.UserName, Email: pr.Poster.Email}
    }
    return provider.PullRequest{
        Number:              int(pr.Index),
        Title:               pr.Title,
        Body:                pr.Body,
        State:               string(pr.State),
        BaseRef:             safeBaseRef(pr.Base),
        BaseSHA:             safeBaseSha(pr.Base),
        HeadSHA:             safeHeadSha(pr.Head),
        Labels:              convertLabels(pr.Labels),
        User:                user,
        RequestedReviewers:  requested,
        MaintainerCanModify: pr.AllowMaintainerEdit,
        Milestone:           milestone,
        UpdatedAt:           safeTime(pr.Updated),
        Merged:              pr.HasMerged,
        MergeCommitSHA:      safeMergeCommit(pr.MergedCommitID),
    }
}

func convertLabels(labels []*gitea.Label) []provider.Label {
    converted := make([]provider.Label, 0, len(labels))
    for _, label := range labels {
        converted = append(converted, provider.Label{Name: label.Name})
    }
    return converted
}

func safeTime(t *time.Time) time.Time {
    if t == nil {
        return time.Time{}
    }
    return *t
}

func safeBaseRef(info *gitea.PRBranchInfo) string {
    if info == nil {
        return ""
    }
    return info.Ref
}

func safeBaseSha(info *gitea.PRBranchInfo) string {
    if info == nil {
        return ""
    }
    return info.Sha
}

func safeHeadSha(info *gitea.PRBranchInfo) string {
    if info == nil {
        return ""
    }
    return info.Sha
}

func safeMergeCommit(sha *string) string {
    if sha == nil {
        return ""
    }
    return *sha
}

func hasLabel(labels []provider.Label, name string) bool {
    for _, label := range labels {
        if label.Name == name {
            return true
        }
    }
    return false
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
