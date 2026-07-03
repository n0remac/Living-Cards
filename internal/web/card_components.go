package web

import (
	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/textarea"
)

func cardComponentRegistry() *cardcomponent.Registry {
	return cardcomponent.MustNewRegistry(
		background.Definition(),
		border.Definition(),
		textarea.Definition(),
	)
}
