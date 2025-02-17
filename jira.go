package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bricktopab/gg/cfg"
	jira "github.com/ctreminiom/go-atlassian/jira/v3"
	"github.com/ctreminiom/go-atlassian/pkg/infra/models"
)

type JiraWrapper struct {
	client *jira.Client
	config *cfg.JiraConfig
}

func NewJiraWrapperWithOldConfig(jiraUser, jiraToken, jiraURL, jiraProject string) (*JiraWrapper, error) {
	return NewJiraWrapper(&cfg.JiraConfig{
		JiraUser:      jiraUser,
		JiraURL:       jiraURL,
		JiraToken:     jiraToken,
		JiraProject:   jiraProject,
		JiraAccountID: nil,
	})
}

func NewJiraWrapper(jiraConfig *cfg.JiraConfig) (*JiraWrapper, error) {
	atlassian, err := jira.New(nil, jiraConfig.JiraURL)
	if err != nil {
		return nil, err
	}
	atlassian.Auth.SetBasicAuth(jiraConfig.JiraUser, jiraConfig.JiraToken)
	return &JiraWrapper{client: atlassian, config: jiraConfig}, nil
}

func (j *JiraWrapper) GetIssueTypes() map[string]string {
	project, resp, err := j.client.Project.Get(context.Background(), j.config.JiraProject, nil)
	if err != nil {
		log.Fatal("JIRA Error: ", resp.Bytes.String())
	}
	if project.IssueTypes == nil {
		log.Fatalf("No project with key: %s found", j.config.JiraProject)
	}
	types := map[string]string{}
	for _, v := range project.IssueTypes {
		if !v.Subtask {
			types[v.Name] = v.ID
		}
	}
	return types
}

func (j *JiraWrapper) LookupMyAccountID() (*string, error) {
	currentUser, resp, err := j.client.MySelf.Details(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("JIRA Error: %s", resp.Bytes.String())
	}
	return &currentUser.AccountID, nil
}

func (j *JiraWrapper) CreateIssue(typeID string, title, description string) *cfg.Task {
	if j.config.JiraAccountID == nil {
		id, err := j.LookupMyAccountID()
		if err == nil {
			j.config.JiraAccountID = id
		}
	}
	payload := &models.IssueScheme{
		Fields: &models.IssueFieldsScheme{
			Summary:   title,
			Project:   &models.ProjectScheme{Key: j.config.JiraProject},
			IssueType: &models.IssueTypeScheme{ID: typeID},
		},
	}

	if j.config.JiraAccountID != nil {
		payload.Fields.Assignee = &models.UserScheme{
			AccountID: *j.config.JiraAccountID,
		}
	}

	if description != "" {
		payload.Fields.Description = &models.CommentNodeScheme{
			Type:    "doc",
			Version: 1,
			Content: []*models.CommentNodeScheme{
				{
					Type: "paragraph",
					Content: []*models.CommentNodeScheme{
						{
							Type: "text",
							Text: description,
						},
					},
				},
			},
		}
	}

	issue, resp, err := j.client.Issue.Create(context.Background(), payload, nil)
	if err != nil {
		log.Fatal("JIRA Error: ", resp.Bytes.String())
	}

	log.Printf("Issue created: %s\n", issue.Key)
	return &cfg.Task{
		IssueID:     issue.Key,
		Title:       title,
		Description: description,
		Type:        typeID,
	}
}

func (j *JiraWrapper) FindOpenIssues(onlyMine bool) []cfg.Task {
	mine := "AND assignee is empty"
	if onlyMine {
		mine = "AND assignee = currentUser()"
	}
	jql := fmt.Sprintf(`project="%s"  %s AND 
		statusCategory in ("To Do", "In Progress") order by updated DESC`, j.config.JiraProject, mine)
	issues, _, err := j.client.Issue.Search.Get(
		context.Background(),
		jql,
		[]string{"summary", "description", "issuetype"},
		[]string{},
		0, 50, "")
	if err != nil {
		log.Fatal("Error: ", err)
	}
	tasks := []cfg.Task{}
	for _, issue := range issues.Issues {
		tasks = append(tasks, cfg.Task{
			IssueID: issue.Key,
			Title:   issue.Fields.Summary,
			Type:    issue.Fields.IssueType.Name,
		})
	}
	return tasks
}
