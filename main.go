package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"mini-llm/chat"
)

type knowledgeFile struct {
	Name        string   `json:"name"`
	Country     string   `json:"country"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
}

func main() {
	data, err := os.ReadFile("data/knowledge.json")
	if err != nil {
		panic(err)
	}

	var file knowledgeFile
	if err := json.Unmarshal(data, &file); err != nil {
		panic(err)
	}

	knowledge := chat.Knowledge{
		Name:        file.Name,
		Country:     file.Country,
		Description: file.Description,
		Features:    file.Features,
	}

	fmt.Println("Ask about EDSPiKE (type \"exit\" to quit).")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		question := scanner.Text()
		if question == "exit" {
			break
		}

		fmt.Println(chat.Answer(question, knowledge))
	}
}
