package main

import (
	"encoding/json"
	"fmt"
	"os"
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

	fmt.Println(knowledge.Name)
	fmt.Println(knowledge.Description)
}
