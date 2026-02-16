package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strings"

    "github.com/go-gitea/giteabot/internal/bot"
    "github.com/go-gitea/giteabot/internal/config"
    "github.com/go-gitea/giteabot/internal/provider"
    "github.com/go-gitea/giteabot/internal/provider/gitea"
    "github.com/go-gitea/giteabot/internal/provider/github"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("config error: %v", err)
    }
    prov, err := buildProvider(cfg)
    if err != nil {
        log.Fatalf("provider error: %v", err)
    }
    app := bot.New(cfg, prov)

    switch cfg.Mode {
    case "server":
        log.Printf("listening on %s", cfg.ListenAddr)
        log.Fatal(runServer(cfg, app))
    case "maintenance":
        if err := runMaintenance(app); err != nil {
            log.Fatalf("maintenance error: %v", err)
        }
    default:
        if err := runEvent(cfg, app); err != nil {
            log.Fatalf("event error: %v", err)
        }
    }
}

func buildProvider(cfg config.Config) (provider.Provider, error) {
    switch cfg.Provider {
    case "gitea":
        return gitea.New(cfg.APIBaseURL, cfg.RepoOwner, cfg.RepoName, cfg.Fork, cfg.Token, cfg.Labels)
    case "github":
        return github.New(cfg.APIBaseURL, cfg.RepoOwner, cfg.RepoName, cfg.Fork, cfg.Token, cfg.Labels)
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
}

func runEvent(cfg config.Config, app *bot.Bot) error {
    if cfg.EventPath == "" {
        return fmt.Errorf("GITEABOT_EVENT_PATH is required in event mode")
    }
    data, err := os.ReadFile(cfg.EventPath)
    if err != nil {
        return err
    }
    event, err := bot.ParseEvent(data)
    if err != nil {
        return err
    }
    return app.HandleEvent(cfg.EventName, event)
}

func runMaintenance(app *bot.Bot) error {
    if app.Config.Features.LabelMaintenance {
        if err := app.RunLabelMaintenance(); err != nil {
            return err
        }
    }
    if app.Config.Features.MergeQueue {
        if err := app.RunMergeQueue(); err != nil {
            return err
        }
    }
    if app.Config.Features.Locking {
        if err := app.RunLocking(); err != nil {
            return err
        }
    }
    if app.Config.Features.Feedback {
        if err := app.RunFeedback(); err != nil {
            return err
        }
    }
    if app.Config.Features.LastCall {
        return app.RunLastCall()
    }
    return nil
}

func runServer(cfg config.Config, app *bot.Bot) error {
    http.HandleFunc(cfg.TriggerPath, func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }
        body, err := io.ReadAll(r.Body)
        if err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        if cfg.WebhookSecret != "" {
            if !verifySignature(cfg.WebhookSecret, r.Header, body) {
                w.WriteHeader(http.StatusUnauthorized)
                return
            }
        }
        eventName := r.Header.Get("X-GitHub-Event")
        if eventName == "" {
            eventName = r.Header.Get("X-Gitea-Event")
        }
        if eventName == "" {
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        event, err := bot.ParseEvent(body)
        if err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        if err := app.HandleEvent(eventName, event); err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            _, _ = w.Write([]byte(err.Error()))
            return
        }
        _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    })
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    })
    return http.ListenAndServe(cfg.ListenAddr, nil)
}

func verifySignature(secret string, headers http.Header, body []byte) bool {
    signature := headers.Get("X-Hub-Signature-256")
    if signature == "" {
        signature = headers.Get("X-Gitea-Signature")
    }
    if signature == "" {
        return false
    }
    signature = strings.TrimPrefix(signature, "sha256=")
    mac := hmac.New(sha256.New, []byte(secret))
    _, _ = mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
