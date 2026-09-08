// Package chunk splits raw document text into overlapping windows
// suitable for embedding. Overlap keeps context from being lost right
// at a chunk boundary (a sentence that straddles two chunks still
// appears whole in at least one of them).
package chunk

import "strings"

// Chunk is one window of text taken from a source document.
type Chunk struct {
	Source string
	Index  int
	Text   string
}

// SplitWords splits text into chunks of roughly chunkSize words, with
// overlap words shared between consecutive chunks.
func SplitWords(source, text string, chunkSize, overlap int) []Chunk {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = 200
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 2
	}
	if overlap < 0 {
		overlap = 0
	}

	var chunks []Chunk
	step := chunkSize - overlap
	idx := 0
	for start := 0; start < len(words); start += step {
		end := start + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, Chunk{
			Source: source,
			Index:  idx,
			Text:   strings.Join(words[start:end], " "),
		})
		idx++
		if end == len(words) {
			break
		}
	}
	return chunks
}
