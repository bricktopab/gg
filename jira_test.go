//go:build integration

package main_test

import (
	"testing"

	"github.com/bricktopab/gg/cfg"
)

func TestJiraWrapper_TryIssueCreate(t *testing.T) {
	config, _ := cfg.LoadOrCreateConfig(func() *cfg.Config {
		return nil
	})
	client, _ := NewJiraWrapper(config.JiraUser, config.JiraToken, config.JiraURL, config.JiraProject)

	types := client.GetIssueTypes()
	client.CreateIssue(types["Task"], "Test issue", "This is a test issue")
}
