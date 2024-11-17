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
	client  *jira.Client
	project string
}

func NewJiraWrapper(user, token, url, project string) (*JiraWrapper, error) {
	atlassian, err := jira.New(nil, url)
	if err != nil {
		return nil, err
	}
	atlassian.Auth.SetBasicAuth(user, token)
	return &JiraWrapper{atlassian, project}, nil
}

func (j *JiraWrapper) GetIssueTypes() map[string]string {
	project, resp, err := j.client.Project.Get(context.Background(), j.project, nil)
	if err != nil {
		log.Fatal("JIRA Error: ", resp.Bytes.String())
	}
	if project.IssueTypes == nil {
		log.Fatalf("No project with key: %s found", j.project)
	}
	types := map[string]string{}
	for _, v := range project.IssueTypes {
		if !v.Subtask {
			types[v.Name] = v.ID
		}
	}
	return types
}

func (j *JiraWrapper) CreateIssue(typeID string, title, description string) *cfg.Task {
	payload := &models.IssueScheme{
		Fields: &models.IssueFieldsScheme{
			Summary:   title,
			Project:   &models.ProjectScheme{Key: j.project},
			IssueType: &models.IssueTypeScheme{ID: typeID},
		},
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

func (j *JiraWrapper) FindMyOpenIssues() []cfg.Task {
	jql := fmt.Sprintf(`project="%s" AND assignee = currentUser() AND 
		statusCategory in ("To Do", "In Progress") order by updated DESC`, j.project)
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
