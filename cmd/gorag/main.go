// Command gorag is a small concurrent RAG CLI: it ingests a folder of
// documents into a local vector store using AWS Bedrock (Titan
// Embeddings V2), then answers questions against that store using
// Claude via Bedrock.
package main

import (
	"fmt"
	"os"

	"gorag/internal/ingest"
	"gorag/internal/query"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "ingest":
		ingest.Run(os.Args[2:])
	case "query":
		query.Run(os.Args[2:])
	case "bench":
		ingest.RunBench(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`gorag - concurrent RAG CLI

Usage:
  gorag ingest -dir <folder> -out <store.json> [-workers N] [-region us-east-1]
  gorag query  -store <store.json> -q "<question>" [-k 4] [-region us-east-1] [-model us.anthropic...]
  gorag bench  -dir <folder> [-workers N] [-region us-east-1]

Requires AWS credentials in the environment (or ~/.aws/credentials) with
access to Bedrock's Titan Embeddings V2 and Claude 3.5 Sonnet models.`)
}
