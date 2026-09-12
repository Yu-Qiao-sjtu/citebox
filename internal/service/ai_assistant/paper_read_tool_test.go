package ai_assistant

import (
	"strings"
	"testing"

	"github.com/xuzhougeng/citebox/internal/model"
)

func TestFallbackPaperEvidencePrefersBodyOverAbstract(t *testing.T) {
	paper := model.Paper{
		ID:           1,
		Title:        "Paper",
		AbstractText: "Abstract only text.",
		PDFText:      "INTRODUCTION body starts here and keeps going with real content.",
	}

	matches := fallbackPaperEvidenceMatches(paper, 1)
	if len(matches) != 1 {
		t.Fatalf("matches = %d, want 1", len(matches))
	}
	m := matches[0]
	if !strings.Contains(m.Snippet.Text, "INTRODUCTION body") {
		t.Fatalf("fallback should prefer PDFText, got %q", m.Snippet.Text)
	}
	if m.Location == "全文" || m.Snippet.Section == "全文" {
		t.Fatalf("fallback must not label the excerpt as 全文, got %q", m.Location)
	}
	if !strings.Contains(m.Location, "关键词未命中") {
		t.Fatalf("fallback should disclose the keyword miss, got %q", m.Location)
	}
}

func TestFallbackPaperEvidenceLabelsAbstractOnlyPapers(t *testing.T) {
	paper := model.Paper{
		ID:           2,
		Title:        "Abstract Only",
		AbstractText: "Only the abstract is available.",
	}

	matches := fallbackPaperEvidenceMatches(paper, 1)
	if len(matches) != 1 {
		t.Fatalf("matches = %d, want 1", len(matches))
	}
	if !strings.Contains(matches[0].Location, "摘要") {
		t.Fatalf("abstract-only fallback should be labeled 摘要, got %q", matches[0].Location)
	}
}

func TestFallbackPaperEvidenceReturnsNilWithoutText(t *testing.T) {
	if got := fallbackPaperEvidenceMatches(model.Paper{ID: 3, Title: "Empty"}, 1); got != nil {
		t.Fatalf("expected nil for paper without text, got %+v", got)
	}
}
