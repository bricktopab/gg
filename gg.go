package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bricktopab/gg/cfg"
)

type GG struct {
	Config *cfg.Config
	Gui    Gui
	Jira   Jira
	Git    Git
}

type Jira interface {
	CreateIssue(typeID string, name string, description string) *cfg.Task
	FindMyOpenIssues() []cfg.Task
	GetIssueTypes() map[string]string
}

type Git interface {
	SwitchLocalBranch(name string)
	GetBranchName() string
	CreatePR() string
	OpenPR(string)
}

func (g *GG) CreateIssue(name string, description string) {
	typeID, typeName, title, description := g.Gui.AskForIssueDetails(name,
		description, g.Jira.GetIssueTypes)

	task := g.Jira.CreateIssue(typeID, title, description)
	task.Type = typeName
	g.Config.AddTask(task)

	branchName := formatBranchName(task.IssueID, task.Title)
	g.Git.SwitchLocalBranch(branchName)
}

var stripOddNameChars = regexp.MustCompile(`[^\w\-\.~]`)
var taskIDRe = regexp.MustCompile(`([A-Z]+-\d+)`)

func formatBranchName(taskID, title string) string {
	branchName := fmt.Sprintf("%s_%s", taskID, strings.ReplaceAll(title, " ", "-"))
	branchName = stripOddNameChars.ReplaceAllString(branchName, "")
	return branchName
}

func (g *GG) PickIssue() {
	task := g.Gui.SelectTask(g.Jira.FindMyOpenIssues)
	g.Config.AddTask(task)
	branchName := formatBranchName(task.IssueID, task.Title)
	g.Git.SwitchLocalBranch(branchName)
}

func (g *GG) CreatePR() {
	branch := g.Git.GetBranchName()
	taskID := taskIDRe.FindString(branch)
	if taskID == "" {
		log.Fatal("Current branch does not contain a task ID")
	}
	task := g.Config.GetTask(taskID)
	title := g.Gui.AskForPRTitle(task)
	g.Git.OpenPR(title)

	g.Gui.ShowSummary(task.IssueID, task.Title, g.Config.JiraURL+"/browse/"+task.IssueID)
}
