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

Ask one question without the interactive prompt:

```text
go run . ask What is EDSPiKE?
```

Build a persisted retrieval model from the knowledge file:

```text
go run . train
```

Ask the persisted model directly:

```text
go run . ask-trained Tell me about school management
```

If `data/model.json` is not present, the regular CLI commands use the source
knowledge directly. Run `go run . train` to enable persisted-model inference.
