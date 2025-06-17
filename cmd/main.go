package main

import (
	"fmt"
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
