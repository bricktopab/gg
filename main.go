package main

import (
	"log"
	"os"

	"github.com/bricktopab/gg/cfg"
	"github.com/bricktopab/gg/gui"
	"github.com/urfave/cli/v2"
)

type Cli interface {
	CreateIssue(name string, description string)
	PickIssue()
	CreatePR()
}

type Gui interface {
	AskForConfig() *cfg.Config
	AskForIssueDetails(string, string, func() map[string]string) (string, string, string, string)
	SelectTask(func(bool) []cfg.Task) *cfg.Task
	AskForPRTitle(*cfg.Task) string
	ShowSummary(string, string, string)
}

func init() {
	// Lets not log timestamps and other jibberish
	log.SetFlags(0)
}

func main() {
	var gui Gui = &gui.Gui{}
	var gg Cli

	app := &cli.App{
		Name:  "gg",
		Usage: "JIRA 🏓 GitHub",
		Before: func(cCtx *cli.Context) error {
			config, err := cfg.LoadOrCreateConfig(gui.AskForConfig)
			if err != nil {
				log.Fatalf("Failed to load or create config: %v", err)
			}
			jira, err := NewJiraWrapperWithOldConfig(config.JiraUser, config.JiraToken, config.JiraURL, config.JiraProject)
			if err != nil {
				log.Fatalf("Failed to create Jira client: %v", err)
			}
			gg = &GG{
				Config: config,
				Jira:   jira,
				Gui:    gui,
				Git:    &ExternalGit{},
			}

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:      "new",
				Aliases:   []string{"n"},
				Args:      true,
				Usage:     "Create issue and local branch interactively",
				ArgsUsage: "[issue title] [issue description]",
				Action: func(cCtx *cli.Context) error {
					gg.CreateIssue(cCtx.Args().First(), cCtx.Args().Get(1))
					return nil
				},
			},
			{
				Name:    "issue",
				Aliases: []string{"i"},
				Usage:   "Looks up one of your issues and creates/switches a local branch",
				Action: func(cCtx *cli.Context) error {
					gg.PickIssue()
					return nil
				},
			},
			{
				Name:    "pull",
				Aliases: []string{"pr"},
				Usage:   "Creates a PR with naming that matches your ticket",
				Action: func(cCtx *cli.Context) error {
					gg.CreatePR()
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
