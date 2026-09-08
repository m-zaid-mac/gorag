# gorag

A concurrent RAG (retrieval-augmented generation) CLI written in Go, using AWS Bedrock for embeddings and generation.

Most RAG demos are a thin wrapper around a framework. This one implements the retrieval mechanism directly — chunking, embedding, and cosine-similarity search — with no vector-database dependency, and uses the ingestion step (which is I/O-bound: one network round trip per chunk) to demonstrate a bounded goroutine worker pool rather than naive one-goroutine-per-item concurrency.

## What it does

- **`gorag ingest`** — walks a directory of `.txt`/`.md` files, chunks them, and embeds every chunk concurrently through a bounded worker pool (goroutines + channels), saving the resulting vectors to a local JSON store.
- **`gorag query`** — embeds a question, retrieves the most similar chunks via cosine similarity, and asks Claude (via Bedrock) to answer using only that retrieved context, citing sources.
- **`gorag bench`** — runs the same corpus through ingestion sequentially and concurrently, and reports the measured speedup.

## Setup

Requires Go 1.22+ and an AWS account with Bedrock model access enabled for `amazon.titan-embed-text-v2:0` (embeddings) and a Claude model (generation) in your target region.

```bash
git clone https://github.com/m-zaid-mac/gorag.git
cd gorag
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/bedrockruntime
go mod tidy

export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_REGION=us-east-1   # or wherever you have Bedrock model access
```

## Usage

```bash
go build -o bin/gorag ./cmd/gorag

# Ingest the included sample docs (or point -dir at your own folder)
./bin/gorag ingest -dir sample_docs -out store.json -workers 8

# Ask a question
./bin/gorag query -store store.json -q "What is a worker pool used for?"

# Compare sequential vs. concurrent ingestion speed
./bin/gorag bench -dir sample_docs -workers 8
```

`query` defaults to `us.anthropic.claude-haiku-4-5-20251001-v1:0` for generation. AWS periodically retires older Claude snapshots and newer models are invoked via an inference-profile ID (a region prefix like `us.` or `global.` in front of the base model ID). If the default stops working, check **AWS Console → Bedrock → Model access** for what your account has enabled and override it:

```bash
./bin/gorag query -store store.json -q "..." -model us.anthropic.claude-sonnet-4-5-20250929-v1:0
```

## Testing

Chunking and vector-store logic are stdlib-only and fully unit tested without AWS credentials or network access:

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

## Design notes

- **Bounded worker pool, not one goroutine per chunk.** A fixed number of goroutines pull from a shared jobs channel and embed concurrently; `sync.WaitGroup` signals completion, closing the results channel so the receiving loop terminates cleanly. This respects Bedrock's per-account rate limits while still parallelizing the I/O wait.
- **Brute-force cosine similarity instead of a vector database.** At small-to-moderate corpus sizes this is simpler, has zero external dependencies, and keeps the entire retrieval mechanism inspectable in one file. It stops being the right choice once a corpus is large enough that linear scan is the bottleneck — see Possible extensions.
- **`context.Context`** is threaded through every Bedrock call for cancellation and future timeout support.

## Possible extensions

- Swap the JSON store for a proper index (e.g. HNSW) once corpus size stops being brute-force-friendly
- Add a `-timeout` flag using `context.WithTimeout` per embedding call
- Turn `query` into an HTTP server (`net/http`) instead of a one-shot CLI
- Replace the manual `WaitGroup` + error-logging with `errgroup` (`golang.org/x/sync/errgroup`) to propagate the first real error

## License

MIT
