# Nyansapo

Nyansapo is a database-aware Go knowledge assistant. It can train a searchable
catalog from database schemas and documentation, then answer questions about
tables, columns, relationships, joins, indexes, and database operations. The
current model is retrieval-based; it does not silently mutate a database or
invent a query result.

Built and maintained by **Hayford Afriyie** — [hayfordafriyie.com](https://hayfordafriyie.com).
Nyansapo is open source under the [MIT License](LICENSE).

Run it from the project root:

```text
go run .
```

Ask questions such as:

```text
Which tables contain customer orders?
How are orders related to customers?
What columns should I use to join customers and orders?
How does a PostgreSQL window function work?
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
curl -X POST http://localhost:8080/ask -H "Content-Type: application/json" -d "{\"question\":\"How are orders related to customers?\"}"
```

Responses include the answer's retrieval confidence and whether the answer was
grounded in matching training evidence:

```json
{"answer":"Orders are related to customers through orders.customer_id and customers.id.","confidence":0.42,"grounded":true}
```

When the question has no sufficiently related evidence, Nyansapo returns
`"I do not know that yet."` instead of presenting an unrelated document as an
answer.

Check API health with `GET /health`:

```text
curl http://localhost:8080/health
```

Open `http://localhost:8080/` in a browser for the chat interface.

Ask one question without the interactive prompt:

```text
go run . ask How are orders related to customers?
```

Build a persisted retrieval model from all supported files in `data/input`:

```text
go run . train
```

The pipeline currently reads `.txt`, `.md`, `.json`, and `.csv` files
recursively. Database catalogs can be exported as JSON and trained the same
way. You can provide another directory:

```text
go run . train path/to/new-data
```

Add new reader adapters in `pipeline/` for formats such as Parquet without
changing the model or API layers.

`data/input` is the default drop folder. It may start empty; the watcher
waits until supported files are added.

The repository includes a database catalog fixture at
`data/input/database-catalog.json`. After adding or changing database metadata,
retrain and test questions such as:

```text
go run . train
go run . ask-trained How are orders related to customers?
go run . ask-trained What is a window function?
```

## Database connection configuration

Credentials are never committed to the repository. Copy `.env.example` to
`.env`, fill in only the variables needed by your provider, and keep `.env`
untracked. Existing environment variables take precedence over `.env`.

Supported provider names are `mysql`, `postgres`, `mongodb`, and `cassandra`.
The configuration accepts either `NYANSAPO_DB_URL` or separate provider fields
such as host, port, user, password, and database. `NYANSAPO_DB_READ_ONLY`
defaults to `true`.

Check the resolved configuration without printing the password:

```text
go run . db-config
```

The sample credentials sometimes published by database projects should be
treated as public demo accounts, not as application secrets. Do not paste
credentials into source files, commit messages, remotes, or chat logs. Use a
local `.env` and rotate any credential that has been exposed.

Nyansapo's database layer represents relational tables, columns, primary keys,
foreign keys, document collections, nested fields, and provider-specific
operations as catalog documents. Live introspection converts connected database
metadata into those catalog documents; it does not copy table data into the
model.

For MySQL and PostgreSQL, the live introspection command is:

```text
go run . db-introspect
go run . train
```

It writes `data/input/live-database.json` by default. MongoDB collections and
nested fields, and Cassandra partition/clustering keys, use the same command
and catalog format. Introspection only runs when read-only mode is enabled.

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
   data/input/database-catalog.json
   data/input/postgres-catalog.json
   data/input/mysql-catalog.json
   data/input/mongodb-catalog.json
   ```

   Plain text and Markdown files become one document. JSON content is combined
   into one contextual document. CSV rows become searchable documents.

2. Put useful database facts in the file. For example, `data/input/database-catalog.json`:

   ```text
   The orders table joins customers through orders.customer_id = customers.id.
   The orders table stores total_amount and created_at.
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
   go run . ask-trained How are orders related to customers?
   go run . ask-trained Which columns store order totals?
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
     -d "{\"question\":\"How are orders related to customers?\"}"
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

Run a single-process evaluation:

```text
go run . evaluate
```

This runs 20 varied database questions 10 times by default. Pass a repetition
count to choose the total; for example, this runs exactly 500 CLI queries:

```text
go run . evaluate 25
```

The report includes unknown, empty, ungrounded, and unique response counts. It
measures the current retrieval/response system; repeated evaluation does not
retrain model weights.

With the API running in another terminal, run the same evaluation through HTTP:

```text
go run . evaluate-api 25
```

## Evidence-based answers

The retrieval model first finds the most relevant trained passage. The response
layer then performs a small answer pipeline: it classifies the question as a
definition, process, or reason question, selects the best sentence, and adds
nearby supporting evidence when available. For example:

```text
How are orders related to customers?
  Orders join customers through orders.customer_id = customers.id.

How can I inspect a slow query?
  Use a read-only EXPLAIN query to inspect the database query plan.
```

Repeated questions can use different explanatory connectors, while the
supporting facts remain grounded in your documents. This is evidence-based
retrieval and composition. The trained model now verifies that the selected
evidence overlaps the question and supports the answer before marking it
grounded. This is not yet a fully generative Transformer; adding one later
would require a language model trained for text generation.
