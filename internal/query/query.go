// Package query implements `gorag query`: embed the question, pull the
// top-k most similar chunks from the store, and ask Claude (via
// Bedrock) to answer using only that retrieved context.
package query

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"gorag/internal/bedrock"
	"gorag/internal/store"
)

// Run implements `gorag query`.
func Run(args []string) {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	storePath := fs.String("store", "store.json", "path to the vector store")
	question := fs.String("q", "", "question to ask")
	k := fs.Int("k", 4, "number of chunks to retrieve")
	region := fs.String("region", "us-east-1", "AWS region for Bedrock")
	model := fs.String("model", bedrock.DefaultGenerateModel, "Bedrock model or inference-profile ID for generation")
	fs.Parse(args)

	if *question == "" {
		log.Fatal("query: -q is required")
	}

	ctx := context.Background()
	client, err := bedrock.NewClient(ctx, *region)
	if err != nil {
		log.Fatalf("connecting to Bedrock: %v", err)
	}

	s, err := store.Load(*storePath)
	if err != nil {
		log.Fatalf("loading store: %v", err)
	}
	if len(s.Entries) == 0 {
		log.Fatal("store is empty — run `gorag ingest` first")
	}

	qEmb, err := client.Embed(ctx, *question)
	if err != nil {
		log.Fatalf("embedding question: %v", err)
	}

	hits := s.TopK(qEmb, *k)

	var sb strings.Builder
	sb.WriteString("Answer the question using only the context below. Cite sources by filename.\n\n")
	for _, h := range hits {
		fmt.Fprintf(&sb, "[%s] %s\n\n", h.Entry.Source, h.Entry.Text)
	}
	fmt.Fprintf(&sb, "Question: %s\n", *question)

	answer, err := client.Generate(ctx, *model, sb.String())
	if err != nil {
		log.Fatalf("generating answer: %v", err)
	}

	fmt.Println(answer)
	fmt.Println("\nSources:")
	for _, h := range hits {
		fmt.Printf("  %s (score %.3f)\n", h.Entry.Source, h.Score)
	}
}
