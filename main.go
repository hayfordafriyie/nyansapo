package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mini-llm/model"
	"mini-llm/pipeline"
)

func main() {
	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "serve") {
		if err := runServer(); err != nil {
			panic(err)
		}
		return
	}

	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "train") {
		source := "data/input"
		if len(os.Args) > 2 {
			source = os.Args[2]
		}
		knowledge, err := model.LoadKnowledge("data/knowledge.json")
		if err != nil && os.IsNotExist(err) {
			knowledge = model.Knowledge{}
		} else if err != nil {
			panic(err)
		}
		trained, err := trainFrom(source, knowledge)
		if err != nil {
			panic(err)
		}
		if err := trained.Save("data/model.json"); err != nil {
			panic(err)
		}
		fmt.Printf("trained %d documents from %s; model saved to data/model.json\n", len(trained.Candidates), source)
		return
	}

	knowledge, err := model.LoadKnowledge("data/knowledge.json")
	if err != nil {
		panic(err)
	}

	answer := knowledge.Answer
	if trained, err := model.LoadTrained("data/model.json"); err == nil {
		answer = trained.Answer
	} else if !os.IsNotExist(err) {
		panic(err)
	}

	if len(os.Args) > 2 && strings.EqualFold(os.Args[1], "ask-trained") {
		trained, err := model.LoadTrained("data/model.json")
		if err != nil {
			panic(err)
		}
		fmt.Println(trained.Answer(strings.Join(os.Args[2:], " ")))
		return
	}

	if len(os.Args) > 2 && strings.EqualFold(os.Args[1], "ask") {
		fmt.Println(answer(strings.Join(os.Args[2:], " ")))
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

		fmt.Println(answer(question))
	}
}

func trainFrom(source string, knowledge model.Knowledge) (model.TrainedModel, error) {
	if _, err := os.Stat(source); err == nil {
		documents, err := pipeline.New().Load(source)
		if err != nil {
			return model.TrainedModel{}, err
		}
		texts := make([]string, 0, len(documents))
		for _, document := range documents {
			texts = append(texts, document.Text)
		}
		return model.TrainTexts(texts), nil
	} else if !os.IsNotExist(err) {
		return model.TrainedModel{}, fmt.Errorf("inspect training source: %w", err)
	}

	legacyPath := filepath.Clean("data/knowledge.json")
	if _, err := os.Stat(legacyPath); err != nil {
		return model.TrainedModel{}, fmt.Errorf("training source %s does not exist: %w", source, err)
	}
	return model.Train(knowledge), nil
}
