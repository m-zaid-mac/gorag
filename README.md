# gorag — a concurrent RAG CLI in Go

A small command-line RAG (retrieval-augmented generation) tool written
in Go, using AWS Bedrock for embeddings and generation. Built as a
focused weekend project to get real Go under your fingers — specifically
goroutines, channels, and worker pools — applied to something you
already know deeply (RAG pipelines), so the *only* new variable is the
language.

Why this shape, specifically: nearly every Python RAG project on a
resume is "I called LangChain." This one shows you understand what's
underneath it (chunking, cosine similarity, the embed-then-retrieve
flow — no vector-DB dependency to hide it) *and* shows a Go-specific
skill most candidates can't demonstrate: bounded concurrency for an
I/O-bound workload.

## What it does

- `gorag ingest` — walks a directory of `.txt`/`.md` files, chunks them,
  and embeds every chunk **concurrently** using a bounded worker pool
  (goroutines + channels), then saves the vectors to a local JSON store.
- `gorag query` — embeds your question, finds the most similar chunks
  via cosine similarity, and asks Claude (through Bedrock) to answer
  using only that retrieved context, citing sources.
- `gorag bench` — runs the same corpus through ingestion sequentially
  and concurrently, and prints the measured speedup. **Run this on your
  own corpus and use the real number** — don't guess at one.

## A note on Bedrock model IDs

AWS retires Claude model snapshots on Bedrock periodically, and newer
models are invoked through an **inference profile ID** (a region prefix
like `us.` or `global.` in front of the base model ID) rather than the
bare model ID. `gorag query` defaults to
`us.anthropic.claude-haiku-4-5-20251001-v1:0`, a current, low-cost,
non-deprecated model as of writing — but if AWS has moved on by the time
you run this, override it:

```bash
./bin/gorag query -store store.json -q "..." -model us.anthropic.claude-sonnet-4-5-20250929-v1:0
```

Check **AWS Console → Bedrock → Model access** for exactly which models
your account currently has enabled, and their exact IDs.

## Setup

You'll need Go 1.22+ and an AWS account with Bedrock model access
enabled for `amazon.titan-embed-text-v2:0` and
`anthropic.claude-3-5-sonnet-20240620-v1:0` in your target region.

```bash
cd gorag
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/bedrockruntime
go mod tidy

export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_REGION=us-east-1   # or wherever you have Bedrock model access
```

`go get`/`go mod tidy` need network access, which is why they aren't
pre-run here — run them once on your own machine and they'll pin exact
versions into `go.mod`/`go.sum`.

## Run it

```bash
# Build
go build -o bin/gorag ./cmd/gorag

# Ingest the included sample docs (or point -dir at your own folder)
./bin/gorag ingest -dir sample_docs -out store.json -workers 8

# Ask a question
./bin/gorag query -store store.json -q "What is a worker pool used for?"

# Get your real concurrency-speedup number for a resume bullet
./bin/gorag bench -dir sample_docs -workers 8
```

## Run the tests

The chunking and vector-store logic is stdlib-only, so it's fully unit
tested without needing AWS credentials or network access:

```bash
go test ./internal/chunk/... ./internal/store/... -v
```

## Project layout

```
gorag/
  cmd/gorag/main.go        CLI entrypoint (ingest / query / bench)
  internal/chunk/          word-window chunking + tests
  internal/store/          JSON vector store + cosine similarity + tests
  internal/bedrock/        thin AWS Bedrock client (Titan embed, Claude generate)
  internal/ingest/         worker-pool orchestration + benchmark
  internal/query/          retrieval + answer generation
  sample_docs/             two short docs to try it on immediately
```

## Go concepts this project actually exercises

- Goroutines + a **bounded** worker pool (not one goroutine per item —
  that would blow past Bedrock's rate limits on a real corpus)
- Channels as both a job queue and a fan-in for results
- `sync.WaitGroup` to know when all workers are done, then closing the
  results channel so the receiving `for range` loop terminates cleanly
- `context.Context` threaded through every Bedrock call
- Table-style unit tests (`go test`) on the pieces that don't need
  external services

## Extending it (if you want more than a weekend)

- Swap the JSON store for a real index (e.g. HNSW) once corpus size
  stops being brute-force-friendly
- Add a `-timeout` flag using `context.WithTimeout` per embedding call
- Turn `query` into a small HTTP server (`net/http`) so it's a service,
  not just a CLI
- Add `errgroup` (`golang.org/x/sync/errgroup`) instead of the manual
  `WaitGroup` + error-logging, and propagate the first real error

## Using this for your job search

Once you've run `bench` on a real corpus (even just a folder of PDFs-as-text
or your own project READMEs), you have an honest, specific resume line —
something like:

> Built a concurrent RAG CLI in Go using a bounded goroutine worker pool
> for document embedding via AWS Bedrock, achieving a **[your measured
> X]x** speedup over sequential ingestion on a [N]-document corpus

Fill in the bracketed numbers with what `bench` actually prints for
you — don't reuse a number from someone else's run, and don't round up.
It'll also give you a concrete interview answer to "have you used Go?"
that isn't "I did a tutorial": you can walk through *why* a bounded pool
beats one-goroutine-per-chunk (rate limits, memory), and why brute-force
cosine similarity is a legitimate choice at small scale rather than a
premature-optimization shortcut.
