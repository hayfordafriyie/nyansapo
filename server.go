package main

import (
	"log"
	"net/http"

	"mini-llm/api"
	"mini-llm/model"
)

func runServer() error {
	knowledge, err := model.LoadKnowledge("data/knowledge.json")
	if err != nil {
		return err
	}

	log.Println("API listening on http://localhost:8080")
	return http.ListenAndServe(":8080", api.NewServer(knowledge))
}
