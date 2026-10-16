package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/google/go-github/v92/github"
)

type GithubEventMonitor struct {
	webhookSecretKey []byte
	gh               *github.Client
}

func NewGithubEventMonitor(webhookSecret string, gh *github.Client) *GithubEventMonitor {
	return &GithubEventMonitor{webhookSecretKey: []byte(webhookSecret), gh: gh}
}

func (s *GithubEventMonitor) Process(e *github.PullRequestEvent) {
	cfg := GetConfig()
	ref := e.PullRequest.Head.Ref
	u := fmt.Sprintf("repos/%v/%v/zipball/%s", cfg.GithubRepoOwner, cfg.GithubRepo, *ref)
	req, err := s.gh.NewRequest(context.Background(), "GET", u, nil)
	if err != nil {
		logger.Error("Request Downloading repo", "err", err, "ref", ref, "u", u)
		return
	}

	resp, err := s.gh.BareDo(req)
	if err != nil {
		logger.Error("Downloading repo", "err", err, "ref", ref)
		return
	}
	defer resp.Body.Close()
	logger.Info("Resp", "headers", resp.Header)

	fname := fmt.Sprintf("snapshots/%s.zip", *ref)
	f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		logger.Error("Opening file", "err", err)
		return
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		logger.Error("Copy", "err", err)
		return
	}

	logger.Info("File uploaded", "path", fname)
}

func (s *GithubEventMonitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, s.webhookSecretKey)
	if err != nil {
		logger.Error("Error validating payload", "error", err)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		logger.Error("Error parsing payload", "error", err)
		return
	}

	switch event := event.(type) {
	case *github.PullRequestEvent:
		s.Process(event)
	}
}

type AppConfig struct {
	GithubToken     string
	GithubRepo      string
	GithubRepoOwner string

	Host                string
	Port                string
	WebhookURL          string
	GithubWebhookSecret string
}

var (
	cfg    *AppConfig
	once   sync.Once
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)

func GetConfig() *AppConfig {
	once.Do(func() {
		cfg = &AppConfig{}

		githubWebhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
		panicIfMissingEnv(githubWebhookSecret, "GITHUB_WEBHOOK_SECRET")

		githubToken := os.Getenv("GITHUB_TOKEN")
		panicIfMissingEnv(githubToken, "GITHUB_TOKEN")

		githubRepo := os.Getenv("GITHUB_REPO")
		panicIfMissingEnv(githubRepo, "GITHUB_REPO")

		githubRepoOwner := os.Getenv("GITHUB_REPO_OWNER")
		panicIfMissingEnv(githubRepoOwner, "GITHUB_REPO_OWNER")

		host := os.Getenv("HOST")
		port := os.Getenv("PORT")

		if host == "" {
			host = "127.0.0.1"
		}

		if port == "" {
			port = "8080"
		}

		webhookUrl := os.Getenv("WEBHOOK_URL")
		if webhookUrl == "" {
			webhookUrl = fmt.Sprintf("http://%s:%s/webhook", host, port)
		}

		cfg.GithubWebhookSecret = githubWebhookSecret
		cfg.Port = port
		cfg.Host = host
		cfg.WebhookURL = webhookUrl
		cfg.Host = host
		cfg.GithubToken = githubToken
		cfg.GithubRepoOwner = githubRepoOwner
		cfg.GithubRepo = githubRepo
	})
	return cfg
}

func panicIfMissingEnv(val string, key string) {
	if val == "" {
		panic(fmt.Sprintf("Missing env %s", key))
	}
}

func main() {
	cfg := GetConfig()

	gh, err := github.NewClient(github.WithAuthToken(cfg.GithubToken))
	if err != nil {
		logger.Error("error authenticating github", "err", err.Error())
		os.Exit(1)
	}

	hookName := "vetopr"
	contentType := "json"
	insecureSSL := "0"
	ghHook := &github.Hook{
		Name:   &hookName,
		Events: []string{"pull_request"},
		Config: &github.HookConfig{
			ContentType: &contentType,
			InsecureSSL: &insecureSSL,
			URL:         &cfg.WebhookURL,
			Secret:      &cfg.GithubWebhookSecret,
		},
	}

	hook, _, err := gh.Repositories.CreateHook(
		context.Background(),
		cfg.GithubRepoOwner,
		cfg.GithubRepo,
		ghHook,
	)

	if err != nil {
		if e, ok := errors.AsType[*github.ErrorResponse](err); ok {
			firstErr := e.Errors[0]
			if firstErr.Message == "Hook already exists on this repository" {
				logger.Info("Hook already created")
			} else {
				logger.Error("creating hook", "err", err)
				os.Exit(1)
			}
		} else {
			logger.Error("creating hook", "err", err)
			os.Exit(1)
		}
	} else {
		logger.Info("Hook created", "hook", hook.ID)
	}

	ghEventMonitory := NewGithubEventMonitor(cfg.GithubWebhookSecret, gh)

	logger.Info("starting server", "port", cfg.Port)
	err = http.ListenAndServe(":"+cfg.Port, ghEventMonitory)
	if err != nil {
		logger.Error("error starting server", "listenAndServe", err)
	}
}
