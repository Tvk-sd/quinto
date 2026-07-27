package tui

import (
	"strings"
	"testing"
)

func TestGoatIconDecodesToTheRightShape(t *testing.T) {
	icon := goatIcon()
	if len(icon) != 13 {
		t.Fatalf("goatIcon() returned %d rows, want 13", len(icon))
	}
	// The source image has a near-white background and margin at the very
	// top and bottom of the crop, so those rows render as blank — a decode
	// or chroma-key regression would instead either error (a visible
	// placeholder string) or paint the whole thing solid.
	any := false
	for _, row := range icon {
		if row != "" {
			any = true
		}
	}
	if !any {
		t.Error("goatIcon() rendered nothing at all — check the embed and the decode")
	}
}

func TestPickerBannerIncludesTheIconAndTheWordmark(t *testing.T) {
	out := pickerBanner()
	if !strings.Contains(out, "Q U I N T O") {
		t.Error("banner is missing the wordmark")
	}
	// Every icon row should show up somewhere in the banner, even though
	// each carries ANSI colour codes now rather than being plain text.
	for _, row := range goatIcon() {
		if row != "" && !strings.Contains(out, row) {
			t.Errorf("banner is missing an icon row: %q", row)
		}
	}
}
