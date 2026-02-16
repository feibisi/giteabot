package gitops

import (
    "bytes"
    "fmt"
    "net/url"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/go-gitea/giteabot/internal/config"
)

type Repo struct {
    Path string
}

func Initialize(cfg config.Config, username, email string) (Repo, error) {
    if err := os.MkdirAll(cfg.WorkDir, 0o755); err != nil {
        return Repo{}, err
    }
    repoPath := filepath.Join(cfg.WorkDir, "gitea")
    _ = os.RemoveAll(repoPath)

    cloneURL, err := buildCloneURL(cfg)
    if err != nil {
        return Repo{}, err
    }
    if err := run("", "git", "clone", cloneURL, repoPath); err != nil {
        return Repo{}, err
    }

    upstreamURL, err := buildUpstreamURL(cfg)
    if err != nil {
        return Repo{}, err
    }
    if err := run(repoPath, "git", "remote", "add", "upstream", upstreamURL); err != nil {
        return Repo{}, err
    }

    if username != "" {
        if err := run(repoPath, "git", "config", "user.name", username); err != nil {
            return Repo{}, err
        }
    }
    if email == "" {
        email = "teabot@gitea.io"
    }
    if err := run(repoPath, "git", "config", "user.email", email); err != nil {
        return Repo{}, err
    }
    return Repo{Path: repoPath}, nil
}

func CherryPick(repo Repo, commitHash string, prNumber int, version string) (bool, error) {
    if err := run(repo.Path, "git", "fetch", "upstream", "main"); err != nil {
        return false, err
    }
    if err := run(repo.Path, "git", "fetch", "upstream", fmt.Sprintf("release/v%s", version)); err != nil {
        return false, err
    }
    branchName := backportBranch(prNumber, version)
    if err := run(repo.Path, "git", "checkout", fmt.Sprintf("upstream/release/v%s", version), "-b", branchName); err != nil {
        return false, err
    }
    if err := run(repo.Path, "git", "cherry-pick", commitHash); err != nil {
        _ = run(repo.Path, "git", "cherry-pick", "--abort")
        return false, nil
    }
    if err := run(repo.Path, "git", "push", "origin", branchName); err != nil {
        return false, err
    }
    return true, nil
}

func backportBranch(prNumber int, version string) string {
    return fmt.Sprintf("backport-%d-v%s", prNumber, version)
}

func buildCloneURL(cfg config.Config) (string, error) {
    if cfg.Fork == "" {
        return "", fmt.Errorf("GITEABOT_FORK must be set for backports")
    }
    host, err := hostFromAPI(cfg.APIBaseURL)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("https://%s@%s/%s.git", cfg.Token, host, cfg.Fork), nil
}

func buildUpstreamURL(cfg config.Config) (string, error) {
    host, err := hostFromAPI(cfg.APIBaseURL)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("https://%s/%s/%s.git", host, cfg.RepoOwner, cfg.RepoName), nil
}

func hostFromAPI(apiBase string) (string, error) {
    parsed, err := url.Parse(apiBase)
    if err != nil {
        return "", err
    }
    host := parsed.Host
    if host == "" {
        host = strings.TrimPrefix(apiBase, "https://")
        host = strings.TrimPrefix(host, "http://")
        host = strings.Split(host, "/")[0]
    }
    if host == "api.github.com" {
        return "github.com", nil
    }
    return host, nil
}

func run(dir string, command string, args ...string) error {
    cmd := exec.Command(command, args...)
    if dir != "" {
        cmd.Dir = dir
    }
    var stdout bytes.Buffer
    var stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    if err := cmd.Run(); err != nil {
        output := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
        return fmt.Errorf("%s %v failed: %s", command, args, output)
    }
    return nil
}
