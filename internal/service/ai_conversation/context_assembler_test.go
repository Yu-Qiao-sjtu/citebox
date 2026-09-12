package ai_conversation

import (
	"strings"
	"testing"

	"github.com/xuzhougeng/citebox/internal/model"
	"github.com/xuzhougeng/citebox/internal/repository"
)

func TestAssembleForTurnIncludesFullAbstractAndTruncationHint(t *testing.T) {
	svc, libRepo, _ := newServiceForTest(t)

	paperID := mustInsertPaperForTest(t, libRepo, "Spatial Omni Paper", "10.1/spatial")
	abstract := strings.Repeat("摘要句子。", 400) // 2000 runes, well above the old 800 cap
	longTail := strings.Repeat("RESULTS AND METHODS tail ", 1500)
	pdfText := "INTRODUCTION head " + longTail
	if _, err := libRepo.DB().Exec(
		`UPDATE papers SET abstract_text = ?, pdf_text = ? WHERE id = ?`,
		abstract, pdfText, paperID); err != nil {
		t.Fatalf("update paper text: %v", err)
	}

	conv := repository.AIConversation{StrictEvidence: false}
	pinned := []repository.AIPinnedPaper{{PaperID: paperID, Title: "Spatial Omni Paper"}}

	asm, err := svc.assembleForTurn(conv, pinned, nil, "这篇论文的图注说了什么？", "", model.AISettings{})
	if err != nil {
		t.Fatalf("assembleForTurn: %v", err)
	}

	if !strings.Contains(asm.userPrompt, abstract) {
		t.Fatalf("pinned block should include the full abstract")
	}
	if !strings.Contains(asm.userPrompt, "正文开头，全文更长") {
		t.Fatalf("pinned block should tell the model the body is truncated: %s", asm.userPrompt)
	}
	if !strings.Contains(asm.userPrompt, "文献检索工具") {
		t.Fatalf("pinned block should point the model at retrieval tools")
	}
}

func TestAssembleForTurnKeepsBodyWholeUnderCap(t *testing.T) {
	svc, libRepo, _ := newServiceForTest(t)

	paperID := mustInsertPaperForTest(t, libRepo, "Short Body Paper", "10.1/short")
	pdfText := strings.Repeat("short body ", 200) // 2400 runes, under the cap
	if _, err := libRepo.DB().Exec(
		`UPDATE papers SET pdf_text = ? WHERE id = ?`, pdfText, paperID); err != nil {
		t.Fatalf("update paper text: %v", err)
	}

	conv := repository.AIConversation{StrictEvidence: false}
	pinned := []repository.AIPinnedPaper{{PaperID: paperID, Title: "Short Body Paper"}}

	asm, err := svc.assembleForTurn(conv, pinned, nil, "总结方法", "", model.AISettings{})
	if err != nil {
		t.Fatalf("assembleForTurn: %v", err)
	}

	if strings.Contains(asm.userPrompt, "正文开头，全文更长") {
		t.Fatalf("short body should not carry the truncation hint")
	}
	if !strings.Contains(asm.userPrompt, strings.TrimRight(pdfText, " ")) {
		t.Fatalf("pinned block should include the whole body under the cap")
	}
}
