package main

import (
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	first_jira "prob_info_screen/HTTPServer/handlers/jira/first-jira"
	in_progress "prob_info_screen/HTTPServer/handlers/jira/in-progress"
	"prob_info_screen/config"
	"prob_info_screen/storage"
)

func main() {

	config := config.MustLoadConfig()

	storage, storageErr := storage.InitDB()

	if storageErr != nil {
		panic(storageErr)
	}

	jiraClient, err := jiraConnect(config)

	if err != nil {
		log.Fatal("Failed to connect to Jira: ", err)
	}

	fmt.Println("Jira client initialized successfully:", jiraClient)

	fmt.Println(jiraClient.Board.GetAllBoards(nil))
	testIssue, _, _ := jiraClient.Issue.Get("TKP-2", nil)
	if testIssue != nil {
		fmt.Println("Test issue retrieved successfully:", testIssue.Fields.Summary)
	} else {
		fmt.Println("Failed to retrieve test issue.")
	}

	jql := "project = 'TKP' AND status = 'IN PROGRESS' "

	issues, resp, err := jiraClient.Issue.Search(jql, nil)

	if err != nil {
		log.Fatal("Failed to search issues: ", err)

	}
	fmt.Printf("Found %d issues (HTTP %d)\n", len(issues), resp.StatusCode)
	for _, issue := range issues {
		fmt.Printf("- %s: %s\n", issue.Key, issue.Fields.Summary)
	}

	jql2 := "created >= -1d AND project = 'TKP'"

	issues2, resp2, err := jiraClient.Issue.Search(jql2, nil)
	if err != nil {
		log.Fatal("Failed to search issues with JQL: ", err)
	}

	fmt.Printf("Found %d issues in the last 24 hours (HTTP %d)\n", len(issues2), resp2.StatusCode)
	for _, issue := range issues2 {
		fmt.Printf("- %s: %s\n", issue.Key, issue.Fields.Summary)
	}

	opts := &jira.SearchOptions{
		StartAt:       0,
		MaxResults:    5,
		Fields:        []string{"summary", "status", "assignee"},
		ValidateQuery: "strict",
	}

	issues3, resp3, err := jiraClient.Issue.Search("project = 'TKP' AND status = 'TO DO'", opts)
	if err != nil {
		log.Fatal("Failed to search issues with options: ", err)
	}

	fmt.Printf("Found %d issues with options (HTTP %d)\n", len(issues3), resp3.StatusCode)
	for _, issue := range issues3 {
		fmt.Printf("- %s: %s (Status: %s)\n", issue.Key, issue.Fields.Summary, issue.Fields.Status.Name)
	}

	fmt.Println(storage)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)

	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/api", func(r chi.Router) {
		r.Get("/info", first_jira.New())
	})

	router.Route("/api/jira", func(r chi.Router) {

		r.Get("/in_progress", in_progress.New())
	})

	// Start the HTTP server
	serverAddress := config.HTTPServer.Address

	fmt.Printf("Starting server on %s\n", serverAddress)

	if err := http.ListenAndServe(serverAddress, router); err != nil {
		log.Fatal("Failed to start server: ", err)
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
