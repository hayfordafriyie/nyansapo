# Mini LLM

A small Go knowledge assistant that loads EDSPiKE information from JSON,
tokenizes questions, and answers with direct knowledge lookup or simple
vector-similarity search.

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
