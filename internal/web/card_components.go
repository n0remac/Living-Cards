package web

import (
	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/textarea"
)

func cardComponentRegistry() *cardcomponent.Registry {
	return cardcomponent.MustNewRegistry(
		background.Definition(),
		border.Definition(),
		textarea.Definition(),
		shape.Definition(),
		imagecomponent.Definition(),
	)
}
