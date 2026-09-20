package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"nyansapo/evaluation"
	"nyansapo/model"
	"nyansapo/pipeline"
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

	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "evaluate") {
		result := evaluation.Run(trained, evaluationRepetitions(10))
		fmt.Printf("queries: %d\nunknown: %d\nempty: %d\nungrounded: %d\nunique answers: %d\n", result.Total, result.Unknown, result.Empty, result.Ungrounded, result.Unique)
		return
	}

	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "evaluate-api") {
		result, err := evaluation.RunAPI(http.DefaultClient, "http://localhost:8080/ask", evaluationRepetitions(10))
		if err != nil {
			panic(err)
		}
		fmt.Printf("API queries: %d\nAPI unknown: %d\nAPI empty: %d\nAPI ungrounded: %d\nAPI unique answers: %d\n", result.Total, result.Unknown, result.Empty, result.Ungrounded, result.Unique)
		return
	}

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

func evaluationRepetitions(defaultValue int) int {
	if len(os.Args) < 3 {
		return defaultValue
	}
	value, err := strconv.Atoi(os.Args[2])
	if err != nil || value < 1 {
		panic("evaluation repetitions must be a positive integer")
	}
	return value
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
