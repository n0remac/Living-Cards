package appheader

import (
	"net/http"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/web/component"
)

const ComponentType = "app-header"

func Definition() component.Definition {
	return component.Definition{
		Type: ComponentType,
		Render: func(cards.ComponentInstance) *godom.Node {
			return View()
		},
		ClientInitializer: "initAppHeader",
		ContextFiles: []string{
			"internal/web/components/appheader/view.go",
			"internal/web/components/appheader/handlers.go",
			"internal/web/components/appheader/client.ts",
		},
	}
}

func RegisterRoutes(_ *http.ServeMux, _ component.Dependencies) {
}
