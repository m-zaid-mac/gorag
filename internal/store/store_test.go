package store

import (
	"path/filepath"
	"testing"
)

func TestCosineIdentical(t *testing.T) {
	v := []float32{1, 2, 3}
	if got := cosine(v, v); got < 0.999 || got > 1.001 {
		t.Errorf("expected cosine(v, v) ~= 1, got %f", got)
	}
}

func TestCosineOrthogonal(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{0, 1}
	if got := cosine(a, b); got != 0 {
		t.Errorf("expected orthogonal vectors to score 0, got %f", got)
	}
}

func TestCosineMismatchedLength(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0}
	if got := cosine(a, b); got != 0 {
		t.Errorf("expected mismatched-length vectors to score 0, got %f", got)
	}
}

func TestTopK(t *testing.T) {
	s := &Store{}
	s.Add(Entry{Source: "a", Text: "close", Embedding: []float32{1, 0}})
	s.Add(Entry{Source: "b", Text: "far", Embedding: []float32{0, 1}})
	s.Add(Entry{Source: "c", Text: "closer", Embedding: []float32{0.9, 0.1}})

	results := s.TopK([]float32{1, 0}, 2)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Entry.Text != "close" {
		t.Errorf("expected 'close' to rank first, got %q", results[0].Entry.Text)
	}
	if results[1].Entry.Text != "closer" {
		t.Errorf("expected 'closer' to rank second, got %q", results[1].Entry.Text)
	}
}

func TestTopKMoreThanAvailable(t *testing.T) {
	s := &Store{}
	s.Add(Entry{Source: "a", Embedding: []float32{1, 0}})
	if results := s.TopK([]float32{1, 0}, 10); len(results) != 1 {
		t.Errorf("expected 1 result when k exceeds store size, got %d", len(results))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	s := &Store{}
	s.Add(Entry{Source: "doc.txt", ChunkIdx: 0, Text: "hello", Embedding: []float32{0.1, 0.2}})
	if err := s.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Entries) != 1 || loaded.Entries[0].Source != "doc.txt" {
		t.Errorf("loaded store doesn't match saved store: %+v", loaded.Entries)
	}
}

func TestLoadMissingFile(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(s.Entries) != 0 {
		t.Errorf("expected empty store for missing file, got %d entries", len(s.Entries))
	}
}
