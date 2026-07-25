package main

import (
	"strings"
	"testing"
)

func TestLoadGameCardJobsIncludesEveryCurrentCard(t *testing.T) {
	root, err := projectRoot()
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := loadGameCardJobs(root, 42)
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := []string{
		"archive-operator-log",
		"archive-password-input-component",
		"archive-submit-button-component",
		"archive-terminal",
		"burnt-ribbon",
		"copper-washer",
		"fuse-box-note",
		"glass-fuse",
		"border-component",
		"generator-panel",
		"numbered-gauge",
		"slider-component",
		"bent-iron-key",
		"faded-photograph",
		"inventory-label",
		"rusted-cell-door",
		"sleeping-switch",
	}
	if len(jobs) != len(wantIDs) {
		t.Fatalf("loaded %d game cards, want %d", len(jobs), len(wantIDs))
	}
	for index, wantID := range wantIDs {
		job := jobs[index]
		if job.cardID != wantID || job.baseName != wantID {
			t.Fatalf("job %d = %#v, want card %q", index, job, wantID)
		}
		if job.deckID == "" || job.cardName == "" {
			t.Fatalf("job %q is missing deck metadata", job.cardID)
		}
		if !strings.Contains(job.prompt.Text, job.cardName) {
			t.Fatalf("prompt for %q does not include its card name", job.cardID)
		}
	}
}

func TestGameCardModeRejectsGenericPromptFlags(t *testing.T) {
	_, err := buildJobs(options{gameCards: true, count: 2, seed: 42})
	if err == nil {
		t.Fatal("game card mode accepted -count")
	}
}
