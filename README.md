# giteabot

giteabot is a Go-based helper bot for Gitea and GitHub repositories. It is a drop-in replacement for the
GiteaBot backporter workflow and can run as a GitHub Action, a Gitea Action, or a standalone webhook service.

## Behavior

### Backport creation

giteabot fetches the current Gitea development versions from milestones. It looks for merged PRs
with the label `backport/v{gitea_version}` but without `backport/done` or `backport/manual`.
It will cherry-pick the merge commit onto the release branch, push to your fork, and open a new
backport pull request with labels copied from the original PR.

### Label maintenance

giteabot removes `reviewed/wait-merge`, `reviewed/prioritize-merge`, and `pr/last-call` from merged
pull requests. It also removes `backport/*` labels from pull requests that already target release
branches. If a PR description contains `## :warning: BREAKING`, it adds the `pr/breaking` label.

### Merge queue synchronization

giteabot looks for PRs with `reviewed/wait-merge` and updates their branches. If a merge conflict
is detected during the update, the PR is removed from the merge queue.

### Milestone maintenance

When a pull request is created, giteabot assigns a milestone based on the target branch. For PRs
targeting `main`, the latest release milestone is used. It also removes milestones from closed,
unmerged pull requests.

### LGTM

giteabot tracks approvals and updates the `lgtm/*` labels (`need 2`, `need 1`, `done`, or `blocked`)
plus a commit status context `giteabot/lgtm` indicating whether the PR is ready.

### Comments

If a PR changes non-English translation files under `options/locale/`, giteabot comments to direct
contributors to Crowdin and asks them to revert the changes.

### Locks

giteabot locks issues and pull requests that have been closed for 3 months. If there was discussion
within the last two weeks, it posts a follow-up comment asking to open a new issue.

### Feedback

Issues labeled `issue/needs-feedback` are closed if no activity occurs for a month.

### Maintainer commands

Maintainers can apply `giteabot/update-branch` to request an update of the PR branch.

### Last call

PRs labeled `pr/last-call` are closed after two weeks of inactivity. After one week, giteabot
notifies the technical oversight committee if only one approval is missing.

## Configuration

giteabot reads settings from environment variables (Action inputs map to these values).

Required:

- `GITEABOT_TOKEN`: API token for GitHub or Gitea.

Optional in Actions (auto-detected), required elsewhere:

- `GITEABOT_REPO`: target repository in `owner/name` format (auto-detected from `GITHUB_REPOSITORY` or
  `GITEA_REPOSITORY` when running in Actions).

Optional:

- `GITEABOT_FORK`: fork repository for backport branches (required for backport creation).
- `GITEABOT_PROVIDER`: `github` or `gitea` (auto-detected when possible).
- `GITEABOT_API_BASE`: GitHub API base (`https://api.github.com`) or Gitea API base (`https://gitea.com/api/v1`).
- `GITEABOT_MODE`: `event` (default), `maintenance`, or `server`.
- `GITEABOT_EVENT_PATH`: event JSON path in `event` mode.
- `GITEABOT_EVENT_NAME`: event name string in `event` mode.
- `GITEABOT_WEBHOOK_SECRET`: webhook secret used in `server` mode.
- `GITEABOT_LISTEN`: listen address for `server` mode (default `:8000`).
- `GITEABOT_TRIGGER_PATH`: webhook path for `server` mode (default `/trigger`).

Feature toggles (default `true`):

- `GITEABOT_FEATURE_BACKPORT`
- `GITEABOT_FEATURE_LABEL_MAINTENANCE`
- `GITEABOT_FEATURE_MERGE_QUEUE`
- `GITEABOT_FEATURE_MILESTONES`
- `GITEABOT_FEATURE_LGTM`
- `GITEABOT_FEATURE_COMMENTS`
- `GITEABOT_FEATURE_LOCKING`
- `GITEABOT_FEATURE_FEEDBACK`
- `GITEABOT_FEATURE_LAST_CALL`
- `GITEABOT_FEATURE_PR_ACTIONS`

Label overrides (defaults match go-gitea/gitea):

