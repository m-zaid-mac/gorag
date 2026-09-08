package chunk

import "testing"

func TestSplitWordsBasic(t *testing.T) {
	text := "one two three four five six seven eight nine ten"
	chunks := SplitWords("doc.txt", text, 4, 1)

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0].Text != "one two three four" {
		t.Errorf("unexpected first chunk: %q", chunks[0].Text)
	}
	if chunks[0].Source != "doc.txt" {
		t.Errorf("expected source to propagate, got %q", chunks[0].Source)
	}
}

func TestSplitWordsOverlap(t *testing.T) {
	text := "one two three four five six seven eight"
	chunks := SplitWords("doc.txt", text, 4, 2)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	// With chunkSize=4, overlap=2, step=2: chunk0 = words[0:4], chunk1 = words[2:6]
	if chunks[1].Text != "three four five six" {
		t.Errorf("expected overlap to reuse words 'three four', got %q", chunks[1].Text)
	}
}

func TestSplitWordsEmpty(t *testing.T) {
	if chunks := SplitWords("empty.txt", "", 100, 10); chunks != nil {
		t.Errorf("expected nil for empty text, got %v", chunks)
	}
	if chunks := SplitWords("spaces.txt", "   \n\t  ", 100, 10); chunks != nil {
		t.Errorf("expected nil for whitespace-only text, got %v", chunks)
	}
}

func TestSplitWordsShortText(t *testing.T) {
	// Fewer words than chunkSize should still produce exactly one chunk.
	chunks := SplitWords("short.txt", "just three words", 200, 40)
	if len(chunks) != 1 {
		t.Fatalf("expected exactly 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Text != "just three words" {
		t.Errorf("unexpected chunk text: %q", chunks[0].Text)
	}
}
