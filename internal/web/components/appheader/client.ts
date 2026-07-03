import { byID } from "../../../../web/src/dom";

interface AppHeaderDeps {
  loadCards: () => Promise<void>;
}

export function initAppHeader(deps: AppHeaderDeps): void {
  const reload = byID<HTMLButtonElement>("reload-cards-btn");
  if (reload) {
    reload.addEventListener("click", () => {
      void deps.loadCards();
    });
  }
}