- `GITEABOT_LABEL_BACKPORT_PREFIX`
- `GITEABOT_LABEL_BACKPORT_DONE`
- `GITEABOT_LABEL_BACKPORT_MANUAL`
- `GITEABOT_LABEL_REVIEWED_PREFIX`
- `GITEABOT_LABEL_REVIEWED_WAIT_MERGE`
- `GITEABOT_LABEL_REVIEWED_PRIORITIZE_MERGE`
- `GITEABOT_LABEL_PR_LAST_CALL`
- `GITEABOT_LABEL_PR_BREAKING`
- `GITEABOT_LABEL_LGTM_PREFIX`
- `GITEABOT_LABEL_LGTM_NEED_2`
- `GITEABOT_LABEL_LGTM_NEED_1`
- `GITEABOT_LABEL_LGTM_DONE`
- `GITEABOT_LABEL_LGTM_BLOCKED`
- `GITEABOT_LABEL_BOT_PREFIX`
- `GITEABOT_LABEL_UPDATE_BRANCH`
- `GITEABOT_LABEL_TOC_REMINDED`
- `GITEABOT_LABEL_NEEDS_FEEDBACK`
- `GITEABOT_LABEL_SIZE_PREFIX`
- `GITEABOT_LABEL_PR_PREFIX`

## GitHub usage (GitHub Actions)

Example workflow:

```yaml
name: giteabot
on:
  push:
    branches: [main]
  pull_request:
    types:
      - opened
      - labeled
      - unlabeled
      - synchronize
      - review_requested
      - review_request_removed
      - closed
  pull_request_review:

jobs:
  giteabot:
    runs-on: ubuntu-latest
    steps:
      - uses: your-org/giteabot@v1
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          repo: ${{ github.repository }}
          fork: your-user/gitea
          provider: github
          api_base: https://api.github.com
          mode: event
          event_path: ${{ github.event_path }}
          event_name: ${{ github.event_name }}
```

Scheduled maintenance (GitHub Actions):

```yaml
name: giteabot-maintenance
on:
  schedule:
    - cron: "0 3 * * *"
  workflow_dispatch:

jobs:
  giteabot:
    runs-on: ubuntu-latest
    steps:
      - uses: your-org/giteabot@v1
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          repo: ${{ github.repository }}
          fork: your-user/gitea
          provider: github
          api_base: https://api.github.com
          mode: maintenance
```

Notes:

- `fork` should be a fork of the target repo with write access.
- For backports, the token must be able to create branches and PRs.
- You can override labels and feature flags via `env` on the step.

## Gitea usage (Gitea Actions)

Example workflow:

```yaml
name: giteabot
on:
  push:
    branches: [main]
  pull_request:
    types:
      - opened
      - labeled
      - unlabeled
      - synchronize
      - review_requested
      - review_request_removed
      - closed
  pull_request_review:

jobs:
  giteabot:
    runs-on: ubuntu-latest
    steps:
      - uses: your-org/giteabot@v1
        with:
          token: ${{ secrets.GITEA_TOKEN }}
          repo: ${{ gitea.repository }}
          fork: your-user/gitea
          provider: gitea
          api_base: https://gitea.com/api/v1
          mode: event
          event_path: ${{ gitea.event_path }}
          event_name: ${{ gitea.event_name }}
```

Scheduled maintenance (Gitea Actions):

```yaml
name: giteabot-maintenance
on:
  schedule:
    - cron: "0 3 * * *"
  workflow_dispatch:

jobs:
  giteabot:
    runs-on: ubuntu-latest
    steps:
      - uses: your-org/giteabot@v1
        with:
          token: ${{ secrets.GITEA_TOKEN }}
          repo: ${{ gitea.repository }}
          fork: your-user/gitea
          provider: gitea
          api_base: https://gitea.com/api/v1
          mode: maintenance
```

Notes:

- Ensure the token can read/write issues, PRs, and statuses.
- `api_base` should match your Gitea server.
- You can override labels and feature flags via `env` on the step.

Example label/feature overrides:

```yaml
      - uses: your-org/giteabot@v1
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          repo: ${{ github.repository }}
        env:
          GITEABOT_FEATURE_BACKPORT: "false"
          GITEABOT_LABEL_BACKPORT_PREFIX: "bp/"
          GITEABOT_LABEL_REVIEWED_WAIT_MERGE: "queue/wait-merge"
```

## Server mode (webhooks)

Run a webhook server (default port `8000`):

```bash
GITEABOT_MODE=server \
GITEABOT_REPO=go-gitea/gitea \
GITEABOT_FORK=your-user/gitea \
GITEABOT_PROVIDER=github \
GITEABOT_API_BASE=https://api.github.com \
GITEABOT_TOKEN=... \
GITEABOT_WEBHOOK_SECRET=... \
./giteabot
```

Configure a webhook pointing to `http(s)://<host>:8000/trigger`.
