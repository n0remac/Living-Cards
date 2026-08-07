package creature

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

func TestConfigNormalizationBoundsAndProperties(t *testing.T) {
	config := NormalizeConfig(Config{Health: 7, MaxHealth: 6, Width: 2, HealthColor: " #fff ", BackgroundColor: " #000 ", AccentColor: " red "})
	if config.HealthColor != "#fff" || config.BackgroundColor != "#000" || config.AccentColor != "red" {
		t.Fatalf("normalized = %#v", config)
	}
	if issues := configIssues(config); len(issues) == 0 {
		t.Fatal("invalid health and width were accepted")
	}

	definition := Definition()
	raw, issues := definition.CanonicalizeConfig(card.RawConfig{Present: true, Value: json.RawMessage(`{"health":4,"max_health":6,"x":50,"y":80,"width":72,"health_color":"#4ade80","background_color":"#0f172a","accent_color":"#bbf7d0"}`)})
	if len(issues) != 0 {
		t.Fatalf("canonical issues = %#v", issues)
	}
	for _, property := range []string{"health", "max_health"} {
		value, present, propertyIssues := definition.ReadProperty(raw, property)
		if len(propertyIssues) != 0 || !present || value.Kind != schema.PropertyNumber {
			t.Fatalf("%s property = %#v, %v, %#v", property, value, present, propertyIssues)
		}
	}
	updated, writable, writeIssues := definition.WriteProperty(raw, "health", schema.NumberValue(2))
	if len(writeIssues) != 0 || !writable || !strings.Contains(string(updated), `"health":2`) {
		t.Fatalf("write = %s, %v, %#v", updated, writable, writeIssues)
	}
	if _, writable, _ := definition.WriteProperty(raw, "max_health", schema.NumberValue(8)); writable {
		t.Fatal("max_health unexpectedly writable")
	}
	for _, control := range definition.ControlIDs() {
		if control == "health" || control == "max_health" {
			t.Fatalf("combat value has control %q", control)
		}
	}
	if _, installable := definition.Install(); installable {
		t.Fatal("creature is installable")
	}
}

func TestRenderLayerShowsHealth(t *testing.T) {
	body := RenderLayerWithContext("guardian-health", Config{Health: 3, MaxHealth: 6, X: 50, Y: 80, Width: 72, HealthColor: "#4ade80", BackgroundColor: "#0f172a", AccentColor: "#bbf7d0"}, card.RenderContext{DOMIDPrefix: "world"}).Render()
	for _, marker := range []string{`id="world-guardian-health-layer"`, `data-component-kind="creature"`, `data-health="3"`, `3 / 6`, `width: 50%`, `role="meter"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q: %s", marker, body)
		}
	}
}

func configIssues(config Config) []schema.Issue {
	raw, _ := json.Marshal(config)
	_, issues := Definition().CanonicalizeConfig(card.RawConfig{Present: true, Value: raw})
	return issues
}
