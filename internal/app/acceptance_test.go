package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildAcceptanceReportPassed(t *testing.T) {
	dir := t.TempDir()
	impl := filepath.Join(dir, "impl.md")
	validation := filepath.Join(dir, "validation.md")
	writeTestFile(t, impl, "# Report\n\nGoal status: complete\nChanged files: app.js")
	writeTestFile(t, validation, "# Validation\n\nGoal status: complete\nValidation passed.\nRisks: none known.\nNext: ship it.")
	task := &Task{Status: StatusFinished, Trees: []*Tree{
		{Status: StatusFinished, FruitPath: impl},
		{Status: StatusFinished, IsValidation: true, FruitPath: validation},
	}}

	report := buildAcceptanceReport(task)
	if report.Status != "passed" {
		t.Fatalf("status = %q, want passed; report=%#v", report.Status, report)
	}
	if report.Score < 90 {
		t.Fatalf("score = %d, want >= 90", report.Score)
	}
	if len(report.Checklist) != 4 {
		t.Fatalf("checklist length = %d", len(report.Checklist))
	}
}

func TestBuildAcceptanceReportDetectsBlockedValidation(t *testing.T) {
	dir := t.TempDir()
	validation := filepath.Join(dir, "validation.md")
	writeTestFile(t, validation, "# Validation\n\nGoal status: blocked\nValidation failed: tests failed.\nRisk: data loss possible.")
	task := &Task{Status: StatusFinished, UpdatedAt: time.Now(), Trees: []*Tree{{Status: StatusFinished, IsValidation: true, FruitPath: validation}}}

	report := buildAcceptanceReport(task)
	if report.Status != "blocked" {
		t.Fatalf("status = %q, want blocked; report=%#v", report.Status, report)
	}
	if report.Score >= 70 {
		t.Fatalf("score = %d, want below 70", report.Score)
	}
}
