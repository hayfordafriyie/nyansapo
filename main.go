package main

import (
	"bufio"
	"fmt"
	"os"

	"mini-llm/model"
)

func main() {
	knowledge, err := model.LoadKnowledge("data/knowledge.json")
	if err != nil {
		panic(err)
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

		fmt.Println(knowledge.Answer(question))
	}
}
