package gui

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bricktopab/gg/cfg"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

type Gui struct {
}

func (g *Gui) AskForConfig() *cfg.Config {
	var cfg cfg.Config = cfg.NewConfg()
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().Title("Welcome to gg!").
				Description("This seems to be your first run. Please provde basic config which we'll hide in ~/.gg\n\n").
				Next(true).
				NextLabel("Continue"),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Jira Username").
				Placeholder("someone@example.com").
				Validate(validateString("username cannot be empty")).
				Value(&cfg.JiraUser),
			huh.NewInput().
				Title("JIRA API Token").
				Description("Generate one at https://id.atlassian.com/manage-profile/security/api-tokens").
				Placeholder("..hamana...").
				EchoMode(huh.EchoModePassword).
				Validate(validateString("API Token cannot be empty")).
				Value(&cfg.JiraToken),
			huh.NewInput().
				Title("Jira URL").
				Placeholder("https://example.atlassian.net").
				Validate(validateString("URL cannot be empty")).
				Value(&cfg.JiraURL),
			huh.NewInput().
				Title("JIRA projct key").
				Placeholder("PROJ").
				Validate(validateString("project key cannot be empty")).
				Value(&cfg.JiraProject),
		),
	)
	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}

	return &cfg
}

func validateString(message string) func(s string) error {
	return func(s string) error {
		if s == "" {
			return errors.New(message)
		}
		return nil
	}
}

func (g *Gui) AskForIssueDetails(preTitle, preDesc string, typesFn func() map[string]string) (string, string, string, string) {
	typeOptions := []huh.Option[string]{}
	var types map[string]string
	_ = spinner.New().Title("Asking JIRA for issue types...").Action(
		func() {
			types = typesFn()
			for k, v := range types {
				typeOptions = append(typeOptions, huh.NewOption(k, v))
			}
		},
	).Run()
	var issueTypeID string
	title := preTitle
	description := preDesc
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Provide issue details"),
			huh.NewInput().
				Title("Issue title").
				Inline(true).
				Placeholder(title).
				Value(&title).
				Validate(validateString("Title cannot be empty")),
			huh.NewInput().
				Title("Issue description").
				Inline(true).
				Placeholder(description).
				Value(&description),
			huh.NewSelect[string]().
				Title("Issue type").
				Options(typeOptions...).
				Value(&issueTypeID),
		),
	)
	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}

	return issueTypeID, types[issueTypeID], title, description
}

func (g *Gui) SelectTask(fetch func(bool) []cfg.Task) *cfg.Task {
	var task cfg.Task

	reloadDummyTask := cfg.Task{
		Type: "dummy-all",
	}

	loadIssuesWithSpinner := func(mine bool) []huh.Option[cfg.Task] {
		var options []huh.Option[cfg.Task]
		_ = spinner.New().Title("Asking JIRA for issues...").Action(
			func() {
				tasks := fetch(mine)
				for _, t := range tasks {
					options = append(options, huh.NewOption(t.IssueID+" "+t.Title, t))
				}
				assigned := "unassigned"
				if !mine {
					assigned = "my issues"
				}
				options = append(options,

					huh.NewOption(
						fmt.Sprintf("...fetch %s", assigned),
						reloadDummyTask),
				)
			},
		).Run()
		return options
	}

	toggleMine := true
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[cfg.Task]().
				Title("Choose a task to start working on").
				Filtering(true).
				OptionsFunc(func() []huh.Option[cfg.Task] { return loadIssuesWithSpinner(toggleMine) }, &toggleMine).
				// Options(options...).
				Value(&task).Validate(func(t cfg.Task) error {
				// hackisk way to trigger reload
				if t.Type == "dummy-all" {
					toggleMine = !toggleMine
					return errors.New("loading more..")
				}
				return nil
			}),
		),
	)
	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}
	return &task
}

func (g *Gui) AskForPRTitle(task *cfg.Task) string {
	if task == nil {
		log.Fatal("task cannot be nil")
	}

	prTitle := fmt.Sprintf("%s: %s", task.IssueID, task.Title)
	prefix := "-"
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Create PR").Description("Select the style of PR you want."),
			huh.NewSelect[string]().
				Title("PR title prefix").
				Height(6).
				Options(huh.NewOptions("-", "feat", "fix", "chore")...).
				Value(&prefix),
			huh.NewNote().TitleFunc(func() string {
				switch prefix {
				case "chore":
					prTitle = fmt.Sprintf("chore(%s): %s", task.IssueID, task.Title)
				case "feat":
					prTitle = fmt.Sprintf("feat(%s): %s", task.IssueID, task.Title)
				case "fix":
					prTitle = fmt.Sprintf("fix(%s): %s", task.IssueID, task.Title)
				default:
					prTitle = fmt.Sprintf("%s: %s", task.IssueID, task.Title)
				}
				return prTitle
			}, &prefix),
		),
	)

	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}
	return prTitle
}

func (g *Gui) ShowSummary(issueID, title, url string) {
	keyValue := func(k, v string) string {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("grey")).Render(k) +
			":\t" +
			lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render(v)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb,
		`%s

%s
%s
%s`,
		lipgloss.NewStyle().Bold(true).Render("PR Summary"),
		keyValue("Issue", issueID),
		keyValue("Title", title),
		lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Underline(true).Render(url),
	)

	log.Println(
		lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			Render(sb.String()),
	)
}
