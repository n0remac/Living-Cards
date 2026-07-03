package profile

import (
	"context"
	"testing"
)

func TestParseFactUpdatesAcceptsValidFacts(t *testing.T) {
	t.Parallel()

	updates, err := ParseFactUpdates(`{"facts":[{"key":"preferences","value":"likes concise technical explanations","confidence":0.86,"evidence":"I prefer short technical explanations."}]}`)
	if err != nil {
		t.Fatalf("ParseFactUpdates() error = %v", err)
	}
	if len(updates) != 1 {
		t.Fatalf("len(updates) = %d, want 1", len(updates))
	}
	if updates[0].Key != "preferences" {
		t.Fatalf("Key = %q", updates[0].Key)
	}
}

func TestParseFactUpdatesRejectsMalformedOutput(t *testing.T) {
	t.Parallel()

	if _, err := ParseFactUpdates(`not json`); err == nil {
		t.Fatal("ParseFactUpdates() error = nil, want error")
	}
	if _, err := ParseFactUpdates(`{"facts":[{"key":"unknown","value":"x","confidence":0.9,"evidence":"x"}]}`); err == nil {
		t.Fatal("ParseFactUpdates() invalid key error = nil, want error")
	}
}

func TestProfileSummaryUsesAcceptedFactsOnly(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	ctx := context.Background()
	_, err := store.ApplyFactUpdates(ctx, "tester", []FactUpdate{
		{
			Key:        "preferences",
			Value:      "likes concise replies",
			Confidence: 0.9,
			Evidence:   "I like concise replies.",
		},
		{
			Key:        "goals",
			Value:      "is exploring an app idea",
			Confidence: 0.5,
			Evidence:   "I am exploring an app idea.",
		},
	})
	if err != nil {
		t.Fatalf("ApplyFactUpdates() error = %v", err)
	}

	profile, err := store.Get(ctx, "tester")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if profile.ProfileSummary != "- preferences: likes concise replies" {
		t.Fatalf("ProfileSummary = %q", profile.ProfileSummary)
	}
	if len(profile.Facts) != 2 {
		t.Fatalf("len(Facts) = %d, want 2", len(profile.Facts))
	}
}

func TestResetClearsActiveFactsAndSummary(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	ctx := context.Background()
	_, err := store.ApplyFactUpdates(ctx, "tester", []FactUpdate{{
		Key:        "preferences",
		Value:      "likes concise replies",
		Confidence: 0.9,
		Evidence:   "I like concise replies.",
	}})
	if err != nil {
		t.Fatalf("ApplyFactUpdates() error = %v", err)
	}
	if err := store.Reset(ctx, "tester"); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	profile, err := store.Get(ctx, "tester")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if profile.ProfileSummary != "" || len(profile.Facts) != 0 {
		t.Fatalf("profile after reset = %#v", profile)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir() + "/profile.db")
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}
