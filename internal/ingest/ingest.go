// Package ingest turns a directory of text files into a populated
// vector store. The interesting part isn't the embedding call itself —
// it's that embedding is I/O-bound (network round trip to Bedrock per
// chunk), which is exactly the case Go's goroutines were built for.
// A bounded worker pool lets many embedding calls be in flight at once
// without spawning one goroutine per chunk (which would blow past
// Bedrock's per-account rate limits on a large corpus).
package ingest

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gorag/internal/bedrock"
	"gorag/internal/chunk"
	"gorag/internal/store"
)

// Run implements `gorag ingest`.
func Run(args []string) {
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	dir := fs.String("dir", "", "directory of .txt/.md files to ingest")
	out := fs.String("out", "store.json", "path to write the vector store")
	workers := fs.Int("workers", 8, "number of concurrent embedding workers")
	region := fs.String("region", "us-east-1", "AWS region for Bedrock")
	fs.Parse(args)

	if *dir == "" {
		log.Fatal("ingest: -dir is required")
	}

	ctx := context.Background()
	client, err := bedrock.NewClient(ctx, *region)
	if err != nil {
		log.Fatalf("connecting to Bedrock: %v", err)
	}

	chunks, err := loadChunks(*dir)
	if err != nil {
		log.Fatalf("loading documents: %v", err)
	}
	fmt.Printf("loaded %d chunks from %s\n", len(chunks), *dir)

	start := time.Now()
	entries := embedConcurrently(ctx, client, chunks, *workers)
	elapsed := time.Since(start)
	fmt.Printf("embedded %d/%d chunks in %s using %d workers\n", len(entries), len(chunks), elapsed, *workers)

	s := &store.Store{Entries: entries}
	if err := s.Save(*out); err != nil {
		log.Fatalf("saving store: %v", err)
	}
	fmt.Printf("saved store to %s\n", *out)
}

// RunBench implements `gorag bench`: it embeds the same corpus with a
// single worker and again with -workers workers, and prints the
// measured speedup. That number is the honest resume metric — run it
// on your own corpus rather than reusing a number from someone else's.
func RunBench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	dir := fs.String("dir", "", "directory of .txt/.md files to benchmark")
	workers := fs.Int("workers", 8, "number of concurrent embedding workers")
	region := fs.String("region", "us-east-1", "AWS region for Bedrock")
	fs.Parse(args)

	if *dir == "" {
		log.Fatal("bench: -dir is required")
	}

	ctx := context.Background()
	client, err := bedrock.NewClient(ctx, *region)
	if err != nil {
		log.Fatalf("connecting to Bedrock: %v", err)
	}

	chunks, err := loadChunks(*dir)
	if err != nil {
		log.Fatalf("loading documents: %v", err)
	}
	fmt.Printf("benchmarking %d chunks\n\n", len(chunks))

	seqStart := time.Now()
	embedConcurrently(ctx, client, chunks, 1)
	seqElapsed := time.Since(seqStart)
	fmt.Printf("sequential (1 worker):   %s\n", seqElapsed)

	concStart := time.Now()
	embedConcurrently(ctx, client, chunks, *workers)
	concElapsed := time.Since(concStart)
	fmt.Printf("concurrent (%d workers): %s\n", *workers, concElapsed)

	if concElapsed > 0 {
		speedup := float64(seqElapsed) / float64(concElapsed)
		fmt.Printf("\nspeedup: %.2fx\n", speedup)
	}
}

func loadChunks(dir string) ([]chunk.Chunk, error) {
	var all []chunk.Chunk
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := filepath.Ext(path)
		if ext != ".txt" && ext != ".md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		all = append(all, chunk.SplitWords(filepath.Base(path), string(data), 200, 40)...)
		return nil
	})
	return all, err
}

// embedConcurrently runs a bounded worker pool over chunks, calling the
// Bedrock embeddings API for each. `workers` goroutines pull from a
// shared jobs channel; results stream back on a separate channel and
// are drained by the caller. Order of the result slice isn't the same
// as input order — that's fine, since a similarity index doesn't care
// what order entries were added in.
func embedConcurrently(ctx context.Context, client *bedrock.Client, chunks []chunk.Chunk, workers int) []store.Entry {
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan chunk.Chunk)
	results := make(chan store.Entry)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range jobs {
				emb, err := client.Embed(ctx, c.Text)
				if err != nil {
					log.Printf("embedding %s#%d failed: %v", c.Source, c.Index, err)
					continue
				}
				results <- store.Entry{
					Source:    c.Source,
					ChunkIdx:  c.Index,
					Text:      c.Text,
					Embedding: emb,
				}
			}
		}()
	}

	go func() {
		for _, c := range chunks {
			jobs <- c
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var entries []store.Entry
	for e := range results {
		entries = append(entries, e)
	}
	return entries
}
