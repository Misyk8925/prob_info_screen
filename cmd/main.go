package main

import (
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	first_jira "prob_info_screen/HTTPServer/handlers/jira/first-jira"
	"prob_info_screen/config"
	"prob_info_screen/storage"
)

func main() {

	config := config.MustLoadConfig()

	storage, storageErr := storage.InitDB()

	if storageErr != nil {
		panic(storageErr)
	}

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
	}

	fmt.Printf("✅ Success! Authenticated as: %s\n", user.DisplayName)

	fmt.Println("Storage initialized successfully:", storage)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)

	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/api", func(r chi.Router) {
		r.Get("/info", first_jira.New())
	})

	// Start the HTTP server
	serverAddress := config.HTTPServer.Address

	fmt.Printf("Starting server on %s\n", serverAddress)

	if err := http.ListenAndServe(serverAddress, router); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
