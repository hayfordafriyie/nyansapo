package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mini-llm/model"
)

func main() {
	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "serve") {
		if err := runServer(); err != nil {
			panic(err)
		}
		return
	}

	knowledge, err := model.LoadKnowledge("data/knowledge.json")
	if err != nil {
		panic(err)
	}

	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "train") {
		if err := model.Train(knowledge).Save("data/model.json"); err != nil {
			panic(err)
		}
		fmt.Println("trained model saved to data/model.json")
		return
	}

	if len(os.Args) > 2 && strings.EqualFold(os.Args[1], "ask") {
		fmt.Println(knowledge.Answer(strings.Join(os.Args[2:], " ")))
		return
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
