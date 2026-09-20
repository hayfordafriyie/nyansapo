# Mini LLM

A small Go knowledge assistant that loads EDSPiKE information from JSON,
tokenizes questions, and answers with direct knowledge lookup or vector-
similarity search. After training, the CLI and API use the same persisted
model.

Run it from the project root:

```text
go run .
```

Ask questions such as:

```text
What is EDSPiKE?
What country?
Tell me about learning
exit
```

Start the JSON API:

```text
go run . serve
```

The API loads `data/model.json`, so run `go run . train` once before starting
the server. Invalid or empty model artifacts are rejected at startup.

Send a question with `POST /ask`:

```text
curl -X POST http://localhost:8080/ask -H "Content-Type: application/json" -d "{\"question\":\"What is EDSPiKE?\"}"
```

Check API health with `GET /health`:

```text
curl http://localhost:8080/health
```

Open `http://localhost:8080/` in a browser for the chat interface.

Ask one question without the interactive prompt:

```text
go run . ask What is EDSPiKE?
```

Build a persisted retrieval model from all supported files in `data/input`:

```text
go run . train
```

The pipeline currently reads `.txt`, `.md`, `.json`, and `.csv` files
recursively. You can provide another directory:

```text
go run . train path/to/new-data
```

Add new reader adapters in `pipeline/` for formats such as Parquet without
changing the model or API layers.

`data/input` is the default drop folder. It may start empty; the watcher
waits until supported files are added.

The repository includes small sample files for testing the pipeline:
`science.txt`, `statistics.json`, `experiments.csv`, `edspike-overview.md`,
`edspike-facts.json`, and `edspike-features.csv`. After adding or changing
data, retrain and test questions such as:

```text
go run . train
go run . ask-trained What is the median?
go run . ask-trained What is a hypothesis?
```

For automatic retraining whenever supported files are added or changed, run:

```text
go run . watch data/input
```

The watcher rebuilds `data/model.json` when it detects a file change. The API
reloads and validates that artifact for each question, so new data becomes
available without restarting the server.

Ask the persisted model directly:

```text
go run . ask-trained Tell me about school management
```

If `data/model.json` is not present, the regular CLI commands use the source
knowledge directly. Run `go run . train` to enable persisted-model inference.
