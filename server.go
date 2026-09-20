package main

import (
	"fmt"
	"log"
	"net/http"

	"mini-llm/api"
)

func runServer() error {
	server, err := api.NewReloadingServer("data/model.json")
	if err != nil {
		return fmt.Errorf("load trained model; run `go run . train` first: %w", err)
	}

	log.Println("API listening on http://localhost:8080")
	return http.ListenAndServe(":8080", server)
}
