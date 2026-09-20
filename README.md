# Mini LLM

A small Go knowledge assistant that loads EDSPiKE information from JSON,
tokenizes questions, and answers with direct knowledge lookup or simple
token-overlap search.

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
