package web

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/textarea"
)

type designerState struct {
	mu          sync.Mutex
	document    cardcomponent.Document
	library     []cardcomponent.LibraryItem
	lastApplied *cardcomponent.LibraryItem
}

func newDesignerState() *designerState {
	return &designerState{
		document: cardcomponent.DefaultDocument(),
		library:  seededLibrary(),
	}
}

func seededLibrary() []cardcomponent.LibraryItem {
	items := append([]cardcomponent.LibraryItem{}, background.Presets()...)
	items = append(items, border.Presets()...)
	items = append(items, textarea.Presets()...)
	return cloneLibrary(items)
}

func (s *designerState) snapshot() (cardcomponent.Document, []cardcomponent.LibraryItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return cloneValue(s.document), cloneLibrary(s.library)
}

func (s *designerState) reset() (cardcomponent.Document, []cardcomponent.LibraryItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.document = cardcomponent.DefaultDocument()
	s.lastApplied = nil
	return cloneValue(s.document), cloneLibrary(s.library)
}

func (s *designerState) apply(raw json.RawMessage) (cardcomponent.Document, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	document := cloneValue(s.document)
	normalized, item, err := applyGeneratedFragmentToDocument(raw, &document)
	if err != nil {
		return cardcomponent.Document{}, nil, err
	}
	s.document = document
	s.lastApplied = &item
	return cloneValue(s.document), normalized, nil
}

func (s *designerState) applyLibraryItem(id string) (cardcomponent.Document, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var item cardcomponent.LibraryItem
	for _, candidate := range s.library {
		if candidate.ID == id {
			item = cloneValue(candidate)
			break
		}
	}
	if item.ID == "" {
		return cardcomponent.Document{}, nil, fmt.Errorf("library item %q was not found", id)
	}
	raw, err := generatedRawFromLibraryItem(item)
	if err != nil {
		return cardcomponent.Document{}, nil, err
	}
	document := cloneValue(s.document)
	normalized, applied, err := applyGeneratedFragmentToDocument(raw, &document)
	if err != nil {
		return cardcomponent.Document{}, nil, err
	}
	s.document = document
	s.lastApplied = &applied
	return cloneValue(s.document), normalized, nil
}

func (s *designerState) saveLastApplied() (cardcomponent.LibraryItem, []cardcomponent.LibraryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastApplied == nil {
		return cardcomponent.LibraryItem{}, nil, fmt.Errorf("no applied fragment is available to save")
	}
	item := cloneValue(*s.lastApplied)
	item.Saved = true
	if strings.TrimSpace(item.ID) == "" || strings.HasPrefix(item.ID, "applied-") {
		item.ID = "saved-" + item.Target + "-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	if strings.TrimSpace(item.Name) == "" {
		item.Name = item.Description
	}
	for _, candidate := range s.library {
		if candidate.Target == item.Target && string(candidate.Fragment) == string(item.Fragment) {
			return cloneValue(candidate), cloneLibrary(s.library), nil
		}
	}
	s.library = append([]cardcomponent.LibraryItem{item}, s.library...)
	return cloneValue(item), cloneLibrary(s.library), nil
}

func (s *designerState) libraryForTarget(target string) []cardcomponent.LibraryItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	target = strings.TrimSpace(target)
	if target == "" {
		return cloneLibrary(s.library)
	}
	var out []cardcomponent.LibraryItem
	for _, item := range s.library {
		if item.Target == target {
			out = append(out, cloneValue(item))
		}
	}
	return out
}

func (s *designerState) currentFragment(target string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node := findNodeByType(s.document.Root, target)
	if node == nil || len(node.Fragment) == 0 {
		return "", ""
	}
	raw, err := json.MarshalIndent(struct {
		Target      string          `json:"target"`
		Description string          `json:"description"`
		Fragment    json.RawMessage `json:"fragment"`
	}{
		Target:      target,
		Description: "Current applied fragment",
		Fragment:    node.Fragment,
	}, "", "  ")
	if err != nil {
		return "", node.ID
	}
	return string(raw), node.ID
}

func findNodeByType(node cardcomponent.Node, target string) *cardcomponent.Node {
	if node.Type == target {
		return &node
	}
	for index := range node.Children {
		if match := findNodeByType(node.Children[index], target); match != nil {
			return match
		}
	}
	return nil
}

func generatedRawFromLibraryItem(item cardcomponent.LibraryItem) (json.RawMessage, error) {
	raw, err := json.Marshal(struct {
		Target      string          `json:"target"`
		Description string          `json:"description"`
		Fragment    json.RawMessage `json:"fragment"`
	}{
		Target:      item.Target,
		Description: item.Description,
		Fragment:    item.Fragment,
	})
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func cloneLibrary(items []cardcomponent.LibraryItem) []cardcomponent.LibraryItem {
	out := make([]cardcomponent.LibraryItem, len(items))
	for index, item := range items {
		out[index] = cloneValue(item)
	}
	return out
}

func cloneValue[T any](value T) T {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}
