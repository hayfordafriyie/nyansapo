# Nyansapo

Nyansapo is a small Go knowledge assistant that loads subject knowledge from files,
tokenizes questions, and answers with direct knowledge lookup or vector-
similarity search. After training, the CLI and API use the same persisted
model.

Run it from the project root:

```text
go run .
```

Ask questions such as:

```text
What is photosynthesis?
What is a prime number?
What is good citizenship?
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
curl -X POST http://localhost:8080/ask -H "Content-Type: application/json" -d "{\"question\":\"What is photosynthesis?\"}"
```

Check API health with `GET /health`:

```text
curl http://localhost:8080/health
```

Open `http://localhost:8080/` in a browser for the chat interface.

Ask one question without the interactive prompt:

```text
go run . ask What is photosynthesis?
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

The repository includes neutral subject files for testing the pipeline:
`science.txt`, `mathematics.json`, `social-studies.csv`, `english.md`, and
`statistics.json`. After adding or changing data, retrain and test questions
such as:

```text
go run . train
go run . ask-trained What is the mean?
go run . ask-trained What is a hypothesis?
go run . ask-trained What is good citizenship?
```

For automatic retraining whenever supported files are added or changed, run:

```text
go run . watch data/input
```

The watcher rebuilds `data/model.json` when it detects a file change. The API
reloads and validates that artifact for each question, so new data becomes
available without restarting the server.

API answers are cached in memory for up to 256 questions. Responses include
`X-Cache: MISS` for a newly computed answer and `X-Cache: HIT` for a cached
answer. Retraining changes the model file timestamp, automatically bypassing
old cached answers.

Ask the persisted model directly:

```text
go run . ask-trained Tell me about school management
```

Run `go run . train` before using the CLI or API. The application now uses
only documents under `data/input`; there is no legacy knowledge-file fallback.

## Add data and test training

1. Add a supported file under `data/input`:

   ```text
   data/input/biology.txt
   data/input/biology.json
   data/input/biology.csv
   data/input/biology.md
   ```

   Plain text and Markdown files become one document. JSON content is combined
   into one contextual document. CSV rows become searchable documents.

2. Put useful facts in the file. For example, `data/input/biology.txt`:

   ```text
   Photosynthesis allows plants to convert light energy into chemical energy.
   Chlorophyll helps plants absorb light.
   ```

3. Train the model from all files:

   ```text
   go run . train
   ```

   This scans `data/input` recursively and writes the trained artifact to:

   ```text
   data/model.json
   ```

4. Test the newly trained model:

   ```text
   go run . ask-trained What is photosynthesis?
   go run . ask-trained What helps plants absorb light?
   ```

   The answer should contain the matching biology text. You can also test
   interactively:

   ```text
   go run .
   ```

5. Start the API and test the same model over HTTP:

   ```text
   go run . serve
   curl -X POST http://localhost:8080/ask `
     -H "Content-Type: application/json" `
     -d "{\"question\":\"What is photosynthesis?\"}"
   ```

6. During development, automatically retrain when files are added or changed:

   ```text
   go run . watch data/input
   ```

   Keep the API running in another terminal. It reloads `data/model.json` for
   each question, so newly trained content becomes available without restarting
   the API.

To use another data directory for a one-time training run:

```text
go run . train path\to\my-data
```

To add a new format such as Parquet, register a reader in `pipeline/pipeline.go`;
the model and API layers do not need to change.

Run a single-process 200-query evaluation:

```text
go run . evaluate
```

This runs 20 varied subject questions 10 times (200 total), reporting unknown
answers, empty answers, and unique response count. It measures the current
retrieval/response system; repeated evaluation does not retrain neural weights.

With the API running in another terminal, run the same 200-query evaluation
through HTTP:

```text
go run . evaluate-api
```

## Evidence-based answers

The retrieval model first finds the most relevant trained passage. The response
layer then performs a small answer pipeline: it classifies the question as a
definition, process, or reason question, selects the best sentence, and adds
nearby supporting evidence when available. For example:

```text
What is photosynthesis?
  In simple terms, Photosynthesis allows plants to convert light energy into
  chemical energy. Chlorophyll helps plants absorb light.

How do plants make food?
  The process works like this: Photosynthesis allows plants to convert light
  energy into chemical energy. Chlorophyll helps plants absorb light.
```

Repeated questions can use different explanatory connectors, while the
supporting facts remain grounded in your documents. This is evidence-based
retrieval and composition, not yet a fully generative Transformer; adding one
later would require a language model trained for text generation.
