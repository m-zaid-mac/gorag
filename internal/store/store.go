// Package store implements a small, dependency-free vector store: it
// persists embedded chunks as JSON and answers nearest-neighbour
// queries with brute-force cosine similarity. That's the right trade-off
// for a few thousand chunks, and it keeps the whole retrieval mechanism
// visible in ~80 lines instead of hidden inside a library.
package store

import (
	"encoding/json"
	"math"
	"os"
	"sort"
)

// Entry is one embedded chunk persisted to disk.
type Entry struct {
	Source    string    `json:"source"`
	ChunkIdx  int       `json:"chunk_idx"`
	Text      string    `json:"text"`
	Embedding []float32 `json:"embedding"`
}

// Store is an in-memory, JSON-backed collection of embedded chunks.
type Store struct {
	Entries []Entry `json:"entries"`
}

// Load reads a Store from disk. A missing file yields an empty Store
// so `ingest` can be run for the first time without a pre-existing file.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Store{}, nil
	}
	if err != nil {
		return nil, err
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save writes the Store to disk as indented JSON.
func (s *Store) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Add appends an entry to the store.
func (s *Store) Add(e Entry) {
	s.Entries = append(s.Entries, e)
}

// Result is a scored search hit.
type Result struct {
	Entry Entry
	Score float32
}

// TopK returns the k entries most similar to query, ranked by cosine
// similarity (highest first).
func (s *Store) TopK(query []float32, k int) []Result {
	results := make([]Result, 0, len(s.Entries))
	for _, e := range s.Entries {
		results = append(results, Result{Entry: e, Score: cosine(query, e.Embedding)})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if k > len(results) {
		k = len(results)
	}
	if k < 0 {
		k = 0
	}
	return results[:k]
}

func cosine(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, magA, magB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		magA += float64(a[i]) * float64(a[i])
		magB += float64(b[i]) * float64(b[i])
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(magA) * math.Sqrt(magB)))
}
