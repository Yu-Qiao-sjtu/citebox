package repository

import (
	"testing"

	"github.com/xuzhougeng/citebox/internal/model"
)

func TestFigureTypePersistedAndFilterable(t *testing.T) {
	repo := newTestRepository(t)

	paper, err := repo.CreatePaper(PaperUpsertInput{
		Title:            "Graphical Abstract Paper",
		OriginalFilename: "ga-paper.pdf",
		StoredPDFName:    "ga-paper.pdf",
		FileSize:         128,
		ContentType:      "application/pdf",
		ExtractionStatus: "completed",
		Figures: []FigureUpsertInput{
			{Filename: "figure_plain.png", PageNumber: 1, FigureIndex: 1},
			{Filename: "figure_ga.png", PageNumber: 1, FigureIndex: 2, FigureType: model.FigureTypeGraphicalAbstract},
			{Filename: "figure_defaulted.png", PageNumber: 2, FigureIndex: 1, FigureType: model.FigureTypeGraphicalAbstract},
		},
	})
	if err != nil {
		t.Fatalf("CreatePaper() error = %v", err)
	}
	if len(paper.Figures) != 3 {
		t.Fatalf("CreatePaper() figures = %d, want 3", len(paper.Figures))
	}
	if paper.Figures[0].FigureType != model.FigureTypeFigure {
		t.Fatalf("default figure_type = %q, want %q", paper.Figures[0].FigureType, model.FigureTypeFigure)
	}
	if paper.Figures[1].FigureType != model.FigureTypeGraphicalAbstract {
		t.Fatalf("figure_type = %q, want %q", paper.Figures[1].FigureType, model.FigureTypeGraphicalAbstract)
	}

	figures, total, err := repo.ListFigures(model.FigureFilter{PaperID: &paper.ID})
	if err != nil {
		t.Fatalf("ListFigures() error = %v", err)
	}
	if total != 3 {
		t.Fatalf("ListFigures() total = %d, want 3", total)
	}

	figures, total, err = repo.ListFigures(model.FigureFilter{PaperID: &paper.ID, FigureType: model.FigureTypeGraphicalAbstract})
	if err != nil {
		t.Fatalf("ListFigures(figure_type) error = %v", err)
	}
	if total != 2 || len(figures) != 2 {
		t.Fatalf("ListFigures(figure_type) total=%d len=%d, want 2/2", total, len(figures))
	}
	for _, figure := range figures {
		if figure.FigureType != model.FigureTypeGraphicalAbstract {
			t.Fatalf("ListFigures(figure_type) figure_type = %q, want %q", figure.FigureType, model.FigureTypeGraphicalAbstract)
		}
	}
}

func TestNormalizeFigureType(t *testing.T) {
	cases := map[string]string{
		"":                   model.FigureTypeFigure,
		"figure":             model.FigureTypeFigure,
		"graphical_abstract": model.FigureTypeGraphicalAbstract,
		"something_else":     model.FigureTypeFigure,
	}
	for input, want := range cases {
		if got := model.NormalizeFigureType(input); got != want {
			t.Fatalf("NormalizeFigureType(%q) = %q, want %q", input, got, want)
		}
	}
}
