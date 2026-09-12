package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuzhougeng/citebox/internal/model"
)

func TestManualExtractFiguresStoresFigureType(t *testing.T) {
	svc, repo, cfg := newTestService(t)
	paper := createTestPaper(t, repo)

	if err := os.WriteFile(filepath.Join(cfg.PapersDir(), paper.StoredPDFName), []byte("%PDF-1.4 test"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdf) error = %v", err)
	}

	updated, addedCount, err := svc.ManualExtractFigures(paper.ID, ManualExtractParams{
		Regions: []model.ManualExtractionRegion{
			{
				PageNumber: 1,
				X:          0.1,
				Y:          0.2,
				Width:      0.3,
				Height:     0.4,
				ImageData:  testPNGDataURL(t, 24, 18),
				Caption:    "Graphical abstract",
				FigureType: model.FigureTypeGraphicalAbstract,
			},
			{
				PageNumber: 1,
				X:          0.5,
				Y:          0.2,
				Width:      0.3,
				Height:     0.4,
				ImageData:  testPNGDataURL(t, 24, 18),
				FigureType: "not-a-type",
			},
		},
	})
	if err != nil {
		t.Fatalf("ManualExtractFigures() error = %v", err)
	}
	if addedCount != 2 {
		t.Fatalf("ManualExtractFigures() addedCount = %d, want 2", addedCount)
	}

	manual := 0
	for i := range updated.Figures {
		figure := updated.Figures[i]
		if figure.Source != "manual" {
			continue
		}
		manual++
		if figure.Caption == "Graphical abstract" && figure.FigureType != model.FigureTypeGraphicalAbstract {
			t.Fatalf("GA figure figure_type = %q, want %q", figure.FigureType, model.FigureTypeGraphicalAbstract)
		}
		if figure.Caption == "" && figure.FigureType != model.FigureTypeFigure {
			t.Fatalf("invalid type should normalize to figure, got %q", figure.FigureType)
		}
	}
	if manual != 2 {
		t.Fatalf("manual figures = %d, want 2", manual)
	}
}
