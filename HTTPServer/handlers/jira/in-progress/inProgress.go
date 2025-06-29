package in_progress

import (
	"encoding/json"
	"fmt"
	"github.com/andygrunwald/go-jira"
	"log"
	"net/http"
	"prob_info_screen/config"
	"time"
)

type JiraIssue struct {
	Key       string    `json:"key"`
	Summary   string    `json:"summary"`
	Status    string    `json:"status"`
	Assignee  string    `json:"assignee"`
	UpdatedAt time.Time `json:"updatedat"`
}

// New erstellt einen neuen HTTP-Handler für In-Progress-Issues
func New() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		jiraClient, err := jiraConnect(config.MustLoadConfig())

		if err != nil {
			log.Fatal("Failed to connect to Jira: ", err)
		}
		jql := "project = 'TKP' AND status = 'IN PROGRESS' "

		issues, resp, err := jiraClient.Issue.Search(jql, nil)

		if err != nil {
			log.Fatal("Failed to search issues: ", err)

		}

		var jiraResIssues []JiraIssue
		for _, issue := range issues {
			var assigneeName string
			if issue.Fields.Assignee == nil {
				assigneeName = "Unassigned"
			} else {
				assigneeName = issue.Fields.Assignee.DisplayName
			}

			jiraIssue := JiraIssue{
				Key:       issue.Key,
				Summary:   issue.Fields.Summary,
				Status:    issue.Fields.Status.Name,
				Assignee:  assigneeName,
				UpdatedAt: time.Time(issue.Fields.Updated),
			}
			jiraResIssues = append(jiraResIssues, jiraIssue)
		}

		json.NewEncoder(w).Encode(jiraResIssues)
		fmt.Printf("Found %d issues (HTTP %d)\n", len(issues), resp.StatusCode)

		return
	}
}

func jiraConnect(config *config.Config) (jira.Client, error) {
	tp := jira.BasicAuthTransport{
		Username: config.JiraConfig.Username,
		Password: config.JiraConfig.Password,
	}

	client, err := jira.NewClient(tp.Client(), config.JiraConfig.JiraURL)
	if err != nil {
		log.Fatal("Failed to create Jira client: ", err)
	}

	// Test authentication
	user, _, err := client.User.GetSelf()
	if err != nil {
		fmt.Println("❌ Authentication failed!")
		log.Fatal("Failed to authenticate: ", err)
		return jira.Client{}, err
	}

	fmt.Printf("✅ Success! Authenticated as: %s\n", user.DisplayName)
	return *client, nil
}
