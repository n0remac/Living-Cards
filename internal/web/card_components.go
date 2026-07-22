package web

import (
	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	"github.com/n0remac/Living-Card/internal/components/button"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/components/textinput"
)

func cardComponentRegistry() *cardcomponent.Registry {
	return cardcomponent.MustNewRegistry(
		background.Definition(),
		border.Definition(),
		textarea.Definition(),
		shape.Definition(),
		imagecomponent.Definition(),
		slider.Definition(),
		textinput.Definition(),
		button.Definition(),
	)
}
