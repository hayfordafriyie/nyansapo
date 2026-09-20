package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Knowledge struct {
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

	var knowledge Knowledge
	if err := json.Unmarshal(data, &knowledge); err != nil {
		panic(err)
	}

	fmt.Println("Ask about EDSPiKE (type \"exit\" to quit).")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		question := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if question == "exit" {
			break
		}

		switch {
		case strings.Contains(question, "what is") && strings.Contains(question, "edspike"):
			fmt.Println(knowledge.Description)
		case strings.Contains(question, "country"):
			fmt.Println(knowledge.Country)
		case strings.Contains(question, "feature"):
			fmt.Println(strings.Join(knowledge.Features, ", "))
		default:
			fmt.Println("I do not know that yet.")
		}
	}
}
