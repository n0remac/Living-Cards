package attack

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

func TestConfigNormalizationBoundsPropertiesAndInstallation(t *testing.T) {
	config := NormalizeConfig(Config{Label: " Claws ", Power: 0, Width: 4, BackgroundColor: " #400 ", AccentColor: " #faa "})
	if config.Label != "Claws" || config.BackgroundColor != "#400" || config.AccentColor != "#faa" {
		t.Fatalf("normalized = %#v", config)
	}
	raw, _ := json.Marshal(config)
	if _, issues := Definition().CanonicalizeConfig(card.RawConfig{Present: true, Value: raw}); len(issues) == 0 {
		t.Fatal("invalid power and width were accepted")
	}

	valid, issues := Definition().CanonicalizeConfig(card.RawConfig{Present: true, Value: json.RawMessage(`{"label":"Crystal Claws","power":3,"x":50,"y":64,"width":64,"background_color":"#400","accent_color":"#faa"}`)})
	if len(issues) != 0 {
		t.Fatalf("valid issues = %#v", issues)
	}
	value, present, propertyIssues := Definition().ReadProperty(valid, "power")
	if len(propertyIssues) != 0 || !present || value != schema.NumberValue(3) {
		t.Fatalf("power = %#v, %v, %#v", value, present, propertyIssues)
	}
	for _, control := range Definition().ControlIDs() {
		if control == "power" {
			t.Fatal("power has a player-facing control")
		}
	}
	install, ok := Definition().Install()
	if !ok || install.Policy != card.InstallAppend {
		t.Fatalf("install = %#v, %v", install, ok)
	}
}

func TestRenderLayerShowsLabelAndPower(t *testing.T) {
	body := RenderLayerWithContext("crystal-claws", Config{Label: "Crystal Claws", Power: 3, X: 50, Y: 64, Width: 64, BackgroundColor: "#400", AccentColor: "#faa"}, card.RenderContext{DOMIDPrefix: "library"}).Render()
	for _, marker := range []string{`id="library-crystal-claws-layer"`, `data-component-kind="attack"`, `data-attack-power="3"`, `Crystal Claws`, `+3`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q: %s", marker, body)
		}
	}
}
