// Package selflearn periodically re-introspects the connected database and
// retrains the in-memory model so answers stay current as the schema and data
// evolve, without a restart.
package selflearn

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nyansapo/database"
	"nyansapo/model"
	"nyansapo/pipeline"
)

// Trainer owns a trained model plus an optional live data assistant. Every
// interval it refreshes the catalog from the configured database, retrains
// from the training source, and atomically swaps the in-memory model.
type Trainer struct {
	source    string
	modelPath string
	livePath  string
	interval  time.Duration
	assistant *database.DataAssistant
	mu        sync.RWMutex
	trained   model.TrainedModel
	cancel    context.CancelFunc
	lastSig   string
}

// Options configures the self-learning trainer.
type Options struct {
	// Source is the directory of training documents.
	Source string
	// ModelPath is where the retrained model is persisted.
	ModelPath string
	// LivePath is where the regenerated live catalog is written.
	LivePath string
	// Interval between refresh cycles.
	Interval time.Duration
	// Assistant supplies the live database connection and catalog.
	Assistant *database.DataAssistant
}

// New returns a trainer seeded with the provided initial model.
func New(options Options, initial model.TrainedModel) *Trainer {
	if options.Source == "" {
		options.Source = "data/input"
	}
	if options.ModelPath == "" {
		options.ModelPath = "data/model.json"
	}
	if options.LivePath == "" {
		options.LivePath = "data/input/live-database.json"
	}
	if options.Interval <= 0 {
		options.Interval = 5 * time.Second
	}
	return &Trainer{
		source:    options.Source,
		modelPath: options.ModelPath,
		livePath:  options.LivePath,
		interval:  options.Interval,
		assistant: options.Assistant,
		trained:   initial,
	}
}

// Start runs the refresh loop in a background goroutine until Stop is called.
func (t *Trainer) Start(ctx context.Context) {
	if t.cancel != nil {
		return
	}
	loopCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()
		for {
			select {
			case <-loopCtx.Done():
				return
			case <-ticker.C:
				if err := t.Refresh(); err != nil {
					log.Printf("selflearn: refresh failed: %v", err)
				}
			}
		}
	}()
}

// Stop halts the background refresh loop.
func (t *Trainer) Stop(ctx context.Context) {
	if t == nil || t.cancel == nil {
		return
	}
	t.cancel()
}

// Model returns a snapshot of the current trained model.
func (t *Trainer) Model() model.TrainedModel {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.trained
}

// Answer forwards to the current in-memory model.
func (t *Trainer) Answer(question string) string {
	return t.Model().Answer(question)
}

// Teach adds a fact to the current model and persists it.
func (t *Trainer) Teach(question, answer string) error {
	t.mu.Lock()
	t.trained.Teach(question, answer)
	trained := t.trained
	t.mu.Unlock()
	return trained.Save(t.modelPath)
}

// Refresh re-introspects the database when available, regenerates the live
// catalog, retrains from the training source when the catalog changed, and
// swaps the in-memory model.
func (t *Trainer) Refresh() error {
	catalog := database.Catalog{}
	if t.assistant != nil {
		if err := t.assistant.RefreshCatalog(context.Background()); err != nil {
			return fmt.Errorf("refresh live catalog: %w", err)
		}
		catalog = t.assistant.Catalog()
		if err := database.SaveCatalog(t.livePath, catalog); err != nil {
			return fmt.Errorf("save live catalog: %w", err)
		}
	}

	signature := catalogSignature(catalog)
	t.mu.Lock()
	if signature != "" && signature == t.lastSig {
		t.mu.Unlock()
		return nil
	}
	t.mu.Unlock()

	documents, err := pipeline.New().Load(t.source)
	if err != nil {
		return fmt.Errorf("load training source: %w", err)
	}

	texts := make([]string, 0, len(documents))
	for _, document := range documents {
		if strings.HasPrefix(strings.ToLower(filepath.Base(document.Source)), "live-") {
			continue
		}
		if strings.TrimSpace(document.Text) != "" {
			texts = append(texts, document.Text)
		}
	}
	if catalog.Provider != "" {
		texts = append(texts, catalog.Documents()...)
	}
	if len(texts) == 0 {
		return fmt.Errorf("no documentation available to retrain")
	}

	fresh := model.TrainTexts(texts)
	t.mu.RLock()
	fresh.Facts = t.trained.Facts
	t.mu.RUnlock()

	if err := fresh.Save(t.modelPath); err != nil {
		return fmt.Errorf("save retrained model: %w", err)
	}

	t.mu.Lock()
	t.trained = fresh
	t.lastSig = signature
	t.mu.Unlock()
	log.Printf("selflearn: retrained %d documents on %s", len(fresh.Candidates), time.Now().Format("15:04:05"))
	return nil
}

func catalogSignature(catalog database.Catalog) string {
	var builder strings.Builder
	builder.WriteString(string(catalog.Provider))
	builder.WriteString("|")
	builder.WriteString(catalog.Database)
	builder.WriteString("|")
	for _, table := range catalog.Tables {
		builder.WriteString(table.Schema)
		builder.WriteString(".")
		builder.WriteString(table.Name)
		builder.WriteString("(")
		for _, column := range table.Columns {
			builder.WriteString(column.Name)
			builder.WriteString(":")
			builder.WriteString(column.Type)
			builder.WriteString(",")
		}
		builder.WriteString(")")
		builder.WriteString(";")
	}
	for _, collection := range catalog.Collections {
		builder.WriteString(collection.Name)
		builder.WriteString(";")
	}
	return builder.String()
}

// HasAssistant reports whether a live database is connected.
func (t *Trainer) HasAssistant() bool {
	return t != nil && t.assistant != nil
}

// ErrNoAssistant is returned when asking the live assistant is impossible.
var ErrNoAssistant = errors.New("no connected database assistant")