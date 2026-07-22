package game

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/components/textinput"
)

func TestInstallableComponentRegistryMergePolicies(t *testing.T) {
	t.Parallel()

	t.Run("append layers receive independent ids", func(t *testing.T) {
		document := testInstallableDocument()
		component := Card{Name: "Text", State: map[string]any{
			"componentKind":     textarea.Kind,
			"componentDefaults": map[string]any{"content": "installed"},
		}}
		for index := 0; index < 2; index++ {
			node, policy, err := componentNodeFromCard(component, document)
			if err != nil {
				t.Fatalf("componentNodeFromCard() error = %v", err)
			}
			if policy != componentMergeAppend {
				t.Fatalf("policy = %q, want append", policy)
			}
			installComponentNode(&document, node, policy)
		}
		if findNodeByID(document.Root, "blank-controller-textarea") == nil || findNodeByID(document.Root, "blank-controller-textarea-2") == nil {
			t.Fatalf("children = %#v, want two independently identified textarea nodes", document.Root.Children)
		}
	})

	t.Run("singleton kinds replace their existing config", func(t *testing.T) {
		document := testInstallableDocument()
		component := Card{Name: "Border", State: map[string]any{
			"componentKind":     border.Kind,
			"componentDefaults": map[string]any{"border_width_px": 7, "border_color": "#123456", "border_style": "dashed"},
		}}
		node, policy, err := componentNodeFromCard(component, document)
		if err != nil {
			t.Fatalf("componentNodeFromCard() error = %v", err)
		}
		if policy != componentMergeReplace {
			t.Fatalf("policy = %q, want replace", policy)
		}
		installComponentNode(&document, node, policy)
		if len(document.Root.Children) != 1 || document.Root.Children[0].ID != "existing-border" {
			t.Fatalf("children = %#v, want the existing singleton node retained", document.Root.Children)
		}
		if !strings.Contains(string(document.Root.Children[0].Config), `"border_width_px":7`) {
			t.Fatalf("border config = %s, want replacement defaults", document.Root.Children[0].Config)
		}
	})
}

func TestInstallableComponentRegistryDefaultsAndValidation(t *testing.T) {
	t.Parallel()

	document := testInstallableDocument()
	node, policy, err := componentNodeFromCard(Card{Name: "Input", State: map[string]any{
		"componentKind": textinput.Kind,
		"componentId":   "password-input",
	}}, document)
	if err != nil {
		t.Fatalf("componentNodeFromCard() error = %v", err)
	}
	if policy != componentMergeAppend || node.ID != "password-input" {
		t.Fatalf("node = %#v policy = %q", node, policy)
	}
	var config textinput.Config
	if err := json.Unmarshal(node.Config, &config); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if config.FormID != textinput.DefaultConfig().FormID || config.Name != textinput.DefaultConfig().Name {
		t.Fatalf("config = %#v, want normalized defaults", config)
	}

	for name, component := range map[string]Card{
		"unknown kind": {
			Name:  "Unknown",
			State: map[string]any{"componentKind": "future-widget"},
		},
		"invalid config": {
			Name: "Bad input",
			State: map[string]any{
				"componentKind":     textinput.Kind,
				"componentDefaults": map[string]any{"input_type": "email"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := componentNodeFromCard(component, document); err == nil {
				t.Fatal("componentNodeFromCard() error = nil, want validation error")
			}
		})
	}
}

func TestInstallableComponentDefinitionsAreComplete(t *testing.T) {
	t.Parallel()

	for kind := range installableComponentDefinitions {
		definition, ok := installableComponentDefinitionForKind(kind)
		if !ok {
			t.Fatalf("installable component %q is missing its kind, defaults, normalization, or merge policy", kind)
		}
		if _, err := definition.normalizedConfig(nil); err != nil {
			t.Fatalf("installable component %q defaults are invalid: %v", kind, err)
		}
	}
}

func testInstallableDocument() cardcomponent.Document {
	return cardcomponent.Document{
		CardID: "blank-controller",
		Name:   "Blank Controller",
		Root: cardcomponent.Node{
			ID:            "blank-controller-root",
			ComponentKind: cardcomponent.Kind,
			Children: []cardcomponent.Node{{
				ID:            "existing-border",
				ComponentKind: border.Kind,
				Config:        json.RawMessage(`{"border_width_px":1,"border_radius_px":16,"border_color":"#ffffff","border_style":"solid","css":""}`),
			}},
		},
	}
}
