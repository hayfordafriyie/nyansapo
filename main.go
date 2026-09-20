package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"nyansapo/api"
	"nyansapo/conversation"
	"nyansapo/database"
	"nyansapo/evaluation"
	"nyansapo/model"
	"nyansapo/pipeline"
	"nyansapo/selflearn"
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
		if existing, err := model.LoadTrained("data/model.json"); err == nil && len(existing.Facts) > 0 {
			trained.Facts = existing.Facts
		}
		if err := trained.Save("data/model.json"); err != nil {
			panic(err)
		}
		fmt.Printf("trained %d documents from %s; model saved to data/model.json\n", len(trained.Candidates), source)
		return
	}

	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "db-config") {
		config, err := database.LoadConfig()
		if err != nil {
			panic(err)
		}
		fmt.Printf("provider: %s\ndatabase: %s\nuser: %s\nendpoint: %s\nread-only: %t\npassword configured: %t\n",
			config.Provider, config.Database, config.User, config.Safe().Endpoint, config.ReadOnly, config.Safe().HasPassword)
		return
	}

	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "db-introspect") {
		config, err := database.LoadConfig()
		if err != nil {
			panic(err)
		}
		catalog, err := database.IntrospectConfigured(context.Background(), config)
		if err != nil {
			panic(err)
		}
		output := "data/input/live-database.json"
		if len(os.Args) > 2 {
			output = os.Args[2]
		}
		if err := database.SaveCatalog(output, catalog); err != nil {
			panic(err)
		}
		fmt.Printf("introspected %d tables and %d collections; catalog saved to %s\n",
			len(catalog.Tables), len(catalog.Collections), output)
		return
	}

	trained, err := model.LoadTrained("data/model.json")
	if err != nil {
		panic(err)
	}
	dataAssistant, err := loadOptionalDataAssistant()
	if err != nil {
		panic(err)
	}
	if dataAssistant != nil {
		defer dataAssistant.Close()
	}
	selfCtx, selfCancel := context.WithCancel(context.Background())
	defer selfCancel()
	trainer := startSelfLearning(selfCtx, dataAssistant)
	answer := func(question string) string {
		if dataAssistant != nil {
			result, err := dataAssistant.Ask(context.Background(), question)
			if err == nil {
				return result.Answer
			}
			if !errors.Is(err, database.ErrUnsupportedDataQuestion) {
				return fmt.Sprintf("I could not safely query the configured database: %v", err)
			}
		}
		if database.IsSchemaQuestion(question) {
			return "Connect a read-only business database to ask questions about live records. Nyansapo is not intended to answer schema-only questions."
		}
		if trainer != nil {
			return trainer.Answer(question)
		}
		return trained.Answer(question)
	}

	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "evaluate") {
		result := evaluation.RunWith(answer, evaluationRepetitions(10))
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

	if len(os.Args) > 2 && strings.EqualFold(os.Args[1], "ask") {
		fmt.Println(answer(strings.Join(os.Args[2:], " ")))
		return
	}

	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "teach") {
		if err := teachSession(answer, trainer); err != nil {
			panic(err)
		}
		return
	}

	fmt.Println("Run a quick self-check before chatting...")
	check := evaluation.RunWith(answer, 10)
	fmt.Printf("self-check: %d queries, %d unknown, %d empty, %d ungrounded, %d unique answers\n", check.Total, check.Unknown, check.Empty, check.Ungrounded, check.Unique)

	fmt.Println("Ask a question (type \"exit\" to quit).")

	memory := conversation.NewMemory(0)
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

		if reply, ok := memory.Repeat(question); ok {
			fmt.Println(reply)
			memory.Remember(question, reply)
			continue
		}

		if reply, ok := memory.Respond(question); ok {
			fmt.Println(reply)
			memory.Remember(question, reply)
			continue
		}

		original := question
		enriched := memory.Enrich(question)
		response := answer(enriched)
		fmt.Println(response)
		memory.Remember(original, response)
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

func teachSession(answer func(string) string, trainer *selflearn.Trainer) error {
	fmt.Println("Training mode: ask questions. If the answer is wrong or missing, type the correct one at the prompt.")
	fmt.Println("Type \"test <question>\" to verify without teaching, or \"exit\" to quit.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if strings.EqualFold(input, "exit") {
			break
		}

		if len(input) > 5 && strings.EqualFold(input[:5], "test ") {
			question := strings.TrimSpace(input[5:])
			reply := answer(question)
			fmt.Println(reply)
			continue
		}

		question := input
		reply := answer(question)
		fmt.Println(reply)

		fmt.Print("Correct? (press Enter to accept, or type the correct answer): ")
		if !scanner.Scan() {
			break
		}
		correction := strings.TrimSpace(scanner.Text())
		if correction == "" {
			continue
		}
		if strings.EqualFold(correction, "exit") {
			break
		}
		if trainer != nil {
			if err := trainer.Teach(question, correction); err != nil {
				return fmt.Errorf("save trained model: %w", err)
			}
		} else {
			trained, err := model.LoadTrained("data/model.json")
			if err != nil {
				return err
			}
			trained.Teach(question, correction)
			if err := trained.Save("data/model.json"); err != nil {
				return fmt.Errorf("save trained model: %w", err)
			}
		}
		fmt.Printf("Saved! Now testing... %s\n", answer(question))
	}
	return nil
}

func trainFrom(source string) (model.TrainedModel, error) {
	if _, err := os.Stat(source); err == nil {
		documents, err := pipeline.New().Load(source)
		if err != nil {
			return model.TrainedModel{}, err
		}

		texts := make([]string, 0, len(documents))
		for _, document := range documents {
			if strings.HasPrefix(strings.ToLower(filepath.Base(document.Source)), "live-") {
				continue
			}
			texts = append(texts, document.Text)
		}
		if len(texts) == 0 {
			return model.TrainedModel{}, fmt.Errorf("training source %s contains no documentation after excluding live catalogs", source)
		}
		return model.TrainTexts(texts), nil
	} else if !os.IsNotExist(err) {
		return model.TrainedModel{}, fmt.Errorf("inspect training source: %w", err)
	}

	return model.TrainedModel{}, fmt.Errorf("training source %s does not exist", source)
}

func loadOptionalDataAssistant() (*database.DataAssistant, error) {
	config, err := database.LoadConfig()
	if err != nil {
		if os.Getenv("NYANSAPO_DB_PROVIDER") == "" {
			return nil, nil
		}
		return nil, err
	}
	return database.NewDataAssistant(context.Background(), config)
}

func startSelfLearning(ctx context.Context, assistant *database.DataAssistant) *selflearn.Trainer {
	initial, err := model.LoadTrained("data/model.json")
	if err != nil {
		return nil
	}
	trainer := selflearn.New(selflearn.Options{
		Source:    "data/input",
		ModelPath: "data/model.json",
		LivePath:  "data/input/live-database.json",
		Interval:  5 * time.Second,
		Assistant: assistant,
	}, initial)
	trainer.Start(ctx)
	return trainer
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

func runServer() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataAssistant, err := loadOptionalDataAssistant()
	if err != nil {
		return fmt.Errorf("load configured database: %w", err)
	}
	if dataAssistant != nil {
		defer dataAssistant.Close()
	}
	server, err := api.NewDataServer("data/model.json", dataAssistant)
	if err != nil {
		return fmt.Errorf("load trained model; run `go run . train` first: %w", err)
	}

	trainer := startSelfLearning(ctx, dataAssistant)
	if trainer != nil {
		defer trainer.Stop(ctx)
	}

	log.Println("API listening on http://localhost:8080")
	return http.ListenAndServe(":8080", server)
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
