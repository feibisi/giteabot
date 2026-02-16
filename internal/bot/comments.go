package bot

import (
    "strings"

    "github.com/go-gitea/giteabot/internal/provider"
)

func (b *Bot) CommentIfTranslationsChanged(pr provider.PullRequest) error {
    files, err := b.Provider.FetchPullRequestFileNames(pr.Number)
    if err != nil {
        return err
    }
    for name := range files {
        if strings.HasPrefix(name, "options/locale/") &&
            (strings.HasSuffix(name, ".ini") || strings.HasSuffix(name, ".json")) &&
            !strings.HasSuffix(name, "en-US.ini") && !strings.HasSuffix(name, "en-US.json") {
            comment := "@" + pr.User.Login + " I noticed you've updated the locales for non-English languages. These will be overwritten during the sync from our translation tool Crowdin. If you'd like to contribute your translations, please visit https://crowdin.com/project/gitea. Please revert the changes done on these files. :tea:"
            return b.Provider.AddComment(pr.Number, comment)
        }
    }
    return nil
}
