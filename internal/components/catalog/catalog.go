// Package catalog is the only production composition root for component
// definitions. Adding a component to the backend requires one entry here.
package catalog

import (
	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	"github.com/n0remac/Living-Card/internal/components/button"
	"github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/components/text"
	"github.com/n0remac/Living-Card/internal/components/textinput"
)

func New() (*card.Registry, error) {
	return card.NewRegistry(
		card.RootDefinition(),
		background.Definition(),
		border.Definition(),
		text.Definition(),
		shape.Definition(),
		imagecomponent.Definition(),
		slider.Definition(),
		textinput.Definition(),
		button.Definition(),
	)
}

func MustNew() *card.Registry {
	registry, err := New()
	if err != nil {
		panic(err)
	}
	return registry
}
