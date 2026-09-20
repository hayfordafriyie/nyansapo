package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "watch") {
		source := "data/input"
		if len(os.Args) > 2 {
			source = os.Args[2]
		}
		if err := watchTraining(source); err != nil {
			panic(err)
		}
		return
	}

	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "train") {
		source := "data/input"
		if len(os.Args) > 2 {
			source = os.Args[2]
		}
		trained, err := trainFrom(source)
		if err != nil {
			panic(err)
		}
		if err := trained.Save("data/model.json"); err != nil {
			panic(err)
		}
		fmt.Printf("trained %d documents from %s; model saved to data/model.json\n", len(trained.Candidates), source)
		return
	}

	trained, err := model.LoadTrained("data/model.json")
	if err != nil {
		panic(err)
	}
	answer := trained.Answer

	if len(os.Args) > 2 && strings.EqualFold(os.Args[1], "ask-trained") {
		fmt.Println(trained.Answer(strings.Join(os.Args[2:], " ")))
		return
	}

	if len(os.Args) > 2 && strings.EqualFold(os.Args[1], "ask") {
		fmt.Println(answer(strings.Join(os.Args[2:], " ")))
		return
	}

	fmt.Println("Ask a question (type \"exit\" to quit).")

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

func trainFrom(source string) (model.TrainedModel, error) {
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

	return model.TrainedModel{}, fmt.Errorf("training source %s does not exist", source)
}

func watchTraining(source string) error {
	var previous string
	for {
		fingerprint, err := sourceFingerprint(source)
		if err != nil {
			return err
		}
		if fingerprint != previous {
			if fingerprint == "" {
				time.Sleep(5 * time.Second)
				continue
			}
			trained, err := trainFrom(source)
			if err != nil {
				return err
			}
			if err := trained.Save("data/model.json"); err != nil {
				return err
			}
			fmt.Printf("trained %d documents from %s\n", len(trained.Candidates), source)
			previous = fingerprint
		}
		time.Sleep(5 * time.Second)
	}
}

func sourceFingerprint(root string) (string, error) {
	var parts []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".txt", ".md", ".markdown", ".json", ".csv":
			parts = append(parts, fmt.Sprintf("%s:%d:%d", path, info.Size(), info.ModTime().UnixNano()))
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("fingerprint training source: %w", err)
	}
	return strings.Join(parts, "|"), nil
}
