package diag

import (
	"strings"
	"testing"
)

func TestSplitAlwaysAnswers(t *testing.T) {
	pages := split(nil)
	if len(pages) != 1 {
		t.Fatalf("pages = %d, want 1", len(pages))
	}
	if len(pages[0]) != 0 {
		t.Errorf("page 0 = %v, want empty", pages[0])
	}
}

func TestSplitKeepsShortLogsWhole(t *testing.T) {
	lines := []string{"one", "two", "three"}
	pages := split(lines)

	if len(pages) != 1 {
		t.Fatalf("pages = %d, want 1", len(pages))
	}
	if len(pages[0]) != len(lines) {
		t.Errorf("page 0 has %d lines, want %d", len(pages[0]), len(lines))
	}
}

// No page may outgrow what the transport carries, and no line may be cut in half to manage it.
func TestSplitBoundsEachPageWithoutSplittingALine(t *testing.T) {
	line := strings.Repeat("x", 1000)
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = line
	}

	pages := split(lines)
	if len(pages) < 2 {
		t.Fatalf("pages = %d, want more than one", len(pages))
	}

	var seen int
	for i, p := range pages {
		size := 0
		for _, l := range p {
			if l != line {
				t.Fatalf("page %d: line was altered", i)
			}
			size += len(l) + 1
		}
		if size > logPage {
			t.Errorf("page %d is %d bytes, over the %d limit", i, size, logPage)
		}
		seen += len(p)
	}

	if seen != len(lines) {
		t.Errorf("kept %d lines of %d", seen, len(lines))
	}
}
